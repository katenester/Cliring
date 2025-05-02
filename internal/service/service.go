package service

import (
	"context"
	"errors"
	"fmt"

	"cliring/internal/domain"
	"cliring/internal/repository"
)

// Errors returned by the service layer.
var (
	ErrInvalidInput = errors.New("invalid input")
	ErrNotFound     = errors.New("resource not found")
	ErrUnauthorized = errors.New("unauthorized access")
)

// Service contains business logic for the Cliring API.
type Service struct {
	repo *repository.Repository
}

// NewService creates a new Service instance.
func NewService(repo *repository.Repository) *Service {
	return &Service{repo: repo}
}

// CreateDeal creates a new deal for the specified client.
func (s *Service) CreateDeal(ctx context.Context) (*domain.Deal, error) {
	createdDeal, err := s.repo.CreateDeal(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create deal: %w", err)
	}

	return createdDeal, nil
}

// DeleteDeal deletes a deal.
func (s *Service) DeleteDeal(ctx context.Context, clientID, dealID int) error {
	// Verify deal exists
	_, err := s.repo.GetDeal(ctx, dealID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return fmt.Errorf("deal not found: %w", ErrNotFound)
		}
		return fmt.Errorf("failed to get deal: %w", err)
	}

	if err := s.repo.DeleteDeal(ctx, dealID); err != nil {
		return fmt.Errorf("failed to delete deal: %w", err)
	}

	return nil
}

// ListOrders retrieves a paginated list of orders for the client.
func (s *Service) ListOrders(ctx context.Context, clientID int) ([]*domain.Order, int, error) {

	orders, total, err := s.repo.ListOrders(ctx, clientID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list orders: %w", err)
	}

	// Verify client_id ownership
	for _, order := range orders {
		if order.ClientID != clientID {
			return nil, 0, fmt.Errorf("unauthorized access to order %d: %w", order.OrderID, ErrUnauthorized)
		}
	}

	return orders, total, nil
}

