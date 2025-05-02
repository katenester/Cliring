package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"cliring/internal/domain"
	"cliring/pkg/postgres"
)

const (
	PendingStatus   = "pending"
	ExecutedStatus  = "executed"
	CancelledStatus = "cancelled"
)

// Errors returned by the repository layer.
var (
	ErrNotFound     = errors.New("resource not found")
	ErrInvalidInput = errors.New("invalid input")
)

// Repository provides access to the database.
type Repository struct {
	db *postgres.Postgres
}

// NewRepository creates a new Repository instance.
func NewRepository(db *postgres.Postgres) *Repository {
	return &Repository{db: db}
}

// CreateDeal creates a new deal.
func (r *Repository) CreateDeal(ctx context.Context) (*domain.Deal, error) {
	query := `
		INSERT INTO deals (is_completed, created_at, updated_at)
		VALUES ($1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		RETURNING deal_id, is_completed, created_at, updated_at
	`
	var createdDeal domain.Deal
	err := r.db.Conn.QueryRow(ctx, query, false).Scan(
		&createdDeal.DealID,
		&createdDeal.IsCompleted,
		&createdDeal.CreatedAt,
		&createdDeal.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create deal: %w", err)
	}

	return &createdDeal, nil
}

// GetDeal retrieves a deal by ID.
func (r *Repository) GetDeal(ctx context.Context, dealID int) (*domain.Deal, error) {
	query := `
		SELECT deal_id, is_completed, created_at, updated_at
		FROM deals
		WHERE deal_id = $1
	`
	var deal domain.Deal
	err := r.db.Conn.QueryRow(ctx, query, dealID).Scan(
		&deal.DealID,
		&deal.IsCompleted,
		&deal.CreatedAt,
		&deal.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("deal %d: %w", dealID, ErrNotFound)
		}
		return nil, fmt.Errorf("failed to get deal: %w", err)
	}

	return &deal, nil
}

