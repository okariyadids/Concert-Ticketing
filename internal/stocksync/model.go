package stocksync

type StockUpdate struct {
	TicketID int64
	Quantity int
	Version  int
}
