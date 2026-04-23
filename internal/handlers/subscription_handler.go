package handlers

import (
	"errors"
	"gorm.io/gorm"
	"net/http"
	"strconv"
	"time"

	"github.com/ivanov-nikolay/REST-service/internal/models"
	"github.com/ivanov-nikolay/REST-service/internal/service"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

type SubscriptionHandler struct {
	svc       service.SubscriptionService
	log       *logrus.Logger
	validator *validator.Validate
}

func NewSubscriptionHandler(svc service.SubscriptionService, log *logrus.Logger) *SubscriptionHandler {
	return &SubscriptionHandler{
		svc:       svc,
		log:       log,
		validator: validator.New(),
	}
}

type CreateSubscriptionRequest struct {
	ServiceName string  `json:"service_name" validate:"required,min=1"`
	Price       int     `json:"price" validate:"gte=0"`
	UserID      string  `json:"user_id" validate:"required,uuid"`
	StartDate   string  `json:"start_date" validate:"required,datetime=01-2006"`
	EndDate     *string `json:"end_date" validate:"omitempty,datetime=01-2006"`
}

type UpdateSubscriptionRequest struct {
	ServiceName string  `json:"service_name" validate:"required,min=1"`
	Price       int     `json:"price" validate:"gte=0"`
	StartDate   string  `json:"start_date" validate:"required,datetime=01-2006"`
	EndDate     *string `json:"end_date" validate:"omitempty,datetime=01-2006"`
}

// @Summary Create subscription
// @Tags subscriptions
// @Accept json
// @Produce json
// @Param request body CreateSubscriptionRequest true "Subscription data"
// @Success 201 {object} models.Subscription
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /subscriptions [post]
func (h *SubscriptionHandler) Create(c echo.Context) error {
	var req CreateSubscriptionRequest

	if err := c.Bind(&req); err != nil {
		h.log.WithError(err).Error("failed to bind request body")
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	if err := h.validator.Struct(req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	startDate, err := parseMonthYear(req.StartDate)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid start_date format, expected MM-YYYY"})
	}

	var endDate *time.Time
	if req.EndDate != nil {
		ed, err := parseMonthYear(*req.EndDate)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid end_date format, expected MM-YYYY"})
		}
		ed = lastDayOfMonth(ed)
		endDate = &ed
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid user_id"})
	}

	sub := &models.Subscription{
		ServiceName: req.ServiceName,
		Price:       req.Price,
		UserID:      userID,
		StartDate:   startDate,
		EndDate:     endDate,
	}

	if err := h.svc.Create(sub); err != nil {
		h.log.WithError(err).Error("failed to create subscription")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}

	return c.JSON(http.StatusCreated, sub)
}

// @Summary Get subscription by ID
// @Tags subscriptions
// @Produce json
// @Param id path string true "Subscription ID (UUID)"
// @Success 200 {object} models.Subscription
// @Failure 404 {object} map[string]interface{}
// @Router /subscriptions/{id} [get]
func (h *SubscriptionHandler) GetByID(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}

	sub, err := h.svc.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "subscription not found"})
		}
		h.log.WithError(err).Error("failed to get subscription")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal error"})
	}

	return c.JSON(http.StatusOK, sub)
}

// @Summary Update subscription
// @Tags subscriptions
// @Accept json
// @Produce json
// @Param id path string true "Subscription ID"
// @Param request body UpdateSubscriptionRequest true "Updated data"
// @Success 200 {object} models.Subscription
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /subscriptions/{id} [put]
func (h *SubscriptionHandler) Update(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}

	var req UpdateSubscriptionRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	if err := h.validator.Struct(req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	startDate, err := parseMonthYear(req.StartDate)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid start_date format, expected MM-YYYY"})
	}

	var endDate *time.Time

	if req.EndDate != nil {
		ed, err := parseMonthYear(*req.EndDate)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid end_date format, expected MM-YYYY"})
		}
		ed = lastDayOfMonth(ed)
		endDate = &ed
	}

	sub := &models.Subscription{
		ID:          id,
		ServiceName: req.ServiceName,
		Price:       req.Price,
		StartDate:   startDate,
		EndDate:     endDate,
	}

	if err := h.svc.Update(sub); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "subscription not found"})
		}
		return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, sub)
}

