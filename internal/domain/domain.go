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
	StatusExecuted  = "executed"
	StatusCancelled = "cancelled"
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
	DealID       int       `json:"deal_id"`
	IsCompleted  bool      `json:"is_completed"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	DealershipID int       `json:"dealership_id"`
	ManagerID    int       `json:"manager_id"`
	ClientID     int       `json:"client_id"`
}

// Order represents an order entity.
type Order struct {
	OrderID         int       `json:"order_id"`
	DealID          int       `json:"deal_id"`
	OrderTypeID     int       `json:"order_type_id"`
	Amount          float64   `json:"amount"`
	Status          string    `json:"status"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	NeedAndOrdersID *int      `json:"need_and_orders_id,omitempty"`
	BankID          *int      `json:"bank_id,omitempty"`
}

// OrderCreate represents a request to create an order.
type OrderCreate struct {
	DealID          int     `json:"deal_id"`
	OrderTypeID     int     `json:"order_type_id"`
	Amount          float64 `json:"amount"`
	NeedAndOrdersID *int    `json:"need_and_orders_id,omitempty"`
	BankID          *int    `json:"bank_id,omitempty"`
}

// MonetarySettlement represents a monetary settlement entity.
type MonetarySettlement struct {
	MonetarySettlementID int       `json:"monetary_settlement_id"`
	DealID               *int      `json:"deal_id"`
	Amount               float64   `json:"amount"`
	Status               string    `json:"status"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
	BankID               *int      `json:"bank_id,omitempty"`
}

// MonetarySettlementCreate represents a request to create a monetary settlement.
type MonetarySettlementCreate struct {
	DealID *int    `json:"deal_id"`
	Amount float64 `json:"amount"`
	BankID *int    `json:"bank_id,omitempty"`
}
