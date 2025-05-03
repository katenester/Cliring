package service

import (
	"cliring/internal/repository"
	"context"
	"errors"
	"fmt"
	"time"

	"cliring/internal/domain"
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

// CreateDeal creates a new deal.
func (s *Service) CreateDeal(ctx context.Context, req domain.Deal) (*domain.Deal, error) {
	// Validate input
	if req.DealershipID <= 0 {
		return nil, fmt.Errorf("invalid dealership_id: %w", ErrInvalidInput)
	}
	if req.ManagerID <= 0 {
		return nil, fmt.Errorf("invalid manager_id: %w", ErrInvalidInput)
	}
	if req.ClientID <= 0 {
		return nil, fmt.Errorf("invalid client_id: %w", ErrInvalidInput)
	}

	createdDeal, err := s.repo.CreateDeal(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create deal: %w", err)
	}

	return createdDeal, nil
}

// DeleteDeal deletes a deal.
func (s *Service) DeleteDeal(ctx context.Context, dealID int) error {
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
	if clientID <= 0 {
		return nil, 0, fmt.Errorf("invalid client_id: %w", ErrInvalidInput)
	}

	orders, total, err := s.repo.ListOrders(ctx, clientID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list orders: %w", err)
	}

	return orders, total, nil
}

// CreateOrders creates new orders for the specified client.
func (s *Service) CreateOrders(ctx context.Context, clientID int, req []domain.OrderCreate) ([]*domain.Order, error) {
	if clientID <= 0 {
		return nil, fmt.Errorf("invalid client_id: %w", ErrInvalidInput)
	}

	var createdOrders []*domain.Order
	for _, orderReq := range req {
		// Validate input
		if orderReq.Amount <= 0 {
			return nil, fmt.Errorf("amount must be positive: %w", ErrInvalidInput)
		}
		if orderReq.DealID <= 0 {
			return nil, fmt.Errorf("invalid deal_id: %w", ErrInvalidInput)
		}
		if orderReq.OrderTypeID <= 0 {
			return nil, fmt.Errorf("invalid order_type_id: %w", ErrInvalidInput)
		}
		if orderReq.BankID != nil && *orderReq.BankID <= 0 {
			return nil, fmt.Errorf("invalid bank_id: %w", ErrInvalidInput)
		}

		// Verify deal exists
		_, err := s.repo.GetDeal(ctx, orderReq.DealID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return nil, fmt.Errorf("deal not found: %w", ErrNotFound)
			}
			return nil, fmt.Errorf("failed to get deal: %w", err)
		}

		order := &domain.Order{
			DealID:          orderReq.DealID,
			OrderTypeID:     orderReq.OrderTypeID,
			Amount:          orderReq.Amount,
			Status:          domain.StatusPending, // Default status
			NeedAndOrdersID: orderReq.NeedAndOrdersID,
			BankID:          orderReq.BankID,
		}

		createdOrder, err := s.repo.CreateOrder(ctx, order)
		if err != nil {
			return nil, fmt.Errorf("failed to create order: %w", err)
		}
		createdOrders = append(createdOrders, createdOrder)
	}

	return createdOrders, nil
}

