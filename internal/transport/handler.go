package transport

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/sirupsen/logrus"

	"cliring/internal/domain"
	"cliring/internal/service"
)

// Handler handles HTTP requests for the Cliring API.
type Handler struct {
	service *service.Service
}

// NewHandler creates a new Handler instance.
func NewHandler(service *service.Service) *Handler {
	return &Handler{
		service: service,
	}
}

// InitRoutes initializes the Gin router with all API routes.
func (h *Handler) InitRoutes() *gin.Engine {
	router := gin.New()

	// Middleware for logging and recovery
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// API version group
	v1 := router.Group("/v1")
	{
		// Middleware for JWT authentication and client_id validation
		v1.Use(h.authMiddleware())

		// Deals endpoints
		deals := v1.Group("/deals")
		{
			// Создает новую сделку для указанного клиента.
			deals.POST("", h.createDeal)
			// Удаляет сделку по её ID для указанного клиента.
			deals.DELETE("/:deal_id", h.deleteDeal)
		}

		// Orders endpoints
		orders := v1.Group("/orders")
		{
			// Возвращает постраничный список всех заказов для указанного клиента.
			orders.GET("", h.listOrders)
			// Создает новый заказ для указанного клиента.
			orders.POST("", h.createOrder)
			// Обновляет данные конкретного заказа по его ID для указанного клиента.
			orders.PUT("/:order_id", h.updateOrder)
			// Удаляет заказ по его ID для указанного клиента.
			orders.DELETE("/:order_id", h.deleteOrder)
		}

		// Monetary Settlements endpoints
		monetarySettlements := v1.Group("/monetary-settlements")
		{
			// Возвращает постраничный список всех денежных расчетов для указанного клиента.
			monetarySettlements.GET("", h.listMonetarySettlements)
		}

		// Clearing Transactions endpoints
		clearingTransactions := v1.Group("/clearing-transactions")
		{
			// Возвращает постраничный список всех клиринговых транзакций для указанного клиента.
			clearingTransactions.GET("", h.listClearingTransactions)
		}
	}

	return router
}

// authMiddleware checks JWT token and client_id query parameter.
func (h *Handler) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check JWT token
		tokenString := c.GetHeader("Authorization")
		if tokenString == "" || len(tokenString) < 7 || tokenString[:7] != "Bearer " {
			h.errorResponse(c, http.StatusUnauthorized, "ERR_UNAUTHORIZED", "Missing or invalid Authorization header")
			c.Abort()
			return
		}

		token, err := jwt.Parse(tokenString[7:], func(token *jwt.Token) (interface{}, error) {
			// Replace with your JWT secret key retrieval logic
			return []byte("your-secret-key"), nil
		})
		if err != nil || !token.Valid {
			h.errorResponse(c, http.StatusUnauthorized, "ERR_UNAUTHORIZED", "Invalid JWT token")
			c.Abort()
			return
		}

		// Extract client_id from token claims
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			h.errorResponse(c, http.StatusUnauthorized, "ERR_UNAUTHORIZED", "Invalid token claims")
			c.Abort()
			return
		}
		tokenClientID, ok := claims["client_id"].(float64) // JWT stores numbers as float64
		if !ok {
			h.errorResponse(c, http.StatusUnauthorized, "ERR_UNAUTHORIZED", "Missing client_id in token")
			c.Abort()
			return
		}

		// Check client_id query parameter
		clientIDStr := c.Query("client_id")
		if clientIDStr == "" {
			h.errorResponse(c, http.StatusBadRequest, "ERR_INVALID_CLIENT_ID", "Missing client_id query parameter")
			c.Abort()
			return
		}
		clientID, err := strconv.Atoi(clientIDStr)
		if err != nil {
			h.errorResponse(c, http.StatusBadRequest, "ERR_INVALID_CLIENT_ID", "Invalid client_id format")
			c.Abort()
			return
		}

		// Verify client_id matches token
		if int(tokenClientID) != clientID {
			h.errorResponse(c, http.StatusUnauthorized, "ERR_UNAUTHORIZED", "client_id in token does not match query parameter")
			c.Abort()
			return
		}

		// Add client_id to context
		ctx := context.WithValue(c.Request.Context(), domain.ClientIDKey{}, clientID)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

// errorResponse sends an error response in the standard format.
func (h *Handler) errorResponse(c *gin.Context, status int, code, message string) {
	c.JSON(status, domain.ErrorResponse{
		Error: domain.ErrorDetail{
			Code:    code,
			Message: message,
		},
	})
}

// handleServiceError maps service errors to HTTP responses.
func (h *Handler) handleServiceError(c *gin.Context, err error) {
	logrus.Error("Service error: ", err)

	switch {
	case errors.Is(err, service.ErrInvalidInput):
		h.errorResponse(c, http.StatusBadRequest, "ERR_INVALID_INPUT", err.Error())
	case errors.Is(err, service.ErrNotFound):
		h.errorResponse(c, http.StatusNotFound, "ERR_NOT_FOUND", err.Error())
	case errors.Is(err, service.ErrUnauthorized):
		h.errorResponse(c, http.StatusUnauthorized, "ERR_UNAUTHORIZED", err.Error())
	default:
		h.errorResponse(c, http.StatusInternalServerError, "ERR_INTERNAL", "Internal server error")
	}
}

// createDeal handles POST /deals.
func (h *Handler) createDeal(c *gin.Context) {
	_, ok := c.Request.Context().Value(domain.ClientIDKey{}).(int)
	if !ok {
		h.errorResponse(c, http.StatusBadRequest, "ERR_INVALID_CLIENT_ID", "Invalid client_id")
		return
	}

	deal, err := h.service.CreateDeal(c.Request.Context())
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, deal)
}

