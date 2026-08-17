package payment

import (
	"encoding/json"
	"net/http"
)

type Handler struct {
	repository *Repository
}

func NewHandler(repository *Repository) *Handler {
	return &Handler{repository: repository}
}

type webhookRequest struct {
	ExternalPaymentID string  `json:"external_payment_id"`
	TransactionID     int64   `json:"transaction_id"`
	Amount            float64 `json:"amount"`
	Status            string  `json:"status"`
}

type webhookResponse struct {
	Message string `json:"message"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func (handler *Handler) HandleWebhook(responseWriter http.ResponseWriter, request *http.Request) {
	var req webhookRequest
	if err := json.NewDecoder(request.Body).Decode(&req); err != nil {
		writeJSON(responseWriter, http.StatusBadRequest, errorResponse{Error: "invalid request body"})
		return
	}
	if req.ExternalPaymentID == "" {
		writeJSON(responseWriter, http.StatusBadRequest, errorResponse{Error: "external_payment_id is required"})
		return
	}

	inserted, err := handler.repository.SaveIdempotent(request.Context(), Payment{
		TransactionID:     req.TransactionID,
		ExternalPaymentID: req.ExternalPaymentID,
		Amount:            req.Amount,
		Status:            req.Status,
	})
	if err != nil {
		writeJSON(responseWriter, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
		return
	}

	message := "payment saved"
	if !inserted {
		message = "duplicate payment ignored"
	}
	writeJSON(responseWriter, http.StatusOK, webhookResponse{Message: message})
}

func writeJSON(responseWriter http.ResponseWriter, status int, body any) {
	responseWriter.Header().Set("Content-Type", "application/json")
	responseWriter.WriteHeader(status)
	json.NewEncoder(responseWriter).Encode(body)
}