// UpdateOrder updates an existing order.
func (s *Service) UpdateOrder(ctx context.Context, clientID, orderID int, req domain.OrderCreate) (*domain.Order, error) {
	if clientID <= 0 {
		return nil, fmt.Errorf("invalid client_id: %w", ErrInvalidInput)
	}

	// Fetch the order to verify existence
	order, err := s.repo.GetOrder(ctx, orderID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, fmt.Errorf("order not found: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	// Validate input
	if req.Amount <= 0 {
		return nil, fmt.Errorf("amount must be positive: %w", ErrInvalidInput)
	}
	if req.DealID <= 0 {
		return nil, fmt.Errorf("invalid deal_id: %w", ErrInvalidInput)
	}
	if req.OrderTypeID <= 0 {
		return nil, fmt.Errorf("invalid order_type_id: %w", ErrInvalidInput)
	}
	if req.BankID != nil && *req.BankID <= 0 {
		return nil, fmt.Errorf("invalid bank_id: %w", ErrInvalidInput)
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
	order.OrderTypeID = req.OrderTypeID
	order.Amount = req.Amount
	order.NeedAndOrdersID = req.NeedAndOrdersID
	order.BankID = req.BankID

	updatedOrder, err := s.repo.UpdateOrder(ctx, order)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, fmt.Errorf("order not found: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("failed to update order: %w", err)
	}

	return updatedOrder, nil
}

//// ListMonetarySettlements retrieves a paginated list of monetary settlements for the deal.
//func (s *Service) ListMonetarySettlements(ctx context.Context, dealID int) ([]*domain.MonetarySettlement, int, error) {
//	if dealID <= 0 {
//		return nil, 0, fmt.Errorf("invalid deal_id: %w", ErrInvalidInput)
//	}
//
//	// Получить взаиморасчёты с типом заказ в рамках сделки
//	settlements, err := s.repo.ListOrdersByDeals(ctx, dealID)
//	if err != nil {
//		return nil, 0, fmt.Errorf("failed to list monetary settlements: %w", err)
//	}
//	// Построение матрицы обязательств
//	// Если нет ни одного заказа, у которого bank_id не 0,то двухсторонний нетто расчёт (клиент, рольф)
//	// Если есть заказ с bank_id не 0 => многосторонний нетто расчёт (клиент, рольф, банк)
//
//	// Расчёт чистых позиций для каждого участница клиринга (сумма a_ij - сумма a_ji)
//
//	// Отдаём MonetarySettlement , где Amount - задолжность рольфу (от клиента и банка)
//	return monetary_settlement, total, nil
//}

// ListMonetarySettlements performs a netting calculation (bilateral or multilateral) based on orders for a deal.
func (s *Service) ListMonetarySettlements(ctx context.Context, dealID int) ([]*domain.MonetarySettlement, error) {
	if dealID <= 0 {
		return nil, fmt.Errorf("invalid deal_id: %w", ErrInvalidInput)
	}

	// Получить взаиморасчёты с типом заказ в рамках сделки
	orders, err := s.repo.ListOrdersByDeals(ctx, dealID)
	if err != nil {
		return nil, fmt.Errorf("failed to list orders: %w", err)
	}

	// Проверка на многосторонний нетто-расчёт
	hasBank := false
	for _, order := range orders {
		if order.BankID != nil {
			hasBank = true
			break
		}
	}

	// Участники: Клиент (C), Дилерский центр (R), Банк (B) - опционально
	participants := []string{"Client", "Rolf"}
	if hasBank {
		participants = append(participants, "Bank")
	}
	n := len(participants)

	// Составление матрицы обязательств: obligations[i][j] - это сумма, которую участник i должен участнику j
	obligations := make([][]float64, n)
	for i := range obligations {
		obligations[i] = make([]float64, n)
	}

	// Построение матрицы обязательств на основе order_type_id
	for _, order := range orders {
		amount := order.Amount
		switch order.OrderTypeID {
		case 1: // Покупка: Клиент должен Дилерскому центру
			obligations[0][1] += amount // C -> R
		case 2: // Кредит: Банк должен Клиенту
			// (задолжность Клиента перед Банком не отображается, так как выходит за рамки сделки)
			//При этом кредитные средства выделяются именно клиенту, а не Рольфу, так как расчеты Банка с Рольфом также выходят за рамки сделки.
			if order.BankID != nil {
				obligations[2][0] += amount // B -> C
			}
		case 3: // Трейд-ин: Дилерский центр должен Клиенту
			obligations[1][0] += amount // Дилерский центр -> Клиент
		default:
			return nil, fmt.Errorf("unknown order_type_id %d: %w", order.OrderTypeID, ErrInvalidInput)
		}
	}

	// Рассчёт чистых позиций: net[i] = sum(a_ij) - sum(a_ji)
	netPositions := make([]float64, n)
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			if i != j {
				netPositions[i] += obligations[i][j]
				netPositions[i] -= obligations[j][i]
			}
		}
	}

	// Создание денежных расчетов по ненулевым чистым позициям
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
	return settlements, nil
}

//// ListMonetarySettlements retrieves a paginated list of monetary settlements for the deal.
//func (s *Service) ListMonetarySettlements(ctx context.Context, dealID int) ([]*domain.MonetarySettlement, int, error) {
//	if dealID <= 0 {
//		return nil, 0, fmt.Errorf("invalid deal_id: %w", ErrInvalidInput)
//	}
//
//	settlements, total, err := s.repo.ListMonetarySettlements(ctx, dealID)
//	if err != nil {
//		return nil, 0, fmt.Errorf("failed to list monetary settlements: %w", err)
//	}
//
//	return settlements, total, nil
//}

//// CreateMonetarySettlement creates a new monetary settlement.
//func (s *Service) CreateMonetarySettlement(ctx context.Context, req domain.MonetarySettlementCreate) (*domain.MonetarySettlement, error) {
//	// Validate input
//	if req.Amount <= 0 {
//		return nil, fmt.Errorf("amount must be positive: %w", ErrInvalidInput)
//	}
//	if req.DealID != nil && *req.DealID <= 0 {
//		return nil, fmt.Errorf("invalid deal_id: %w", ErrInvalidInput)
//	}
//	if req.BankID != nil && *req.BankID <= 0 {
//		return nil, fmt.Errorf("invalid bank_id: %w", ErrInvalidInput)
//	}
//
//	// Verify deal exists if provided
//	if req.DealID != nil {
//		_, err := s.repo.GetDeal(ctx, *req.DealID)
//		if err != nil {
//			if errors.Is(err, repository.ErrNotFound) {
//				return nil, fmt.Errorf("deal not found: %w", ErrNotFound)
//			}
//			return nil, fmt.Errorf("failed to get deal: %w", err)
//		}
//	}
//
//	settlement := &domain.MonetarySettlement{
//		DealID: req.DealID,
//		Amount: req.Amount,
//		Status: domain.StatusPending, // Default status
//		BankID: req.BankID,
//	}
//
//	createdSettlement, err := s.repo.CreateMonetarySettlement(ctx, settlement)
//	if err != nil {
//		return nil, fmt.Errorf("failed to create monetary settlement: %w", err)
//	}
//
//	return createdSettlement, nil
//}