// deleteDeal handles DELETE /deals/{deal_id}.
func (h *Handler) deleteDeal(c *gin.Context) {
	clientID, ok := c.Request.Context().Value(domain.ClientIDKey{}).(int)
	if !ok {
		h.errorResponse(c, http.StatusBadRequest, "ERR_INVALID_CLIENT_ID", "Invalid client_id")
		return
	}

	dealID, err := strconv.Atoi(c.Param("deal_id"))
	if err != nil {
		h.errorResponse(c, http.StatusBadRequest, "ERR_INVALID_INPUT", "Invalid deal_id")
		return
	}

	if err := h.service.DeleteDeal(c.Request.Context(), clientID, dealID); err != nil {
		h.handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Сделка удалена"})
}

// listOrders handles GET /orders.
func (h *Handler) listOrders(c *gin.Context) {
	clientID, ok := c.Request.Context().Value(domain.ClientIDKey{}).(int)
	if !ok {
		h.errorResponse(c, http.StatusBadRequest, "ERR_INVALID_CLIENT_ID", "Invalid client_id")
		return
	}

	orders, total, err := h.service.ListOrders(c.Request.Context(), clientID)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"orders": orders,
		"total":  total,
	})
}

// createOrder handles POST /orders.
func (h *Handler) createOrder(c *gin.Context) {
	clientID, ok := c.Request.Context().Value(domain.ClientIDKey{}).(int)
	if !ok {
		h.errorResponse(c, http.StatusBadRequest, "ERR_INVALID_CLIENT_ID", "Invalid client_id")
		return
	}

	var req domain.OrderCreate
	if err := c.ShouldBindJSON(&req); err != nil {
		h.errorResponse(c, http.StatusBadRequest, "ERR_INVALID_INPUT", "Invalid request body")
		return
	}

	order, err := h.service.CreateOrder(c.Request.Context(), clientID, req)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, order)
}

// updateOrder handles PUT /orders/{order_id}.
func (h *Handler) updateOrder(c *gin.Context) {
	clientID, ok := c.Request.Context().Value(domain.ClientIDKey{}).(int)
	if !ok {
		h.errorResponse(c, http.StatusBadRequest, "ERR_INVALID_CLIENT_ID", "Invalid client_id")
		return
	}

	orderID, err := strconv.Atoi(c.Param("order_id"))
	if err != nil {
		h.errorResponse(c, http.StatusBadRequest, "ERR_INVALID_INPUT", "Invalid order_id")
		return
	}

	var req domain.OrderCreate
	if err := c.ShouldBindJSON(&req); err != nil {
		h.errorResponse(c, http.StatusBadRequest, "ERR_INVALID_INPUT", "Invalid request body")
		return
	}

	order, err := h.service.UpdateOrder(c.Request.Context(), clientID, orderID, req)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, order)
}

// deleteOrder handles DELETE /orders/{order_id}.
func (h *Handler) deleteOrder(c *gin.Context) {
	clientID, ok := c.Request.Context().Value(domain.ClientIDKey{}).(int)
	if !ok {
		h.errorResponse(c, http.StatusBadRequest, "ERR_INVALID_CLIENT_ID", "Invalid client_id")
		return
	}

	orderID, err := strconv.Atoi(c.Param("order_id"))
	if err != nil {
		h.errorResponse(c, http.StatusBadRequest, "ERR_INVALID_INPUT", "Invalid order_id")
		return
	}

	if err := h.service.DeleteOrder(c.Request.Context(), clientID, orderID); err != nil {
		h.handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Заказ удален"})
}

// listMonetarySettlements handles GET /monetary-settlements.
func (h *Handler) listMonetarySettlements(c *gin.Context) {
	clientID, ok := c.Request.Context().Value(domain.ClientIDKey{}).(int)
	if !ok {
		h.errorResponse(c, http.StatusBadRequest, "ERR_INVALID_CLIENT_ID", "Invalid client_id")
		return
	}

	settlements, total, err := h.service.ListMonetarySettlements(c.Request.Context(), clientID)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"settlements": settlements,
		"total":       total,
	})
}

// createMonetarySettlement handles PUT /monetary-settlements.
func (h *Handler) createMonetarySettlement(c *gin.Context) {
	clientID, ok := c.Request.Context().Value(domain.ClientIDKey{}).(int)
	if !ok {
		h.errorResponse(c, http.StatusBadRequest, "ERR_INVALID_CLIENT_ID", "Invalid client_id")
		return
	}

	var req domain.MonetarySettlementCreate
	if err := c.ShouldBindJSON(&req); err != nil {
		h.errorResponse(c, http.StatusBadRequest, "ERR_INVALID_INPUT", "Invalid request body")
		return
	}

	settlement, err := h.service.CreateMonetarySettlement(c.Request.Context(), clientID, req)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, settlement)
}

// listClearingTransactions handles GET /clearing-transactions.
func (h *Handler) listClearingTransactions(c *gin.Context) {
	clientID, ok := c.Request.Context().Value(domain.ClientIDKey{}).(int)
	if !ok {
		h.errorResponse(c, http.StatusBadRequest, "ERR_INVALID_CLIENT_ID", "Invalid client_id")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if page < 1 || limit < 1 {
		h.errorResponse(c, http.StatusBadRequest, "ERR_INVALID_INPUT", "Invalid pagination parameters")
		return
	}

	transactions, total, err := h.service.ListClearingTransactions(c.Request.Context(), clientID, page, limit)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"transactions": transactions,
		"total":        total,
	})
}
