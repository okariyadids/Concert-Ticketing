package payment

type Payment struct {
	ID                int64
	TransactionID     int64
	ExternalPaymentID string
	Amount            float64
	Status            string
}
