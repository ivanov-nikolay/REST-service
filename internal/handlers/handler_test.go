package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ivanov-nikolay/REST-service/internal/models"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

type MockSubscriptionService struct {
	mock.Mock
}

func (m *MockSubscriptionService) Create(sub *models.Subscription) error {
	args := m.Called(sub)
	return args.Error(0)
}

func (m *MockSubscriptionService) GetByID(id uuid.UUID) (*models.Subscription, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Subscription), args.Error(1)
}

func (m *MockSubscriptionService) Update(sub *models.Subscription) error {
	args := m.Called(sub)
	return args.Error(0)
}

func (m *MockSubscriptionService) Delete(id uuid.UUID) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockSubscriptionService) List(page, pageSize int, filters map[string]interface{}) ([]models.Subscription, int64, error) {
	args := m.Called(page, pageSize, filters)
	return args.Get(0).([]models.Subscription), args.Get(1).(int64), args.Error(2)
}

func (m *MockSubscriptionService) GetTotalCost(userID uuid.UUID, serviceName *string, periodStart, periodEnd time.Time) (int, error) {
	args := m.Called(userID, serviceName, periodStart, periodEnd)
	return args.Int(0), args.Error(1)
}

func newTestHandler(mockSvc *MockSubscriptionService) *SubscriptionHandler {
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)
	return &SubscriptionHandler{
		svc:       mockSvc,
		log:       logger,
		validator: validator.New(),
	}
}

func newEchoContext(method, path string, body interface{}) (echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	var req *http.Request
	if body != nil {
		jsonBody, _ := json.Marshal(body)
		req = httptest.NewRequest(method, path, bytes.NewReader(jsonBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	return c, rec
}

func setParam(c echo.Context, key, value string) {
	c.SetParamNames(key)
	c.SetParamValues(value)
}

func addQueryParam(req *http.Request, key, value string) {
	q := req.URL.Query()
	q.Add(key, value)
	req.URL.RawQuery = q.Encode()
}

func TestSubscriptionHandler_Create(t *testing.T) {
	mockSvc := new(MockSubscriptionService)
	handler := newTestHandler(mockSvc)

	validUserID := uuid.New()
	validReq := CreateSubscriptionRequest{
		ServiceName: "Yandex Plus",
		Price:       400,
		UserID:      validUserID.String(),
		StartDate:   "07-2025",
		EndDate:     nil,
	}

	t.Run("success - 201 Created", func(t *testing.T) {
		mockSvc.On("Create",
			mock.AnythingOfType("*models.Subscription")).
			Return(nil).Once()
		ctx, rec := newEchoContext(http.MethodPost, "/subscriptions", validReq)
		err := handler.Create(ctx)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusCreated, rec.Code)

		var resp models.Subscription
		err = json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, validReq.ServiceName, resp.ServiceName)
		assert.Equal(t, validReq.Price, resp.Price)
		assert.Equal(t, validUserID, resp.UserID)
		assert.Equal(t, time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC), resp.StartDate)
		assert.Nil(t, resp.EndDate)
		mockSvc.AssertExpectations(t)
	})

	t.Run("invalid request body - 400", func(t *testing.T) {
		invalidBody := []byte(`{"service_name": "Netflix", "price": "not_a_number"}`)
		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/subscriptions", bytes.NewReader(invalidBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		ctx := e.NewContext(req, rec)
		err := handler.Create(ctx)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		var resp map[string]string
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.Equal(t, "invalid request", resp["error"])
		mockSvc.AssertNotCalled(t, "Create")
	})

	t.Run("validation error - 400", func(t *testing.T) {
		invalidReq := validReq
		invalidReq.ServiceName = ""
		ctx, rec := newEchoContext(http.MethodPost, "/subscriptions", invalidReq)
		err := handler.Create(ctx)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		var resp map[string]string
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.Contains(t, resp["error"], "ServiceName")
	})

	t.Run("invalid start_date format - 400", func(t *testing.T) {
		invalidReq := validReq
		invalidReq.StartDate = "2025-07"
		ctx, rec := newEchoContext(http.MethodPost, "/subscriptions", invalidReq)
		err := handler.Create(ctx)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		var resp map[string]string
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.Contains(t, resp["error"], "StartDate")
	})

	t.Run("invalid user_id - 400", func(t *testing.T) {
		invalidReq := validReq
		invalidReq.UserID = "not-a-uuid"
		ctx, rec := newEchoContext(http.MethodPost, "/subscriptions", invalidReq)
		err := handler.Create(ctx)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		var resp map[string]string
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.Contains(t, resp["error"], "UserID")
	})

	t.Run("service error - 500", func(t *testing.T) {
		mockSvc.On("Create",
			mock.AnythingOfType("*models.Subscription")).
			Return(errors.New("db error")).Once()
		ctx, rec := newEchoContext(http.MethodPost, "/subscriptions", validReq)
		err := handler.Create(ctx)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		var resp map[string]string
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.Equal(t, "internal server error", resp["error"])
		mockSvc.AssertExpectations(t)
	})
}

