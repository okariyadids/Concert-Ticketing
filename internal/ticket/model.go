package ticket

// Ticket merepresentasikan 1 baris di table tickets.
type Ticket struct {
	ID    int64
	Name  string
	Stock int
	Price float64
}

// Transaction merepresentasikan 1 baris di table transactions.
type Transaction struct {
	ID        int64
	TicketID  int64
	BuyerName string
	Status    string
}
