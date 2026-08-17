package ticket

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
)

// Handler jadi jembatan antara HTTP request/response dan Service.
type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
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
func (h *Handler) HandlePurchase(w http.ResponseWriter, r *http.Request) {
	ticketID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid ticket id"})
		return
	}

	var req purchaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid request body"})
		return
	}
	if req.BuyerName == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "buyer_name is required"})
		return
	}

	transactionID, err := h.service.Purchase(r.Context(), ticketID, req.BuyerName)
	if err != nil {
		if errors.Is(err, ErrSoldOut) {
			writeJSON(w, http.StatusConflict, errorResponse{Error: "ticket sold out"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
		return
	}

	writeJSON(w, http.StatusOK, purchaseResponse{
		TransactionID: transactionID,
		Message:       "purchase success",
	})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}
