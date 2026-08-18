package stocksync

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

type syncRequest struct {
	TicketID int64 `json:"ticket_id"`
	Quantity int   `json:"quantity"`
	Version  int   `json:"version"`
}

type syncResponse struct {
	Applied bool `json:"applied"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func (handler *Handler) HandleSync(responseWriter http.ResponseWriter, request *http.Request) {
	var req syncRequest
	if err := json.NewDecoder(request.Body).Decode(&req); err != nil {
		writeJSON(responseWriter, http.StatusBadRequest, errorResponse{Error: "invalid request body"})
		return
	}

	applied, err := handler.repository.ApplyIfNewer(request.Context(), StockUpdate{
		TicketID: req.TicketID,
		Quantity: req.Quantity,
		Version:  req.Version,
	})
	if err != nil {
		writeJSON(responseWriter, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
		return
	}

	writeJSON(responseWriter, http.StatusOK, syncResponse{Applied: applied})
}

func writeJSON(responseWriter http.ResponseWriter, status int, body any) {
	responseWriter.Header().Set("Content-Type", "application/json")
	responseWriter.WriteHeader(status)
	json.NewEncoder(responseWriter).Encode(body)
}