// @Summary Delete subscription
// @Tags subscriptions
// @Param id path string true "Subscription ID"
// @Success 204
// @Failure 404 {object} map[string]interface{}
// @Router /subscriptions/{id} [delete]
func (h *SubscriptionHandler) Delete(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}
	if err := h.svc.Delete(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "subscription not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal error"})
	}

	return c.NoContent(http.StatusNoContent)
}

// @Summary List subscriptions with pagination and filters
// @Tags subscriptions
// @Produce json
// @Param page query int false "Page number (default 1)"
// @Param page_size query int false "Page size (default 10)"
// @Param user_id query string false "Filter by user ID (UUID)"
// @Param service_name query string false "Filter by service name"
// @Param start_date_from query string false "Start date from (YYYY-MM)"
// @Param start_date_to query string false "Start date to (YYYY-MM)"
// @Param end_date_from query string false "End date from (YYYY-MM)"
// @Param end_date_to query string false "End date to (YYYY-MM)"
// @Success 200 {object} map[string]interface{}
// @Router /subscriptions [get]
func (h *SubscriptionHandler) List(c echo.Context) error {
	page, _ := strconv.Atoi(c.QueryParam("page"))
	pageSize, _ := strconv.Atoi(c.QueryParam("page_size"))
	filters := make(map[string]interface{})

	if userID := c.QueryParam("user_id"); userID != "" {
		uid, err := uuid.Parse(userID)
		if err == nil {
			filters["user_id"] = uid
		}
	}
	if svc := c.QueryParam("service_name"); svc != "" {
		filters["service_name"] = svc
	}
	if sf := c.QueryParam("start_date_from"); sf != "" {
		if t, err := parseMonthYear(sf); err == nil {
			filters["start_date_from"] = t
		}
	}
	if st := c.QueryParam("start_date_to"); st != "" {
		if t, err := parseMonthYear(st); err == nil {
			filters["start_date_to"] = lastDayOfMonth(t)
		}
	}
	if ef := c.QueryParam("end_date_from"); ef != "" {
		if t, err := parseMonthYear(ef); err == nil {
			filters["end_date_from"] = t
		}
	}
	if et := c.QueryParam("end_date_to"); et != "" {
		if t, err := parseMonthYear(et); err == nil {
			filters["end_date_to"] = lastDayOfMonth(t)
		}
	}

	subs, total, err := h.svc.List(page, pageSize, filters)
	if err != nil {
		h.log.WithError(err).Error("failed to list subscriptions")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal error"})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"items":     subs,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// @Summary Calculate total cost for period
// @Tags subscriptions
// @Produce json
// @Param user_id query string true "User ID (UUID)"
// @Param start_date query string true "Period start (MM-YYYY)"
// @Param end_date query string true "Period end (MM-YYYY)"
// @Param service_name query string false "Service name filter"
// @Success 200 {object} map[string]int
// @Failure 400 {object} map[string]interface{}
// @Router /subscriptions/total-cost [get]
func (h *SubscriptionHandler) GetTotalCost(c echo.Context) error {
	userIDStr := c.QueryParam("user_id")
	if userIDStr == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "user_id is required"})
	}
	userID, err := uuid.Parse(userIDStr)

	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid user_id"})
	}

	startStr := c.QueryParam("start_date")
	endStr := c.QueryParam("end_date")

	if startStr == "" || endStr == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "start and end date are required"})
	}

	periodStart, err := parseMonthYear(startStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid start_date format, expected MM-YYYY"})
	}

	periodEnd, err := parseMonthYear(endStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid end_date format, expected MM-YYYY"})
	}

	periodEnd = lastDayOfMonth(periodEnd)

	var serviceName *string
	if sn := c.QueryParam("service_name"); sn != "" {
		serviceName = &sn
	}

	total, err := h.svc.GetTotalCost(userID, serviceName, periodStart, periodEnd)
	if err != nil {
		h.log.WithError(err).Error("failed to get total cost")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal error"})
	}

	return c.JSON(http.StatusOK, map[string]int{"total_cost": total})
}

func parseMonthYear(s string) (time.Time, error) {
	t, err := time.Parse("01-2006", s)
	if err != nil {
		return time.Time{}, err
	}

	return t, nil
}

func lastDayOfMonth(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month()+1, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, -1)
}
