package accounting

type OutboxEntry struct {
	ID            int64
	TransactionID int64
	AttemptCount  int
	TicketID      int64
	BuyerName     string
}
