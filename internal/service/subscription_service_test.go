package service

import (
	"errors"
	"testing"
	"time"

	"github.com/ivanov-nikolay/REST-service/internal/models"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockSubscriptionRepository struct {
	mock.Mock
}

func (m *MockSubscriptionRepository) Create(sub *models.Subscription) error {
	args := m.Called(sub)
	return args.Error(0)
}

func (m *MockSubscriptionRepository) GetByID(id uuid.UUID) (*models.Subscription, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Subscription), args.Error(1)
}

func (m *MockSubscriptionRepository) Update(sub *models.Subscription) error {
	args := m.Called(sub)
	return args.Error(0)
}

func (m *MockSubscriptionRepository) Delete(id uuid.UUID) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockSubscriptionRepository) List(offset, limit int, filters map[string]interface{}) ([]models.Subscription, int64, error) {
	args := m.Called(offset, limit, filters)
	return args.Get(0).([]models.Subscription), args.Get(1).(int64), args.Error(2)
}

func (m *MockSubscriptionRepository) GetTotalCost(userID uuid.UUID, serviceName *string, periodStart, periodEnd time.Time) (int, error) {
	args := m.Called(userID, serviceName, periodStart, periodEnd)
	return args.Int(0), args.Error(1)
}

func newTestService(repo *MockSubscriptionRepository) SubscriptionService {
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)
	return NewSubscriptionService(repo, logger)
}