func TestSubscriptionHandler_GetByID(t *testing.T) {
	mockSvc := new(MockSubscriptionService)
	handler := newTestHandler(mockSvc)

	id := uuid.New()
	expectedSub := &models.Subscription{
		ID:          id,
		ServiceName: "Spotify",
		Price:       299,
		UserID:      uuid.New(),
		StartDate:   time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:     nil,
	}

	t.Run("success - 200 OK", func(t *testing.T) {
		mockSvc.On("GetByID", id).Return(expectedSub, nil).Once()
		ctx, rec := newEchoContext(http.MethodGet, "/subscriptions/"+id.String(), nil)
		setParam(ctx, "id", id.String())
		err := handler.GetByID(ctx)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		var resp models.Subscription
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.Equal(t, expectedSub.ID, resp.ID)
		assert.Equal(t, expectedSub.ServiceName, resp.ServiceName)
		mockSvc.AssertExpectations(t)
	})

	t.Run("invalid uuid format - 400", func(t *testing.T) {
		ctx, rec := newEchoContext(http.MethodGet, "/subscriptions/invalid", nil)
		setParam(ctx, "id", "not-a-uuid")
		err := handler.GetByID(ctx)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		var resp map[string]string
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.Equal(t, "invalid id", resp["error"])
	})

	t.Run("subscription not found - 404", func(t *testing.T) {
		mockSvc.On("GetByID", id).Return(nil, gorm.ErrRecordNotFound).Once()
		ctx, rec := newEchoContext(http.MethodGet, "/subscriptions/"+id.String(), nil)
		setParam(ctx, "id", id.String())
		err := handler.GetByID(ctx)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, rec.Code)
		var resp map[string]string
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.Equal(t, "subscription not found", resp["error"])
		mockSvc.AssertExpectations(t)
	})

	t.Run("service error - 500", func(t *testing.T) {
		mockSvc.On("GetByID", id).Return(nil, errors.New("internal error")).Once()
		ctx, rec := newEchoContext(http.MethodGet, "/subscriptions/"+id.String(), nil)
		setParam(ctx, "id", id.String())
		err := handler.GetByID(ctx)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		var resp map[string]string
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.Equal(t, "internal error", resp["error"])
		mockSvc.AssertExpectations(t)
	})
}

func TestSubscriptionHandler_Update(t *testing.T) {
	mockSvc := new(MockSubscriptionService)
	handler := newTestHandler(mockSvc)

	id := uuid.New()
	validReq := UpdateSubscriptionRequest{
		ServiceName: "Updated Service",
		Price:       599,
		StartDate:   "08-2025",
		EndDate:     nil,
	}

	t.Run("success - 200 OK", func(t *testing.T) {
		mockSvc.On("Update",
			mock.AnythingOfType("*models.Subscription")).
			Return(nil).Once()
		ctx, rec := newEchoContext(http.MethodPut, "/subscriptions/"+id.String(), validReq)
		setParam(ctx, "id", id.String())
		err := handler.Update(ctx)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		var resp models.Subscription
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.Equal(t, id, resp.ID)
		assert.Equal(t, validReq.ServiceName, resp.ServiceName)
		mockSvc.AssertExpectations(t)
	})

	t.Run("invalid id - 400", func(t *testing.T) {
		ctx, rec := newEchoContext(http.MethodPut, "/subscriptions/invalid", validReq)
		setParam(ctx, "id", "not-a-uuid")
		err := handler.Update(ctx)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		var resp map[string]string
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.Equal(t, "invalid id", resp["error"])
	})

	t.Run("validation error - 400", func(t *testing.T) {
		invalidReq := validReq
		invalidReq.Price = -100
		ctx, rec := newEchoContext(http.MethodPut, "/subscriptions/"+id.String(), invalidReq)
		setParam(ctx, "id", id.String())
		err := handler.Update(ctx)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		var resp map[string]string
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.Contains(t, resp["error"], "Price")
	})

	t.Run("subscription not found - 404", func(t *testing.T) {
		mockSvc.On("Update",
			mock.AnythingOfType("*models.Subscription")).
			Return(gorm.ErrRecordNotFound).Once()
		ctx, rec := newEchoContext(http.MethodPut, "/subscriptions/"+id.String(), validReq)
		setParam(ctx, "id", id.String())
		err := handler.Update(ctx)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, rec.Code)
		var resp map[string]string
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.Equal(t, "subscription not found", resp["error"])
		mockSvc.AssertExpectations(t)
	})
}

