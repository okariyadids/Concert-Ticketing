package accounting

import (
	"context"
	"log"
	"time"
)

type Worker struct {
	repository   *Repository
	client       *Client
	pollInterval time.Duration
	batchSize    int
	maxAttempts  int
	baseBackoff  time.Duration
	maxBackoff   time.Duration
}

func NewWorker(repository *Repository, client *Client, pollInterval time.Duration, batchSize, maxAttempts int, baseBackoff, maxBackoff time.Duration) *Worker {
	return &Worker{
		repository:   repository,
		client:       client,
		pollInterval: pollInterval,
		batchSize:    batchSize,
		maxAttempts:  maxAttempts,
		baseBackoff:  baseBackoff,
		maxBackoff:   maxBackoff,
	}
}

func (worker *Worker) Start(context context.Context) {
	ticker := time.NewTicker(worker.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-context.Done():
			return
		case <-ticker.C:
			worker.processDue(context)
		}
	}
}

func (worker *Worker) processDue(context context.Context) {
	entries, err := worker.repository.FetchDue(context, worker.batchSize)
	if err != nil {
		log.Printf("accounting worker: failed to fetch due entries: %v", err)
		return
	}

	for _, entry := range entries {
		err := worker.client.Send(context, entry)
		if err == nil {
			if err := worker.repository.MarkSent(context, entry.ID); err != nil {
				log.Printf("accounting worker: failed to mark outbox %d as sent: %v", entry.ID, err)
			}
			continue
		}

		attempt := entry.AttemptCount + 1
		if attempt >= worker.maxAttempts {
			log.Printf("accounting worker: outbox %d gave up after %d attempts: %v", entry.ID, attempt, err)
			if markErr := worker.repository.MarkDead(context, entry.ID, attempt, err.Error()); markErr != nil {
				log.Printf("accounting worker: failed to mark outbox %d as dead: %v", entry.ID, markErr)
			}
			continue
		}

		nextAttemptAt := time.Now().Add(backoffDuration(attempt, worker.baseBackoff, worker.maxBackoff))
		if markErr := worker.repository.MarkForRetry(context, entry.ID, attempt, err.Error(), nextAttemptAt); markErr != nil {
			log.Printf("accounting worker: failed to schedule retry for outbox %d: %v", entry.ID, markErr)
		}
	}
}

func backoffDuration(attempt int, baseDuration, maxDuration time.Duration) time.Duration {
	duration := baseDuration
	for backoffStep := 1; backoffStep < attempt; backoffStep++ {
		duration *= 2
		if duration >= maxDuration {
			return maxDuration
		}
	}
	return duration
}
