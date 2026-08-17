package ticket

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
)

// Handler jadi jembatan antara HTTP request/response dan Queue.
type Handler struct {
	queue *Queue
}

func NewHandler(queue *Queue) *Handler {
	return &Handler{queue: queue}
}

type purchaseRequest struct {
	BuyerName string `json:"buyer_name"`
}

type purchaseResponse struct {
	TransactionID int64  `json:"transaction_id"`
	Message       string `json:"message"`
}

type errorResponse struct {
	Error string `json:"error"`
}

// HandlePurchase menangani POST /tickets/{id}/purchase
func (handler *Handler) HandlePurchase(responseWriter http.ResponseWriter, request *http.Request) {
	ticketID, err := strconv.ParseInt(request.PathValue("id"), 10, 64)
	if err != nil {
		writeJSON(responseWriter, http.StatusBadRequest, errorResponse{Error: "invalid ticket id"})
		return
	}

	var req purchaseRequest
	if err := json.NewDecoder(request.Body).Decode(&req); err != nil {
		writeJSON(responseWriter, http.StatusBadRequest, errorResponse{Error: "invalid request body"})
		return
	}
	if req.BuyerName == "" {
		writeJSON(responseWriter, http.StatusBadRequest, errorResponse{Error: "buyer_name is required"})
		return
	}

	transactionID, err := handler.queue.Enqueue(request.Context(), ticketID, req.BuyerName)
	if err != nil {
		if errors.Is(err, ErrSoldOut) {
			writeJSON(responseWriter, http.StatusConflict, errorResponse{Error: "ticket sold out"})
			return
		}
		writeJSON(responseWriter, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
		return
	}

	writeJSON(responseWriter, http.StatusOK, purchaseResponse{
		TransactionID: transactionID,
		Message:       "purchase success",
	})
}

func writeJSON(responseWriter http.ResponseWriter, status int, body any) {
	responseWriter.Header().Set("Content-Type", "application/json")
	responseWriter.WriteHeader(status)
	json.NewEncoder(responseWriter).Encode(body)
}
