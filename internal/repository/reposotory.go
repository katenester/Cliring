package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"cliring/internal/domain"
)

// ErrNotFound is returned when a resource is not found.
var ErrNotFound = errors.New("resource not found")

// Repository handles database operations for the Cliring API.
type Repository struct {
	db *sql.DB
}

// NewRepository creates a new Repository instance.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// CreateDeal creates a new deal in the database.
func (r *Repository) CreateDeal(ctx context.Context, req domain.Deal) (*domain.Deal, error) {
	query := `
		INSERT INTO deals (dealership_id, manager_id, client_id, is_completed, created_at, updated_at)
		VALUES ($1, $2, $3, $4, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		RETURNING deal_id, is_completed, created_at, updated_at, dealership_id, manager_id, client_id`

	var deal domain.Deal
	err := r.db.QueryRowContext(ctx, query,
		req.DealershipID, req.ManagerID, req.ClientID, req.IsCompleted,
	).Scan(
		&deal.DealID, &deal.IsCompleted, &deal.CreatedAt, &deal.UpdatedAt,
		&deal.DealershipID, &deal.ManagerID, &deal.ClientID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create deal: %w", err)
	}

	return &deal, nil
}

// GetDeal retrieves a deal by its ID.
func (r *Repository) GetDeal(ctx context.Context, dealID int) (*domain.Deal, error) {
	query := `
		SELECT deal_id, is_completed, created_at, updated_at, dealership_id, manager_id, client_id
		FROM deals
		WHERE deal_id = $1`

	var deal domain.Deal
	err := r.db.QueryRowContext(ctx, query, dealID).Scan(
		&deal.DealID, &deal.IsCompleted, &deal.CreatedAt, &deal.UpdatedAt,
		&deal.DealershipID, &deal.ManagerID, &deal.ClientID,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get deal: %w", err)
	}

	return &deal, nil
}