// CreateOrder creates a new order for the specified client.
func (s *Service) CreateOrder(ctx context.Context, clientID int, req domain.OrderCreate) (*domain.Order, error) {
	// Validate input
	if req.Amount <= 0 {
		return nil, fmt.Errorf("amount must be positive: %w", ErrInvalidInput)
	}
	if req.DealID <= 0 {
		return nil, fmt.Errorf("invalid deal_id: %w", ErrInvalidInput)
	}
	if req.CounterpartyID <= 0 {
		return nil, fmt.Errorf("invalid counterparty_id: %w", ErrInvalidInput)
	}
	if req.Status != "" && req.Status != domain.StatusPending && req.Status != domain.StatusExecuted && req.Status != domain.StatusCancelled {
		return nil, fmt.Errorf("invalid status: %w", ErrInvalidInput)
	}

	// Verify deal exists
	_, err := s.repo.GetDeal(ctx, req.DealID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, fmt.Errorf("deal not found: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("failed to get deal: %w", err)
	}

	order := &domain.Order{
		DealID:          req.DealID,
		Amount:          req.Amount,
		Status:          req.Status,
		ClientID:        clientID,
		CounterpartyID:  req.CounterpartyID,
		NeedAndOrdersID: req.NeedAndOrdersID,
	}

	createdOrder, err := s.repo.CreateOrder(ctx, order)
	if err != nil {
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	return createdOrder, nil
}

// UpdateOrder updates an existing order.
func (s *Service) UpdateOrder(ctx context.Context, clientID, orderID int, req domain.OrderCreate) (*domain.Order, error) {
	// Fetch the order to verify existence and ownership
	order, err := s.repo.GetOrder(ctx, orderID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, fmt.Errorf("order not found: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	// Verify client_id ownership
	if order.ClientID != clientID {
		return nil, fmt.Errorf("unauthorized access to order %d: %w", orderID, ErrUnauthorized)
	}

	// Validate input
	if req.Amount <= 0 {
		return nil, fmt.Errorf("amount must be positive: %w", ErrInvalidInput)
	}
	if req.DealID <= 0 {
		return nil, fmt.Errorf("invalid deal_id: %w", ErrInvalidInput)
	}
	if req.CounterpartyID <= 0 {
		return nil, fmt.Errorf("invalid counterparty_id: %w", ErrInvalidInput)
	}
	if req.Status != "" && req.Status != domain.StatusPending && req.Status != domain.StatusExecuted && req.Status != domain.StatusCancelled {
		return nil, fmt.Errorf("invalid status: %w", ErrInvalidInput)
	}

	// Verify deal exists
	_, err = s.repo.GetDeal(ctx, req.DealID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, fmt.Errorf("deal not found: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("failed to get deal: %w", err)
	}

	// Update order fields
	order.DealID = req.DealID
	order.Amount = req.Amount
	order.Status = req.Status
	order.CounterpartyID = req.CounterpartyID
	order.NeedAndOrdersID = req.NeedAndOrdersID

	updatedOrder, err := s.repo.UpdateOrder(ctx, order)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, fmt.Errorf("order not found: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("failed to update order: %w", err)
	}

	return updatedOrder, nil
}

// DeleteOrder deletes an order.
func (s *Service) DeleteOrder(ctx context.Context, clientID, orderID int) error {
	// Fetch the order to verify existence and ownership
	order, err := s.repo.GetOrder(ctx, orderID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return fmt.Errorf("order not found: %w", ErrNotFound)
		}
		return fmt.Errorf("failed to get order: %w", err)
	}

	// Verify client_id ownership
	if order.ClientID != clientID {
		return fmt.Errorf("unauthorized access to order %d: %w", orderID, ErrUnauthorized)
	}

	if err := s.repo.DeleteOrder(ctx, orderID); err != nil {
		return fmt.Errorf("failed to delete order: %w", err)
	}

	return nil
}

// ListMonetarySettlements retrieves a paginated list of monetary settlements for the client.
func (s *Service) ListMonetarySettlements(ctx context.Context, clientID int) ([]*domain.MonetarySettlement, int, error) {
	settlements, total, err := s.repo.ListMonetarySettlements(ctx, clientID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list monetary settlements: %w", err)
	}

	// Verify client_id ownership
	for _, settlement := range settlements {
		if settlement.ClientID != clientID {
			return nil, 0, fmt.Errorf("unauthorized access to settlement %d: %w", settlement.MonetarySettlementID, ErrUnauthorized)
		}
	}

	return settlements, total, nil
}

// CreateMonetarySettlement creates a new monetary settlement for the specified client.
func (s *Service) CreateMonetarySettlement(ctx context.Context, clientID int, req domain.MonetarySettlementCreate) (*domain.MonetarySettlement, error) {
	// Validate input
	if req.Amount <= 0 {
		return nil, fmt.Errorf("amount must be positive: %w", ErrInvalidInput)
	}
	if req.DealID != nil && *req.DealID <= 0 {
		return nil, fmt.Errorf("invalid deal_id: %w", ErrInvalidInput)
	}
	if req.Status != "" && req.Status != domain.StatusPending && req.Status != domain.StatusExecuted && req.Status != domain.StatusCancelled {
		return nil, fmt.Errorf("invalid status: %w", ErrInvalidInput)
	}
	if req.PaymentMethod != nil {
		if *req.PaymentMethod != domain.PaymentMethodCard && *req.PaymentMethod != domain.PaymentMethodBankTransfer && *req.PaymentMethod != domain.PaymentMethodWallet {
			return nil, fmt.Errorf("invalid payment method: %w", ErrInvalidInput)
		}
	}

	// Verify deal exists if provided
	if req.DealID != nil {
		_, err := s.repo.GetDeal(ctx, *req.DealID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return nil, fmt.Errorf("deal not found: %w", ErrNotFound)
			}
			return nil, fmt.Errorf("failed to get deal: %w", err)
		}
	}

	settlement := &domain.MonetarySettlement{
		DealID:        req.DealID,
		Amount:        req.Amount,
		Status:        req.Status,
		PaymentMethod: req.PaymentMethod,
		ClientID:      clientID,
	}

	createdSettlement, err := s.repo.CreateMonetarySettlement(ctx, settlement)
	if err != nil {
		return nil, fmt.Errorf("failed to create monetary settlement: %w", err)
	}

	return createdSettlement, nil
}

// ListClearingTransactions retrieves a paginated list of clearing transactions for the client.
func (s *Service) ListClearingTransactions(ctx context.Context, clientID, page, limit int) ([]*domain.ClearingTransaction, int, error) {
	if page < 1 || limit < 1 {
		return nil, 0, fmt.Errorf("invalid pagination parameters: %w", ErrInvalidInput)
	}

	transactions, total, err := s.repo.ListClearingTransactions(ctx, clientID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list clearing transactions: %w", err)
	}

	// Verify client_id ownership indirectly through related orders or settlements
	for _, tx := range transactions {
		if tx.OrderID != nil {
			order, err := s.repo.GetOrder(ctx, *tx.OrderID)
			if err != nil {
				return nil, 0, fmt.Errorf("failed to get order %d: %w", *tx.OrderID, err)
			}
			if order.ClientID != clientID {
				return nil, 0, fmt.Errorf("unauthorized access to transaction %d: %w", tx.ClearingTransactionID, ErrUnauthorized)
			}
		}
		if tx.MonetarySettlementID != nil {
			settlement, err := s.repo.GetMonetarySettlement(ctx, *tx.MonetarySettlementID)
			if err != nil {
				return nil, 0, fmt.Errorf("failed to get settlement %d: %w", *tx.MonetarySettlementID, err)
			}
			if settlement.ClientID != clientID {
				return nil, 0, fmt.Errorf("unauthorized access to transaction %d: %w", tx.ClearingTransactionID, ErrUnauthorized)
			}
		}
	}

	return transactions, total, nil
}