func TestSubscriptionHandler_Delete(t *testing.T) {
	mockSvc := new(MockSubscriptionService)
	handler := newTestHandler(mockSvc)

	id := uuid.New()

	t.Run("success - 204 No Content", func(t *testing.T) {
		mockSvc.On("Delete", id).Return(nil).Once()
		ctx, rec := newEchoContext(http.MethodDelete, "/subscriptions/"+id.String(), nil)
		setParam(ctx, "id", id.String())
		err := handler.Delete(ctx)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusNoContent, rec.Code)
		mockSvc.AssertExpectations(t)
	})

	t.Run("invalid id - 400", func(t *testing.T) {
		ctx, rec := newEchoContext(http.MethodDelete, "/subscriptions/invalid", nil)
		setParam(ctx, "id", "bad-uuid")
		err := handler.Delete(ctx)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		var resp map[string]string
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.Equal(t, "invalid id", resp["error"])
	})

	t.Run("subscription not found - 404", func(t *testing.T) {
		mockSvc.On("Delete", id).Return(gorm.ErrRecordNotFound).Once()
		ctx, rec := newEchoContext(http.MethodDelete, "/subscriptions/"+id.String(), nil)
		setParam(ctx, "id", id.String())
		err := handler.Delete(ctx)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, rec.Code)
		var resp map[string]string
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.Equal(t, "subscription not found", resp["error"])
		mockSvc.AssertExpectations(t)
	})

	t.Run("service error - 500", func(t *testing.T) {
		mockSvc.On("Delete", id).Return(errors.New("db error")).Once()
		ctx, rec := newEchoContext(http.MethodDelete, "/subscriptions/"+id.String(), nil)
		setParam(ctx, "id", id.String())
		err := handler.Delete(ctx)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		var resp map[string]string
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.Equal(t, "internal error", resp["error"])
		mockSvc.AssertExpectations(t)
	})
}

func TestSubscriptionHandler_List(t *testing.T) {
	mockSvc := new(MockSubscriptionService)
	handler := newTestHandler(mockSvc)

	userID := uuid.New()
	expectedSubs := []models.Subscription{
		{ID: uuid.New(), ServiceName: "A", Price: 10},
		{ID: uuid.New(), ServiceName: "B", Price: 20},
	}
	total := int64(2)

	t.Run("success with filters - 200 OK", func(t *testing.T) {
		mockSvc.On("List",
			mock.AnythingOfType("int"),
			mock.AnythingOfType("int"),
			mock.AnythingOfType("map[string]interface {}")).
			Return(expectedSubs, total, nil).Once()

		req := httptest.NewRequest(http.MethodGet, "/subscriptions", nil)
		addQueryParam(req, "user_id", userID.String())
		addQueryParam(req, "service_name", "Netflix")
		addQueryParam(req, "start_date_from", "01-2025")
		addQueryParam(req, "start_date_to", "12-2025")
		rec := httptest.NewRecorder()
		e := echo.New()
		ctx := e.NewContext(req, rec)

		err := handler.List(ctx)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		var resp map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.Equal(t, float64(total), resp["total"])
		assert.Len(t, resp["items"], 2)
		mockSvc.AssertExpectations(t)
	})

	t.Run("service error - 500", func(t *testing.T) {
		mockSvc.On("List",
			mock.Anything,
			mock.Anything,
			mock.Anything).
			Return([]models.Subscription{}, int64(0), errors.New("db error")).Once()
		ctx, rec := newEchoContext(http.MethodGet, "/subscriptions", nil)
		err := handler.List(ctx)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		var resp map[string]string
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.Equal(t, "internal error", resp["error"])
		mockSvc.AssertExpectations(t)
	})
}

