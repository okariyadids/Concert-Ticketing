package accounting

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Client struct {
	httpClient *http.Client
	baseURL    string
}

func NewClient(baseURL string, timeout time.Duration) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: timeout},
		baseURL:    baseURL,
	}
}

type sendPayload struct {
	TransactionID int64  `json:"transaction_id"`
	TicketID      int64  `json:"ticket_id"`
	BuyerName     string `json:"buyer_name"`
}

func (client *Client) Send(context context.Context, entry OutboxEntry) error {
	body, err := json.Marshal(sendPayload{
		TransactionID: entry.TransactionID,
		TicketID:      entry.TicketID,
		BuyerName:     entry.BuyerName,
	})
	if err != nil {
		return err
	}

	httpRequest, err := http.NewRequestWithContext(context, http.MethodPost, client.baseURL+"/transaction", bytes.NewReader(body))
	if err != nil {
		return err
	}
	httpRequest.Header.Set("Content-Type", "application/json")

	httpResponse, err := client.httpClient.Do(httpRequest)
	if err != nil {
		return err
	}
	defer httpResponse.Body.Close()

	if httpResponse.StatusCode < 200 || httpResponse.StatusCode >= 300 {
		return fmt.Errorf("accounting service returned status %d", httpResponse.StatusCode)
	}
	return nil
}