// DeleteDeal deletes a deal by its ID along with related orders and monetary settlements.
func (r *Repository) DeleteDeal(ctx context.Context, dealID int) error {
	// Begin transaction
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	// Verify deal exists
	query := `SELECT deal_id FROM deals WHERE deal_id = $1`
	var id int
	err = tx.QueryRowContext(ctx, query, dealID).Scan(&id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("failed to verify deal: %w", err)
	}

	// Delete related orders
	query = `DELETE FROM orders WHERE deal_id = $1`
	_, err = tx.ExecContext(ctx, query, dealID)
	if err != nil {
		return fmt.Errorf("failed to delete orders: %w", err)
	}

	// Delete related monetary settlements
	query = `DELETE FROM monetary_settlements WHERE deal_id = $1`
	_, err = tx.ExecContext(ctx, query, dealID)
	if err != nil {
		return fmt.Errorf("failed to delete monetary settlements: %w", err)
	}

	// Delete deal
	query = `DELETE FROM deals WHERE deal_id = $1`
	result, err := tx.ExecContext(ctx, query, dealID)
	if err != nil {
		return fmt.Errorf("failed to delete deal: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// ListOrders retrieves a paginated list of orders for a client.
func (r *Repository) ListOrders(ctx context.Context, clientID int) ([]*domain.Order, int, error) {
	// Count total orders
	countQuery := `
		SELECT COUNT(o.order_id)
		FROM orders o
		JOIN deals d ON o.deal_id = d.deal_id
		WHERE d.client_id = $1`

	var total int
	err := r.db.QueryRowContext(ctx, countQuery, clientID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count orders: %w", err)
	}

	// Retrieve orders
	query := `
		SELECT o.order_id, o.deal_id, o.order_type_id, o.amount, o.status, o.created_at, o.updated_at, 
			o.need_and_orders_id, o.bank_id
		FROM orders o
		JOIN deals d ON o.deal_id = d.deal_id
		WHERE d.client_id = $1
		ORDER BY o.created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, clientID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query orders: %w", err)
	}
	defer rows.Close()

	var orders []*domain.Order
	for rows.Next() {
		var order domain.Order
		var needAndOrdersID, bankID sql.NullInt64
		err := rows.Scan(
			&order.OrderID, &order.DealID, &order.OrderTypeID, &order.Amount, &order.Status,
			&order.CreatedAt, &order.UpdatedAt, &needAndOrdersID, &bankID,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan order: %w", err)
		}
		if needAndOrdersID.Valid {
			needAndOrdersIDInt := int(needAndOrdersID.Int64)
			order.NeedAndOrdersID = &needAndOrdersIDInt
		}
		if bankID.Valid {
			bankIDInt := int(bankID.Int64)
			order.BankID = &bankIDInt
		}
		orders = append(orders, &order)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating orders: %w", err)
	}

	return orders, total, nil
}

// CreateOrder creates a new order in the database.
func (r *Repository) CreateOrder(ctx context.Context, order *domain.Order) (*domain.Order, error) {
	query := `
		INSERT INTO orders (deal_id, order_type_id, amount, status, created_at, updated_at, need_and_orders_id, bank_id)
		VALUES ($1, $2, $3, $4, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, $5, $6)
		RETURNING order_id, deal_id, order_type_id, amount, status, created_at, updated_at, need_and_orders_id, bank_id`

	var createdOrder domain.Order
	var needAndOrdersID, bankID sql.NullInt64
	err := r.db.QueryRowContext(ctx, query,
		order.DealID, order.OrderTypeID, order.Amount, order.Status, order.NeedAndOrdersID, order.BankID,
	).Scan(
		&createdOrder.OrderID, &createdOrder.DealID, &createdOrder.OrderTypeID, &createdOrder.Amount,
		&createdOrder.Status, &createdOrder.CreatedAt, &createdOrder.UpdatedAt, &needAndOrdersID, &bankID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	if needAndOrdersID.Valid {
		needAndOrdersIDInt := int(needAndOrdersID.Int64)
		createdOrder.NeedAndOrdersID = &needAndOrdersIDInt
	}
	if bankID.Valid {
		bankIDInt := int(bankID.Int64)
		createdOrder.BankID = &bankIDInt
	}

	return &createdOrder, nil
}

// GetOrder retrieves an order by its ID.
func (r *Repository) GetOrder(ctx context.Context, orderID int) (*domain.Order, error) {
	query := `
		SELECT order_id, deal_id, order_type_id, amount, status, created_at, updated_at, need_and_orders_id, bank_id
		FROM orders
		WHERE order_id = $1`

	var order domain.Order
	var needAndOrdersID, bankID sql.NullInt64
	err := r.db.QueryRowContext(ctx, query, orderID).Scan(
		&order.OrderID, &order.DealID, &order.OrderTypeID, &order.Amount, &order.Status,
		&order.CreatedAt, &order.UpdatedAt, &needAndOrdersID, &bankID,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	if needAndOrdersID.Valid {
		needAndOrdersIDInt := int(needAndOrdersID.Int64)
		order.NeedAndOrdersID = &needAndOrdersIDInt
	}
	if bankID.Valid {
		bankIDInt := int(bankID.Int64)
		order.BankID = &bankIDInt
	}

	return &order, nil
}

// UpdateOrder updates an existing order in the database.
func (r *Repository) UpdateOrder(ctx context.Context, order *domain.Order) (*domain.Order, error) {
	query := `
		UPDATE orders
		SET deal_id = $1, order_type_id = $2, amount = $3, status = $4, updated_at = CURRENT_TIMESTAMP,
			need_and_orders_id = $5, bank_id = $6
		WHERE order_id = $7
		RETURNING order_id, deal_id, order_type_id, amount, status, created_at, updated_at, need_and_orders_id, bank_id`

	var updatedOrder domain.Order
	var needAndOrdersID, bankID sql.NullInt64
	err := r.db.QueryRowContext(ctx, query,
		order.DealID, order.OrderTypeID, order.Amount, order.Status, order.NeedAndOrdersID, order.BankID, order.OrderID,
	).Scan(
		&updatedOrder.OrderID, &updatedOrder.DealID, &updatedOrder.OrderTypeID, &updatedOrder.Amount,
		&updatedOrder.Status, &updatedOrder.CreatedAt, &updatedOrder.UpdatedAt, &needAndOrdersID, &bankID,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to update order: %w", err)
	}

	if needAndOrdersID.Valid {
		needAndOrdersIDInt := int(needAndOrdersID.Int64)
		updatedOrder.NeedAndOrdersID = &needAndOrdersIDInt
	}
	if bankID.Valid {
		bankIDInt := int(bankID.Int64)
		updatedOrder.BankID = &bankIDInt
	}

	return &updatedOrder, nil
}

// ListMonetarySettlements retrieves a paginated list of monetary settlements for a deal.
func (r *Repository) ListMonetarySettlements(ctx context.Context, dealID int) ([]*domain.MonetarySettlement, int, error) {
	// Count total settlements
	countQuery := `
		SELECT COUNT(monetary_settlement_id)
		FROM monetary_settlements
		WHERE deal_id = $1`

	var total int
	err := r.db.QueryRowContext(ctx, countQuery, dealID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count monetary settlements: %w", err)
	}

	query := `
		SELECT monetary_settlement_id, deal_id, amount, status, created_at, updated_at, bank_id
		FROM monetary_settlements
		WHERE deal_id = $1`

	rows, err := r.db.QueryContext(ctx, query, dealID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query monetary settlements: %w", err)
	}
	defer rows.Close()

	var settlements []*domain.MonetarySettlement
	for rows.Next() {
		var settlement domain.MonetarySettlement
		var bankID sql.NullInt64
		err := rows.Scan(
			&settlement.MonetarySettlementID, &settlement.DealID, &settlement.Amount, &settlement.Status,
			&settlement.CreatedAt, &settlement.UpdatedAt, &bankID,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan monetary settlement: %w", err)
		}
		if bankID.Valid {
			bankIDInt := int(bankID.Int64)
			settlement.BankID = &bankIDInt
		}
		settlements = append(settlements, &settlement)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating monetary settlements: %w", err)
	}

	return settlements, total, nil
}

// CreateMonetarySettlement creates a new monetary settlement in the database.
func (r *Repository) CreateMonetarySettlement(ctx context.Context, settlement *domain.MonetarySettlement) (*domain.MonetarySettlement, error) {
	query := `
		INSERT INTO monetary_settlements (deal_id, amount, status, created_at, updated_at, bank_id)
		VALUES ($1, $2, $3, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, $4)
		RETURNING monetary_settlement_id, deal_id, amount, status, created_at, updated_at, bank_id`

	var createdSettlement domain.MonetarySettlement
	var bankID sql.NullInt64
	err := r.db.QueryRowContext(ctx, query,
		settlement.DealID, settlement.Amount, settlement.Status, settlement.BankID,
	).Scan(
		&createdSettlement.MonetarySettlementID, &createdSettlement.DealID, &createdSettlement.Amount,
		&createdSettlement.Status, &createdSettlement.CreatedAt, &createdSettlement.UpdatedAt, &bankID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create monetary settlement: %w", err)
	}

	if bankID.Valid {
		bankIDInt := int(bankID.Int64)
		createdSettlement.BankID = &bankIDInt
	}

	return &createdSettlement, nil
}