// DeleteDeal deletes a deal by ID.
func (r *Repository) DeleteDeal(ctx context.Context, dealID int) error {
	tx, err := r.db.Conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Delete from deals
	query := `DELETE FROM deals WHERE deal_id = $1`
	result, err := tx.Exec(ctx, query, dealID)
	if err != nil {
		return fmt.Errorf("failed to delete deal: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("deal %d: %w", dealID, ErrNotFound)
	}

	// Update orders
	query = `UPDATE orders SET status=$1 WHERE deal_id = $2`
	_, err = tx.Exec(ctx, query, CancelledStatus, dealID)
	if err != nil {
		return fmt.Errorf("failed to update orders: %w", err)
	}

	// Update monetary_settlements
	query = `UPDATE monetary_settlements SET status=$1 WHERE deal_id = $2`
	_, err = tx.Exec(ctx, query, CancelledStatus, dealID)
	if err != nil {
		return fmt.Errorf("failed to update monetary settlements: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// CreateOrder creates a new order.
func (r *Repository) CreateOrder(ctx context.Context, order *domain.Order) (*domain.Order, error) {
	query := `
		INSERT INTO orders (deal_id, amount, status, client_id, "counterparty-id", need_and_orders_id, created_at, updated_at)
		VALUES ($1, $2, COALESCE($3, 'pending'), $4, $5, $6, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		RETURNING order_id, deal_id, amount, status, client_id, "counterparty-id", need_and_orders_id, created_at, updated_at
	`
	var createdOrder domain.Order
	err := r.db.Conn.QueryRow(ctx, query,
		order.DealID,
		order.Amount,
		order.Status,
		order.ClientID,
		order.CounterpartyID,
		order.NeedAndOrdersID,
	).Scan(
		&createdOrder.OrderID,
		&createdOrder.DealID,
		&createdOrder.Amount,
		&createdOrder.Status,
		&createdOrder.ClientID,
		&createdOrder.CounterpartyID,
		&createdOrder.NeedAndOrdersID,
		&createdOrder.CreatedAt,
		&createdOrder.UpdatedAt,
	)
	if err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "23503" {
			return nil, fmt.Errorf("invalid deal_id: %w", ErrInvalidInput)
		}
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	return &createdOrder, nil
}

// GetOrder retrieves an order by ID.
func (r *Repository) GetOrder(ctx context.Context, orderID int) (*domain.Order, error) {
	query := `
		SELECT order_id, deal_id, amount, status, client_id, "counterparty-id", need_and_orders_id, created_at, updated_at
		FROM orders
		WHERE order_id = $1
	`
	var order domain.Order
	err := r.db.Conn.QueryRow(ctx, query, orderID).Scan(
		&order.OrderID,
		&order.DealID,
		&order.Amount,
		&order.Status,
		&order.ClientID,
		&order.CounterpartyID,
		&order.NeedAndOrdersID,
		&order.CreatedAt,
		&order.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("order %d: %w", orderID, ErrNotFound)
		}
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	return &order, nil
}

// UpdateOrder updates an existing order.
func (r *Repository) UpdateOrder(ctx context.Context, order *domain.Order) (*domain.Order, error) {
	query := `
		UPDATE orders
		SET deal_id = $1, amount = $2, status = $3, "counterparty-id" = $4, need_and_orders_id = $5, updated_at = CURRENT_TIMESTAMP
		WHERE order_id = $6
		RETURNING order_id, deal_id, amount, status, client_id, "counterparty-id", need_and_orders_id, created_at, updated_at
	`
	var updatedOrder domain.Order
	err := r.db.Conn.QueryRow(ctx, query,
		order.DealID,
		order.Amount,
		order.Status,
		order.CounterpartyID,
		order.NeedAndOrdersID,
		order.OrderID,
	).Scan(
		&updatedOrder.OrderID,
		&updatedOrder.DealID,
		&updatedOrder.Amount,
		&updatedOrder.Status,
		&updatedOrder.ClientID,
		&updatedOrder.CounterpartyID,
		&updatedOrder.NeedAndOrdersID,
		&updatedOrder.CreatedAt,
		&updatedOrder.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("order %d: %w", order.OrderID, ErrNotFound)
		}
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "23503" {
			return nil, fmt.Errorf("invalid deal_id: %w", ErrInvalidInput)
		}
		return nil, fmt.Errorf("failed to update order: %w", err)
	}

	return &updatedOrder, nil
}

// DeleteOrder deletes an order by ID.
func (r *Repository) DeleteOrder(ctx context.Context, orderID int) error {
	query := `DELETE FROM orders WHERE order_id = $1`
	result, err := r.db.Conn.Exec(ctx, query, orderID)
	if err != nil {
		return fmt.Errorf("failed to delete order: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("order %d: %w", orderID, ErrNotFound)
	}

	return nil
}

// ListOrders retrieves a paginated list of orders for a client.
func (r *Repository) ListOrders(ctx context.Context, clientID int) ([]*domain.Order, int, error) {
	// Count total orders
	countQuery := `SELECT COUNT(*) FROM orders WHERE client_id = $1`
	var total int
	err := r.db.Conn.QueryRow(ctx, countQuery, clientID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count orders: %w", err)
	}

	// Fetch orders
	query := `
		SELECT order_id, deal_id, amount, status, client_id, "counterparty-id", need_and_orders_id, created_at, updated_at
		FROM orders
		WHERE client_id = $1
	`
	rows, err := r.db.Conn.Query(ctx, query, clientID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list orders: %w", err)
	}
	defer rows.Close()

	var orders []*domain.Order
	for rows.Next() {
		var order domain.Order
		if err := rows.Scan(
			&order.OrderID,
			&order.DealID,
			&order.Amount,
			&order.Status,
			&order.ClientID,
			&order.CounterpartyID,
			&order.NeedAndOrdersID,
			&order.CreatedAt,
			&order.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan order: %w", err)
		}
		orders = append(orders, &order)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating orders: %w", err)
	}

	return orders, total, nil
}

// CreateMonetarySettlement creates a new monetary settlement.
func (r *Repository) CreateMonetarySettlement(ctx context.Context, settlement *domain.MonetarySettlement) (*domain.MonetarySettlement, error) {
	query := `
		INSERT INTO monetary_settlements (deal_id, amount, status, payment_method, client_id, created_at, updated_at)
		VALUES ($1, $2, COALESCE($3, 'pending'), $4, $5, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		RETURNING monetary_settlement_id, deal_id, amount, status, payment_method, client_id, created_at, updated_at
	`
	var createdSettlement domain.MonetarySettlement
	err := r.db.Conn.QueryRow(ctx, query,
		settlement.DealID,
		settlement.Amount,
		settlement.Status,
		settlement.PaymentMethod,
		settlement.ClientID,
	).Scan(
		&createdSettlement.MonetarySettlementID,
		&createdSettlement.DealID,
		&createdSettlement.Amount,
		&createdSettlement.Status,
		&createdSettlement.PaymentMethod,
		&createdSettlement.ClientID,
		&createdSettlement.CreatedAt,
		&createdSettlement.UpdatedAt,
	)
	if err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "23503" {
			return nil, fmt.Errorf("invalid deal_id: %w", ErrInvalidInput)
		}
		return nil, fmt.Errorf("failed to create monetary settlement: %w", err)
	}

	return &createdSettlement, nil
}

// GetMonetarySettlement retrieves a monetary settlement by ID.
func (r *Repository) GetMonetarySettlement(ctx context.Context, settlementID int) (*domain.MonetarySettlement, error) {
	query := `
		SELECT monetary_settlement_id, deal_id, amount, status, payment_method, client_id, created_at, updated_at
		FROM monetary_settlements
		WHERE monetary_settlement_id = $1
	`
	var settlement domain.MonetarySettlement
	err := r.db.Conn.QueryRow(ctx, query, settlementID).Scan(
		&settlement.MonetarySettlementID,
		&settlement.DealID,
		&settlement.Amount,
		&settlement.Status,
		&settlement.PaymentMethod,
		&settlement.ClientID,
		&settlement.CreatedAt,
		&settlement.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("settlement %d: %w", settlementID, ErrNotFound)
		}
		return nil, fmt.Errorf("failed to get monetary settlement: %w", err)
	}

	return &settlement, nil
}

// ListMonetarySettlements retrieves a paginated list of monetary settlements for a client.
func (r *Repository) ListMonetarySettlements(ctx context.Context, clientID int) ([]*domain.MonetarySettlement, int, error) {

	// Count total settlements
	countQuery := `SELECT COUNT(*) FROM monetary_settlements WHERE client_id = $1`
	var total int
	err := r.db.Conn.QueryRow(ctx, countQuery, clientID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count monetary settlements: %w", err)
	}

	// Fetch settlements
	query := `
		SELECT monetary_settlement_id, deal_id, amount, status, payment_method, client_id, created_at, updated_at
		FROM monetary_settlements
		WHERE client_id = $1
	`
	rows, err := r.db.Conn.Query(ctx, query, clientID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list monetary settlements: %w", err)
	}
	defer rows.Close()

	var settlements []*domain.MonetarySettlement
	for rows.Next() {
		var settlement domain.MonetarySettlement
		if err := rows.Scan(
			&settlement.MonetarySettlementID,
			&settlement.DealID,
			&settlement.Amount,
			&settlement.Status,
			&settlement.PaymentMethod,
			&settlement.ClientID,
			&settlement.CreatedAt,
			&settlement.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan monetary settlement: %w", err)
		}
		settlements = append(settlements, &settlement)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating monetary settlements: %w", err)
	}

	return settlements, total, nil
}

// ListClearingTransactions retrieves a paginated list of clearing transactions for a client.
func (r *Repository) ListClearingTransactions(ctx context.Context, clientID int) ([]*domain.ClearingTransaction, int, error) {
	// Count total transactions
	countQuery := `
		SELECT COUNT(*)
		FROM clearing_transactions ct
		LEFT JOIN orders o ON ct.order_id = o.order_id
		LEFT JOIN monetary_settlements ms ON ct.monetary_settlement_id = ms.monetary_settlement_id
		WHERE o.client_id = $1 OR ms.client_id = $1
	`
	var total int
	err := r.db.Conn.QueryRow(ctx, countQuery, clientID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count clearing transactions: %w", err)
	}

	// Fetch transactions
	query := `
		SELECT ct.clearing_transaction_id, ct.order_id, ct.monetary_settlement_id, ct.amount, ct.status, ct.created_at, ct.updated_at
		FROM clearing_transactions ct
		LEFT JOIN orders o ON ct.order_id = o.order_id
		LEFT JOIN monetary_settlements ms ON ct.monetary_settlement_id = ms.monetary_settlement_id
		WHERE o.client_id = $1 OR ms.client_id = $1
	`
	rows, err := r.db.Conn.Query(ctx, query, clientID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list clearing transactions: %w", err)
	}
	defer rows.Close()

	var transactions []*domain.ClearingTransaction
	for rows.Next() {
		var tx domain.ClearingTransaction
		if err := rows.Scan(
			&tx.ClearingTransactionID,
			&tx.OrderID,
			&tx.MonetarySettlementID,
			&tx.Amount,
			&tx.Status,
			&tx.CreatedAt,
			&tx.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan clearing transaction: %w", err)
		}
		transactions = append(transactions, &tx)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating clearing transactions: %w", err)
	}

	return transactions, total, nil
}
