package domain

import (
	"time"
)

// ClientIDKey is the context key for client_id.
type ClientIDKey struct{}

// Error codes used in API responses.
const (
	ErrCodeInvalidInput    = "ERR_INVALID_INPUT"
	ErrCodeUnauthorized    = "ERR_UNAUTHORIZED"
	ErrCodeNotFound        = "ERR_NOT_FOUND"
	ErrCodeInternal        = "ERR_INTERNAL"
	ErrCodeInvalidClientID = "ERR_INVALID_CLIENT_ID"
)

// Status constants for entities.
const (
	StatusPending   = "pending"
	StatusCompleted = "completed"
	StatusCancelled = "cancelled"
	StatusExecuted  = "executed"
	StatusPlanned   = "planned"
	StatusPaid      = "paid"
	StatusOverdue   = "overdue"
	StatusActive    = "active"
	StatusFrozen    = "frozen"
	StatusClosed    = "closed"
)

// PaymentMethod constants.
const (
	PaymentMethodCard         = "card"
	PaymentMethodBankTransfer = "bank_transfer"
	PaymentMethodWallet       = "wallet"
)

// ErrorResponse represents an API error response.
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail contains details of an error.
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

// Deal represents a deal entity.
type Deal struct {
	DealID      int       `json:"deal_id"`
	IsCompleted bool      `json:"is_completed"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Order represents an order entity.
type Order struct {
	OrderID         int       `json:"order_id"`
	DealID          int       `json:"deal_id"`
	Amount          float64   `json:"amount"`
	Status          string    `json:"status"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	ClientID        int       `json:"client_id"`
	CounterpartyID  int       `json:"counterparty_id"`
	NeedAndOrdersID *int      `json:"need_and_orders_id,omitempty"`
}

// OrderCreate represents a request to create an order.
type OrderCreate struct {
	DealID          int     `json:"deal_id"`
	Amount          float64 `json:"amount"`
	Status          string  `json:"status"`
	CounterpartyID  int     `json:"counterparty_id"`
	NeedAndOrdersID *int    `json:"need_and_orders_id,omitempty"`
}

// MonetarySettlement represents a monetary settlement entity.
type MonetarySettlement struct {
	MonetarySettlementID int       `json:"monetary_settlement_id"`
	DealID               *int      `json:"deal_id,omitempty"`
	Amount               float64   `json:"amount"`
	Status               string    `json:"status"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
	PaymentMethod        *string   `json:"payment_method,omitempty"`
	ClientID             int       `json:"client_id"`
}

// MonetarySettlementCreate represents a request to create a monetary settlement.
type MonetarySettlementCreate struct {
	DealID        *int    `json:"deal_id,omitempty"`
	Amount        float64 `json:"amount"`
	Status        string  `json:"status"`
	PaymentMethod *string `json:"payment_method,omitempty"`
}

// ClearingTransaction represents a clearing transaction entity.
type ClearingTransaction struct {
	ClearingTransactionID int       `json:"clearing_transaction_id"`
	OrderID               *int      `json:"order_id,omitempty"`
	MonetarySettlementID  *int      `json:"monetary_settlement_id,omitempty"`
	Amount                float64   `json:"amount"`
	Status                string    `json:"status"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

// ClearingTransactionCreate represents a request to create a clearing transaction.
type ClearingTransactionCreate struct {
	OrderID              *int    `json:"order_id,omitempty"`
	MonetarySettlementID *int    `json:"monetary_settlement_id,omitempty"`
	Amount               float64 `json:"amount"`
	Status               string  `json:"status"`
}

// PayTransaction represents a payment transaction entity.
type PayTransaction struct {
	TransactionID        int       `json:"transaction_id"`
	MonetarySettlementID int       `json:"monetary_settlement_id"`
	Amount               float64   `json:"amount"`
	BankAccount          string    `json:"bank_account"`
	PaymentMethod        string    `json:"payment_method"`
	Status               string    `json:"status"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

// PayTransactionCreate represents a request to create a payment transaction.
type PayTransactionCreate struct {
	MonetarySettlementID int     `json:"monetary_settlement_id"`
	Amount               float64 `json:"amount"`
	BankAccount          string  `json:"bank_account"`
	PaymentMethod        string  `json:"payment_method"`
	Status               string  `json:"status"`
}

// PaymentSchedule represents a payment schedule entity.
type PaymentSchedule struct {
	ScheduleID           int       `json:"schedule_id"`
	MonetarySettlementID int       `json:"monetary_settlement_id"`
	DueDate              time.Time `json:"due_date"`
	Amount               float64   `json:"amount"`
	Status               string    `json:"status"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

// PaymentScheduleCreate represents a request to create a payment schedule.
type PaymentScheduleCreate struct {
	MonetarySettlementID int       `json:"monetary_settlement_id"`
	DueDate              time.Time `json:"due_date"`
	Amount               float64   `json:"amount"`
	Status               string    `json:"status"`
}

// PersonalWallet represents a personal wallet entity.
type PersonalWallet struct {
	WalletID   int       `json:"wallet_id"`
	Balance    float64   `json:"balance"`
	ContractID int       `json:"contract_id"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	ClientID   int       `json:"client_id"`
}

// PersonalWalletCreate represents a request to create a personal wallet.
type PersonalWalletCreate struct {
	Balance    float64 `json:"balance"`
	ContractID int     `json:"contract_id"`
	Status     string  `json:"status"`
}

// WalletOperation represents a wallet operation request.
type WalletOperation struct {
	Amount float64 `json:"amount"`
}
