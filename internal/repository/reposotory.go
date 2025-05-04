package repository

import (
	"cliring/pkg/postgres"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"cliring/internal/domain"
)

// Errors returned by the service layer.
var (
	ErrInvalidInput = errors.New("invalid input")
	ErrNotFound     = errors.New("resource not found")
	ErrUnauthorized = errors.New("unauthorized access")
)

// Repository handles database operations for the Cliring API.
type Repository struct {
	db *postgres.Postgres
}

// NewRepository creates a new Repository instance.
func NewRepository(db *postgres.Postgres) *Repository {
	return &Repository{db: db}
}

// CreateDeal creates a new deal in the database.
func (r *Repository) CreateDeal(ctx context.Context, req domain.Deal) (*domain.Deal, error) {
	query := `
		INSERT INTO deals (deal_id, dealership_id, manager_id, client_id)
		VALUES ($1, $2, $3, $4)
		RETURNING deal_id, is_completed, created_at, updated_at, dealership_id, manager_id, client_id`

	var deal domain.Deal
	err := r.db.Conn.QueryRow(ctx, query,
		req.DealID, req.DealershipID, req.ManagerID, req.ClientID,
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
	err := r.db.Conn.QueryRow(ctx, query, dealID).Scan(
		&deal.DealID, &deal.IsCompleted, &deal.CreatedAt, &deal.UpdatedAt,
		&deal.DealershipID, &deal.ManagerID, &deal.ClientID,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get deal: %w", err)
	}

	return &deal, nil
}

// DeleteDeal deletes a deal by its ID along with related orders and monetary settlements.
func (r *Repository) DeleteDeal(ctx context.Context, dealID int) error {
	// Begin transaction
	tx, err := r.db.Conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	// Verify deal exists
	query := `SELECT deal_id FROM deals WHERE deal_id = $1`
	var id int
	err = tx.QueryRow(ctx, query, dealID).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("failed to verify deal: %w", err)
	}

	// Delete related orders
	query = `DELETE FROM orders WHERE deal_id = $1`
	_, err = tx.Exec(ctx, query, dealID)
	if err != nil {
		return fmt.Errorf("failed to delete orders: %w", err)
	}

	// Delete related monetary settlements
	query = `DELETE FROM monetary_settlements WHERE deal_id = $1`
	_, err = tx.Exec(ctx, query, dealID)
	if err != nil {
		return fmt.Errorf("failed to delete monetary settlements: %w", err)
	}

	// Delete deal
	query = `DELETE FROM deals WHERE deal_id = $1`
	result, err := tx.Exec(ctx, query, dealID)
	if err != nil {
		return fmt.Errorf("failed to delete deal: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}

	// Commit transaction
	if err = tx.Commit(ctx); err != nil {
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
	err := r.db.Conn.QueryRow(ctx, countQuery, clientID).Scan(&total)
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

	rows, err := r.db.Conn.Query(ctx, query, clientID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query orders: %w", err)
	}
	defer rows.Close()

	var orders []*domain.Order
	for rows.Next() {
		var order domain.Order
		var needAndOrdersID, bankID pgtype.Int4
		err := rows.Scan(
			&order.OrderID, &order.DealID, &order.OrderTypeID, &order.Amount, &order.Status,
			&order.CreatedAt, &order.UpdatedAt, &needAndOrdersID, &bankID,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan order: %w", err)
		}
		if needAndOrdersID.Valid {
			needAndOrdersIDInt := int(needAndOrdersID.Int32)
			order.NeedAndOrdersID = &needAndOrdersIDInt
		}
		if bankID.Valid {
			bankIDInt := int(bankID.Int32)
			order.BankID = &bankIDInt
		}
		orders = append(orders, &order)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating orders: %w", err)
	}

	return orders, total, nil
}

// ListOrdersByDeals retrieves all orders for a specific deal.
func (r *Repository) ListOrdersByDeals(ctx context.Context, dealID int) ([]*domain.Order, error) {
	query := `
		SELECT order_id, deal_id, order_type_id, amount, status, created_at, updated_at, need_and_orders_id, bank_id
		FROM orders
		WHERE deal_id = $1
		ORDER BY created_at DESC`

	rows, err := r.db.Conn.Query(ctx, query, dealID)
	if err != nil {
		return nil, fmt.Errorf("failed to query orders: %w", err)
	}
	defer rows.Close()

	var orders []*domain.Order
	for rows.Next() {
		var order domain.Order
		var needAndOrdersID, bankID pgtype.Int4
		err := rows.Scan(
			&order.OrderID, &order.DealID, &order.OrderTypeID, &order.Amount, &order.Status,
			&order.CreatedAt, &order.UpdatedAt, &needAndOrdersID, &bankID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan order: %w", err)
		}
		if needAndOrdersID.Valid {
			needAndOrdersIDInt := int(needAndOrdersID.Int32)
			order.NeedAndOrdersID = &needAndOrdersIDInt
		}
		if bankID.Valid {
			bankIDInt := int(bankID.Int32)
			order.BankID = &bankIDInt
		}
		orders = append(orders, &order)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating orders: %w", err)
	}

	return orders, nil
}

// CreateOrder creates a new order in the database.
func (r *Repository) CreateOrder(ctx context.Context, order *domain.Order) (*domain.Order, error) {
	query := `
		INSERT INTO orders (deal_id, order_type_id, amount, status, created_at, updated_at, need_and_orders_id, bank_id)
		VALUES ($1, $2, $3, $4, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, $5, $6)
		RETURNING order_id, deal_id, order_type_id, amount, status, created_at, updated_at, need_and_orders_id, bank_id`

	var createdOrder domain.Order
	var needAndOrdersID, bankID pgtype.Int4
	err := r.db.Conn.QueryRow(ctx, query,
		order.DealID, order.OrderTypeID, order.Amount, order.Status, order.NeedAndOrdersID, order.BankID,
	).Scan(
		&createdOrder.OrderID, &createdOrder.DealID, &createdOrder.OrderTypeID, &createdOrder.Amount,
		&createdOrder.Status, &createdOrder.CreatedAt, &createdOrder.UpdatedAt, &needAndOrdersID, &bankID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	if needAndOrdersID.Valid {
		needAndOrdersIDInt := int(needAndOrdersID.Int32)
		createdOrder.NeedAndOrdersID = &needAndOrdersIDInt
	}
	if bankID.Valid {
		bankIDInt := int(bankID.Int32)
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
	var needAndOrdersID, bankID pgtype.Int4
	err := r.db.Conn.QueryRow(ctx, query, orderID).Scan(
		&order.OrderID, &order.DealID, &order.OrderTypeID, &order.Amount, &order.Status,
		&order.CreatedAt, &order.UpdatedAt, &needAndOrdersID, &bankID,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	if needAndOrdersID.Valid {
		needAndOrdersIDInt := int(needAndOrdersID.Int32)
		order.NeedAndOrdersID = &needAndOrdersIDInt
	}
	if bankID.Valid {
		bankIDInt := int(bankID.Int32)
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
	var needAndOrdersID, bankID pgtype.Int4
	err := r.db.Conn.QueryRow(ctx, query,
		order.DealID, order.OrderTypeID, order.Amount, order.Status, order.NeedAndOrdersID, order.BankID, order.OrderID,
	).Scan(
		&updatedOrder.OrderID, &updatedOrder.DealID, &updatedOrder.OrderTypeID, &updatedOrder.Amount,
		&updatedOrder.Status, &updatedOrder.CreatedAt, &updatedOrder.UpdatedAt, &needAndOrdersID, &bankID,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to update order: %w", err)
	}

	if needAndOrdersID.Valid {
		needAndOrdersIDInt := int(needAndOrdersID.Int32)
		updatedOrder.NeedAndOrdersID = &needAndOrdersIDInt
	}
	if bankID.Valid {
		bankIDInt := int(bankID.Int32)
		updatedOrder.BankID = &bankIDInt
	}

	return &updatedOrder, nil
}

// ListMonetarySettlements performs a netting calculation (bilateral or multilateral) based on orders for a deal.
func (r *Repository) ListMonetarySettlements(ctx context.Context, dealID, page, limit int) ([]*domain.MonetarySettlement, int, error) {
	// Validate inputs
	if dealID <= 0 {
		return nil, 0, fmt.Errorf("invalid deal_id: %w", ErrInvalidInput)
	}
	if page < 1 || limit < 1 {
		return nil, 0, fmt.Errorf("invalid pagination parameters: %w", ErrInvalidInput)
	}

	// Get orders for the deal
	orders, err := r.ListOrdersByDeals(ctx, dealID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list orders: %w", err)
	}

	// Check if any order has a bank_id
	hasBank := false
	for _, order := range orders {
		if order.BankID != nil {
			hasBank = true
			break
		}
	}

	// Participants: Client (C), Rolf (R), Bank (B) if applicable
	participants := []string{"Client", "Rolf"}
	if hasBank {
		participants = append(participants, "Bank")
	}
	n := len(participants)

	// Initialize obligation matrix: obligations[i][j] is amount participant i owes to participant j
	obligations := make([][]float64, n)
	for i := range obligations {
		obligations[i] = make([]float64, n)
	}

	// Build obligation matrix based on order_type_id
	for _, order := range orders {
		amount := order.Amount
		switch order.OrderTypeID {
		case 1: // Purchase: Client owes Rolf
			obligations[0][1] += amount // Client -> Rolf
		case 2: // Credit: Client owes Bank, Bank owes Rolf
			if order.BankID != nil {
				obligations[0][2] += amount // Client -> Bank
				obligations[2][1] += amount // Bank -> Rolf
			} else {
				// Fallback to Client -> Rolf if no bank
				obligations[0][1] += amount
			}
		case 3: // Trade-in: Rolf owes Client
			obligations[1][0] += amount // Rolf -> Client
		default:
			return nil, 0, fmt.Errorf("unknown order_type_id %d: %w", order.OrderTypeID, ErrInvalidInput)
		}
	}

	// Calculate net positions: net[i] = sum(a_ij) - sum(a_ji)
	netPositions := make([]float64, n)
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			if i != j {
				netPositions[i] += obligations[i][j] // Outgoing
				netPositions[i] -= obligations[j][i] // Incoming
			}
		}
	}

	// Create MonetarySettlements for non-zero net positions
	var settlements []*domain.MonetarySettlement
	now := time.Now()
	for i, net := range netPositions {
		if net != 0 {
			settlement := &domain.MonetarySettlement{
				MonetarySettlementID: 0, // Not saved in DB yet
				DealID:               &dealID,
				Amount:               net, // Positive: owes, Negative: owed
				Status:               domain.StatusPending,
				CreatedAt:            now,
				UpdatedAt:            now,
			}
			if hasBank && participants[i] == "Bank" {
				// Set BankID for bank participant (assume bank_id from first order with bank)
				for _, order := range orders {
					if order.BankID != nil {
						settlement.BankID = order.BankID
						break
					}
				}
			}
			settlements = append(settlements, settlement)
		}
	}

	// Apply pagination
	total := len(settlements)
	start := (page - 1) * limit
	end := start + limit
	if start > total {
		return nil, total, nil
	}
	if end > total {
		end = total
	}

	return settlements[start:end], total, nil
}

// CreateMonetarySettlement creates a new monetary settlement in the database.
func (r *Repository) CreateMonetarySettlement(ctx context.Context, settlement *domain.MonetarySettlement) (*domain.MonetarySettlement, error) {
	query := `
		INSERT INTO monetary_settlements (deal_id, amount, status, created_at, updated_at, bank_id)
		VALUES ($1, $2, $3, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, $4)
		RETURNING monetary_settlement_id, deal_id, amount, status, created_at, updated_at, bank_id`

	var createdSettlement domain.MonetarySettlement
	var bankID pgtype.Int4
	err := r.db.Conn.QueryRow(ctx, query,
		settlement.DealID, settlement.Amount, settlement.Status, settlement.BankID,
	).Scan(
		&createdSettlement.MonetarySettlementID, &createdSettlement.DealID, &createdSettlement.Amount,
		&createdSettlement.Status, &createdSettlement.CreatedAt, &createdSettlement.UpdatedAt, &bankID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create monetary settlement: %w", err)
	}

	if bankID.Valid {
		bankIDInt := int(bankID.Int32)
		createdSettlement.BankID = &bankIDInt
	}

	return &createdSettlement, nil
}
