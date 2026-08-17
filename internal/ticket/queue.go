package ticket

import "context"

type purchaseJob struct {
	context   context.Context
	ticketID  int64
	buyerName string
	result    chan purchaseResult
}

type purchaseResult struct {
	transactionID int64
	err           error
}

type Queue struct {
	jobs    chan purchaseJob
	service *Service
}

func NewQueue(service *Service, workerCount int, queueSize int) *Queue {
	queue := &Queue{
		jobs:    make(chan purchaseJob, queueSize),
		service: service,
	}

	for workerIndex := 0; workerIndex < workerCount; workerIndex++ {
		go queue.worker()
	}

	return queue
}

func (queue *Queue) worker() {
	for job := range queue.jobs {
		transactionID, err := queue.service.Purchase(job.context, job.ticketID, job.buyerName)
		job.result <- purchaseResult{transactionID: transactionID, err: err}
	}
}

func (queue *Queue) Enqueue(context context.Context, ticketID int64, buyerName string) (int64, error) {
	job := purchaseJob{
		context:   context,
		ticketID:  ticketID,
		buyerName: buyerName,
		result:    make(chan purchaseResult, 1),
	}

	queue.jobs <- job
	result := <-job.result
	return result.transactionID, result.err
}