func TestSubscriptionHandler_GetTotalCost(t *testing.T) {
	mockSvc := new(MockSubscriptionService)
	handler := newTestHandler(mockSvc)

	userID := uuid.New()
	serviceName := "Netflix"

	t.Run("success - 200 OK", func(t *testing.T) {
		mockSvc.On("GetTotalCost",
			userID,
			&serviceName,
			mock.AnythingOfType("time.Time"),
			mock.AnythingOfType("time.Time")).
			Return(1200, nil).Once()

		req := httptest.NewRequest(http.MethodGet, "/subscriptions/total-cost", nil)
		addQueryParam(req, "user_id", userID.String())
		addQueryParam(req, "service_name", serviceName)
		addQueryParam(req, "start_date", "01-2025")
		addQueryParam(req, "end_date", "12-2025")
		rec := httptest.NewRecorder()
		e := echo.New()
		ctx := e.NewContext(req, rec)

		err := handler.GetTotalCost(ctx)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		var resp map[string]int
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.Equal(t, 1200, resp["total_cost"])
		mockSvc.AssertExpectations(t)
	})

	t.Run("missing user_id - 400", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/subscriptions/total-cost", nil)
		addQueryParam(req, "start_date", "01-2025")
		addQueryParam(req, "end_date", "12-2025")
		rec := httptest.NewRecorder()
		e := echo.New()
		ctx := e.NewContext(req, rec)
		err := handler.GetTotalCost(ctx)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		var resp map[string]string
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.Equal(t, "user_id is required", resp["error"])
	})

	t.Run("invalid user_id - 400", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/subscriptions/total-cost", nil)
		addQueryParam(req, "user_id", "bad-uuid")
		addQueryParam(req, "start_date", "01-2025")
		addQueryParam(req, "end_date", "12-2025")
		rec := httptest.NewRecorder()
		e := echo.New()
		ctx := e.NewContext(req, rec)
		err := handler.GetTotalCost(ctx)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		var resp map[string]string
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.Equal(t, "invalid user_id", resp["error"])
	})

	t.Run("missing dates - 400", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/subscriptions/total-cost", nil)
		addQueryParam(req, "user_id", userID.String())
		rec := httptest.NewRecorder()
		e := echo.New()
		ctx := e.NewContext(req, rec)
		err := handler.GetTotalCost(ctx)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		var resp map[string]string
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.Equal(t, "start and end date are required", resp["error"])
	})

	t.Run("invalid date format - 400", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/subscriptions/total-cost", nil)
		addQueryParam(req, "user_id", userID.String())
		addQueryParam(req, "start_date", "2025-01")
		addQueryParam(req, "end_date", "12-2025")
		rec := httptest.NewRecorder()
		e := echo.New()
		ctx := e.NewContext(req, rec)
		err := handler.GetTotalCost(ctx)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		var resp map[string]string
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.Contains(t, resp["error"], "invalid start_date format")
	})

	t.Run("service error - 500", func(t *testing.T) {
		mockSvc.On("GetTotalCost",
			userID, (*string)(nil),
			mock.Anything,
			mock.Anything).
			Return(0, errors.New("calculation failed")).Once()
		req := httptest.NewRequest(http.MethodGet, "/subscriptions/total-cost", nil)
		addQueryParam(req, "user_id", userID.String())
		addQueryParam(req, "start_date", "01-2025")
		addQueryParam(req, "end_date", "12-2025")
		rec := httptest.NewRecorder()
		e := echo.New()
		ctx := e.NewContext(req, rec)
		err := handler.GetTotalCost(ctx)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		var resp map[string]string
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.Equal(t, "internal error", resp["error"])
		mockSvc.AssertExpectations(t)
	})
}

func TestParseMonthYear(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  time.Time
		expectErr bool
	}{
		{
			name:      "valid",
			input:     "11-2025",
			expected:  time.Date(2025, 11, 1, 0, 0, 0, 0, time.UTC),
			expectErr: false,
		},
		{
			name:      "valid with leading zero",
			input:     "01-2023",
			expected:  time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			expectErr: false,
		},
		{
			name:      "invalid format",
			input:     "2025-07",
			expected:  time.Time{},
			expectErr: true,
		},
		{
			name:      "invalid month",
			input:     "13-2025",
			expected:  time.Time{},
			expectErr: true,
		},
		{
			name:      "invalid year",
			input:     "07-25",
			expected:  time.Time{},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseMonthYear(tt.input)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, got)
			}
		})
	}
}

func TestLastDayOfMonth(t *testing.T) {
	tests := []struct {
		name     string
		input    time.Time
		expected time.Time
	}{
		{
			name:     "January",
			input:    time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			expected: time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "February non-leap",
			input:    time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
			expected: time.Date(2025, 2, 28, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "February leap",
			input:    time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
			expected: time.Date(2024, 2, 29, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "December",
			input:    time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC),
			expected: time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := lastDayOfMonth(tt.input)
			assert.Equal(t, tt.expected, got)
		})
	}
}