func TestSubscriptionService_Create(t *testing.T) {
	mockRepo := new(MockSubscriptionRepository)
	svc := newTestService(mockRepo)

	validSub := &models.Subscription{
		ID:          uuid.New(),
		ServiceName: "Netflix",
		Price:       500,
		UserID:      uuid.New(),
		StartDate:   time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:     nil,
	}

	t.Run("success", func(t *testing.T) {
		mockRepo.On("Create", validSub).Return(nil).Once()

		err := svc.Create(validSub)

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("empty service name", func(t *testing.T) {
		invalidSub := &models.Subscription{
			ServiceName: "   ",
			Price:       100,
			UserID:      uuid.New(),
			StartDate:   time.Now(),
		}
		err := svc.Create(invalidSub)
		assert.EqualError(t, err, "service_name cannot be empty")
	})

	t.Run("negative price", func(t *testing.T) {
		invalidSub := &models.Subscription{
			ServiceName: "Valid",
			Price:       -10,
			UserID:      uuid.New(),
			StartDate:   time.Now(),
		}
		err := svc.Create(invalidSub)
		assert.EqualError(t, err, "price cannot be negative")
	})

	t.Run("end date before start date", func(t *testing.T) {
		start := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
		end := time.Date(2025, 5, 1, 0, 0, 0, 0, time.UTC)
		invalidSub := &models.Subscription{
			ServiceName: "Valid",
			Price:       100,
			UserID:      uuid.New(),
			StartDate:   start,
			EndDate:     &end,
		}
		err := svc.Create(invalidSub)
		assert.EqualError(t, err, "end_date must be after start_date")
	})

	t.Run("repository error", func(t *testing.T) {
		sub := &models.Subscription{
			ServiceName: "Spotify",
			Price:       10,
			UserID:      uuid.New(),
			StartDate:   time.Now(),
		}
		expectedErr := errors.New("database connection lost")
		mockRepo.On("Create", sub).Return(expectedErr).Once()

		err := svc.Create(sub)
		assert.EqualError(t, err, expectedErr.Error())
		mockRepo.AssertExpectations(t)
	})
}

func TestSubscriptionService_GetByID(t *testing.T) {
	mockRepo := new(MockSubscriptionRepository)
	svc := newTestService(mockRepo)

	id := uuid.New()
	expectedSub := &models.Subscription{
		ID:          id,
		ServiceName: "Test",
		Price:       100,
		UserID:      uuid.New(),
		StartDate:   time.Now(),
	}

	t.Run("success", func(t *testing.T) {
		mockRepo.On("GetByID", id).Return(expectedSub, nil).Once()
		sub, err := svc.GetByID(id)
		assert.NoError(t, err)
		assert.Equal(t, expectedSub, sub)
		mockRepo.AssertExpectations(t)
	})

	t.Run("not found", func(t *testing.T) {
		notFoundErr := errors.New("record not found")
		mockRepo.On("GetByID", id).Return(nil, notFoundErr).Once()
		sub, err := svc.GetByID(id)
		assert.Nil(t, sub)
		assert.EqualError(t, err, notFoundErr.Error())
		mockRepo.AssertExpectations(t)
	})
}

func TestSubscriptionService_Update(t *testing.T) {
	mockRepo := new(MockSubscriptionRepository)
	svc := newTestService(mockRepo)

	userID := uuid.New()
	existingSub := &models.Subscription{
		ID:          uuid.New(),
		ServiceName: "Old Name",
		Price:       100,
		UserID:      userID,
		StartDate:   time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:     nil,
	}

	t.Run("success", func(t *testing.T) {
		updatedSub := &models.Subscription{
			ID:          existingSub.ID,
			ServiceName: "New Name",
			Price:       200,
			StartDate:   existingSub.StartDate,
			EndDate:     nil,
		}

		mockRepo.On("GetByID", existingSub.ID).Return(existingSub, nil).Once()
		expectedForUpdate := *updatedSub
		expectedForUpdate.UserID = userID
		mockRepo.On("Update", &expectedForUpdate).Return(nil).Once()

		err := svc.Update(updatedSub)
		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("not found on get", func(t *testing.T) {
		sub := &models.Subscription{ID: uuid.New()}
		notFoundErr := errors.New("record not found")
		mockRepo.On("GetByID", sub.ID).Return(nil, notFoundErr).Once()
		err := svc.Update(sub)
		assert.EqualError(t, err, notFoundErr.Error())
		mockRepo.AssertExpectations(t)
	})

	t.Run("empty service name", func(t *testing.T) {
		sub := &models.Subscription{
			ID:          existingSub.ID,
			ServiceName: "   ",
			Price:       200,
			StartDate:   time.Now(),
		}
		mockRepo.On("GetByID", sub.ID).Return(existingSub, nil).Once()
		err := svc.Update(sub)
		assert.EqualError(t, err, "service_name cannot be empty")
		mockRepo.AssertExpectations(t)
	})

	t.Run("negative price", func(t *testing.T) {
		sub := &models.Subscription{
			ID:          existingSub.ID,
			ServiceName: "Valid",
			Price:       -5,
			StartDate:   time.Now(),
		}
		mockRepo.On("GetByID", sub.ID).Return(existingSub, nil).Once()
		err := svc.Update(sub)
		assert.EqualError(t, err, "price cannot be negative")
		mockRepo.AssertExpectations(t)
	})

	t.Run("invalid dates", func(t *testing.T) {
		start := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
		end := time.Date(2025, 5, 1, 0, 0, 0, 0, time.UTC)
		sub := &models.Subscription{
			ID:          existingSub.ID,
			ServiceName: "Valid",
			Price:       100,
			StartDate:   start,
			EndDate:     &end,
		}
		mockRepo.On("GetByID", sub.ID).Return(existingSub, nil).Once()
		err := svc.Update(sub)
		assert.EqualError(t, err, "end_date must be after start_date")
		mockRepo.AssertExpectations(t)
	})

	t.Run("update error", func(t *testing.T) {
		sub := &models.Subscription{
			ID:          existingSub.ID,
			ServiceName: "New",
			Price:       300,
			StartDate:   existingSub.StartDate,
		}
		expectedErr := errors.New("update conflict")
		mockRepo.On("GetByID", sub.ID).Return(existingSub, nil).Once()
		expectedSub := *sub
		expectedSub.UserID = userID
		mockRepo.On("Update", &expectedSub).Return(expectedErr).Once()
		err := svc.Update(sub)
		assert.EqualError(t, err, expectedErr.Error())
		mockRepo.AssertExpectations(t)
	})
}

func TestSubscriptionService_Delete(t *testing.T) {
	mockRepo := new(MockSubscriptionRepository)
	svc := newTestService(mockRepo)

	id := uuid.New()

	t.Run("success", func(t *testing.T) {
		mockRepo.On("Delete", id).Return(nil).Once()
		err := svc.Delete(id)
		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("repository error", func(t *testing.T) {
		expectedErr := errors.New("delete failed")
		mockRepo.On("Delete", id).Return(expectedErr).Once()
		err := svc.Delete(id)
		assert.EqualError(t, err, expectedErr.Error())
		mockRepo.AssertExpectations(t)
	})
}

func TestSubscriptionService_List(t *testing.T) {
	mockRepo := new(MockSubscriptionRepository)
	svc := newTestService(mockRepo)

	filters := map[string]interface{}{"user_id": uuid.New()}
	expectedSubs := []models.Subscription{
		{ID: uuid.New(), ServiceName: "A", Price: 10},
		{ID: uuid.New(), ServiceName: "B", Price: 20},
	}
	total := int64(2)

	t.Run("normal pagination", func(t *testing.T) {
		page, pageSize := 2, 5
		expectedOffset := (page - 1) * pageSize
		mockRepo.On("List", expectedOffset, pageSize, filters).Return(expectedSubs, total, nil).Once()
		subs, totalCount, err := svc.List(page, pageSize, filters)
		assert.NoError(t, err)
		assert.Equal(t, expectedSubs, subs)
		assert.Equal(t, total, totalCount)
		mockRepo.AssertExpectations(t)
	})

	t.Run("page less than 1", func(t *testing.T) {
		page, pageSize := 0, 10
		mockRepo.On("List", page, pageSize, filters).Return(expectedSubs, total, nil).Once()
		subs, totalCount, err := svc.List(page, pageSize, filters)
		assert.NoError(t, err)
		assert.Equal(t, expectedSubs, subs)
		assert.Equal(t, total, totalCount)
		mockRepo.AssertExpectations(t)
	})

	t.Run("pageSize less than 1", func(t *testing.T) {
		page, pageSize := 1, 0
		mockRepo.On("List", 0, 10, filters).Return(expectedSubs, total, nil).Once()
		subs, totalCount, err := svc.List(page, pageSize, filters)
		assert.NoError(t, err)
		assert.Equal(t, expectedSubs, subs)
		assert.Equal(t, total, totalCount)
		mockRepo.AssertExpectations(t)
	})

	t.Run("repository error", func(t *testing.T) {
		expectedErr := errors.New("query failed")
		mockRepo.On("List",
			0,
			10,
			filters).
			Return([]models.Subscription{}, int64(0), expectedErr).Once()
		subs, totalCount, err := svc.List(1, 10, filters)
		assert.EqualError(t, err, expectedErr.Error())
		assert.Nil(t, subs)
		assert.Equal(t, int64(0), totalCount)
		mockRepo.AssertExpectations(t)
	})
}

func TestSubscriptionService_GetTotalCost(t *testing.T) {
	mockRepo := new(MockSubscriptionRepository)
	svc := newTestService(mockRepo)

	userID := uuid.New()
	serviceName := "Netflix"
	periodStart := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
	periodEnd := time.Date(2025, 12, 20, 0, 0, 0, 0, time.UTC)

	normalizedStart := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	normalizedEnd := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)

	expectedCost := 1200

	t.Run("success with service filter", func(t *testing.T) {
		mockRepo.On("GetTotalCost",
			userID,
			&serviceName,
			normalizedStart,
			normalizedEnd).
			Return(expectedCost, nil).Once()
		cost, err := svc.GetTotalCost(userID, &serviceName, periodStart, periodEnd)
		assert.NoError(t, err)
		assert.Equal(t, expectedCost, cost)
		mockRepo.AssertExpectations(t)
	})

	t.Run("success without service filter", func(t *testing.T) {
		mockRepo.On("GetTotalCost",
			userID,
			(*string)(nil),
			normalizedStart,
			normalizedEnd).
			Return(expectedCost, nil).Once()
		cost, err := svc.GetTotalCost(userID, nil, periodStart, periodEnd)
		assert.NoError(t, err)
		assert.Equal(t, expectedCost, cost)
		mockRepo.AssertExpectations(t)
	})

	t.Run("invalid period", func(t *testing.T) {
		invalidStart := time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC)
		invalidEnd := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		cost, err := svc.GetTotalCost(userID, nil, invalidStart, invalidEnd)
		assert.EqualError(t, err, "period_start must be before or equal to period_end")
		assert.Equal(t, 0, cost)
	})

	t.Run("repository error", func(t *testing.T) {
		expectedErr := errors.New("db error")
		mockRepo.On("GetTotalCost",
			userID,
			&serviceName,
			normalizedStart,
			normalizedEnd).
			Return(0, expectedErr).Once()
		cost, err := svc.GetTotalCost(userID, &serviceName, periodStart, periodEnd)
		assert.EqualError(t, err, expectedErr.Error())
		assert.Equal(t, 0, cost)
		mockRepo.AssertExpectations(t)
	})
}
