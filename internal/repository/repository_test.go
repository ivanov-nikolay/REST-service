package repository

import (
	"regexp"
	"testing"
	"time"

	"github.com/ivanov-nikolay/REST-service/internal/models"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestSubscriptionRepo_Create(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: db}), &gorm.Config{
		SkipDefaultTransaction: true,
	})
	require.NoError(t, err)

	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)
	repo := NewSubscriptionRepo(gormDB, logger)

	sub := &models.Subscription{
		ID:          uuid.New(),
		ServiceName: "YandexMusic",
		Price:       299,
		UserID:      uuid.New(),
		StartDate:   time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:     nil,
	}

	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "subscriptions"`)).WithArgs(
		sub.ServiceName,
		sub.Price,
		sub.UserID,
		sub.StartDate,
		nil,
		sqlmock.AnyArg(),
		sqlmock.AnyArg(),
		sub.ID,
	).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(sub.ID.String()))

	err = repo.Create(sub)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSubscriptionRepo_GetByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: db}), &gorm.Config{
		SkipDefaultTransaction: true,
	})
	require.NoError(t, err)

	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)
	repo := NewSubscriptionRepo(gormDB, logger)

	id := uuid.New()
	rows := sqlmock.NewRows([]string{"id", "service_name", "price", "user_id", "start_date", "end_date", "created_at", "updated_at"}).
		AddRow(id.String(), "YandexMusic", 299, uuid.New().String(), time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), nil, time.Now(), time.Now())

	mock.ExpectQuery("SELECT \\* FROM \"subscriptions\" WHERE id = \\$1 ORDER BY \"subscriptions\"\\.\"id\" LIMIT \\$2").
		WithArgs(id.String(), 1).
		WillReturnRows(rows)

	sub, err := repo.GetByID(id)
	assert.NoError(t, err)
	assert.NotNil(t, sub)
	assert.Equal(t, id, sub.ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSubscriptionRepo_Update(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: db}), &gorm.Config{
		SkipDefaultTransaction: true,
	})
	require.NoError(t, err)

	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)
	repo := NewSubscriptionRepo(gormDB, logger)

	sub := &models.Subscription{
		ID:          uuid.New(),
		ServiceName: "Updated Service",
		Price:       999,
		UserID:      uuid.New(),
		StartDate:   time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:     nil,
	}

	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "subscriptions" SET`)).
		WithArgs(
			sub.ServiceName,
			sub.Price,
			sub.UserID,
			sub.StartDate,
			sub.EndDate,
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			sub.ID,
		).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = repo.Update(sub)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSubscriptionRepo_Delete(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: db}), &gorm.Config{
		SkipDefaultTransaction: true,
	})
	require.NoError(t, err)

	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)
	repo := NewSubscriptionRepo(gormDB, logger)

	id := uuid.New()
	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "subscriptions" WHERE id = $1`)).
		WithArgs(id.String()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = repo.Delete(id)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSubscriptionRepo_List(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: db}), &gorm.Config{
		SkipDefaultTransaction: true,
	})
	require.NoError(t, err)

	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)
	repo := NewSubscriptionRepo(gormDB, logger)

	offset, limit := 0, 10
	userID := uuid.New()
	filters := map[string]interface{}{
		"user_id": userID,
	}

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "subscriptions" WHERE user_id = $1`)).
		WithArgs(userID.String()).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	rows := sqlmock.NewRows([]string{"id", "service_name", "price", "user_id", "start_date", "end_date", "created_at", "updated_at"}).
		AddRow(uuid.New().String(), "ServiceA", 100, userID.String(), time.Now(), nil, time.Now(), time.Now())

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "subscriptions" WHERE user_id = $1 ORDER BY start_date ASC LIMIT $2`)).
		WithArgs(userID.String(), limit).
		WillReturnRows(rows)

	subs, total, err := repo.List(offset, limit, filters)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, subs, 1)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSubscriptionRepo_List_WithOffset(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: db}), &gorm.Config{
		SkipDefaultTransaction: true,
	})
	require.NoError(t, err)

	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)
	repo := NewSubscriptionRepo(gormDB, logger)

	offset, limit := 5, 10
	userID := uuid.New()
	filters := map[string]interface{}{
		"user_id": userID,
	}

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "subscriptions" WHERE user_id = $1`)).
		WithArgs(userID.String()).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	rows := sqlmock.NewRows([]string{"id", "service_name", "price", "user_id", "start_date", "end_date", "created_at", "updated_at"}).
		AddRow(uuid.New().String(), "ServiceA", 100, userID.String(), time.Now(), nil, time.Now(), time.Now())

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "subscriptions" WHERE user_id = $1 ORDER BY start_date ASC LIMIT $2 OFFSET $3`)).
		WithArgs(userID.String(), limit, offset).
		WillReturnRows(rows)

	subs, total, err := repo.List(offset, limit, filters)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, subs, 1)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSubscriptionRepo_GetTotalCost(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: db}), &gorm.Config{
		SkipDefaultTransaction: true,
	})
	require.NoError(t, err)

	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)
	repo := NewSubscriptionRepo(gormDB, logger)

	userID := uuid.New()
	periodStart := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	periodEnd := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)
	serviceName := "Netflix"

	rows := sqlmock.NewRows([]string{"id", "service_name", "price", "user_id", "start_date", "end_date", "created_at", "updated_at"}).
		AddRow(uuid.New().String(), serviceName, 500, userID.String(), time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), nil, time.Now(), time.Now()).
		AddRow(uuid.New().String(), serviceName, 300, userID.String(), time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC), nil, time.Now(), time.Now())

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "subscriptions" WHERE user_id = $1 AND start_date <= $2 AND (end_date IS NULL OR end_date >= $3) AND service_name = $4`)).
		WithArgs(userID.String(), periodEnd, periodStart, serviceName).
		WillReturnRows(rows)

	cost, err := repo.GetTotalCost(userID, &serviceName, periodStart, periodEnd)
	assert.NoError(t, err)
	assert.Greater(t, cost, 0)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSubscriptionRepo_GetTotalCost_NoServiceFilter(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: db}), &gorm.Config{
		SkipDefaultTransaction: true,
	})
	require.NoError(t, err)

	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)
	repo := NewSubscriptionRepo(gormDB, logger)

	userID := uuid.New()
	periodStart := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	periodEnd := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)

	rows := sqlmock.NewRows([]string{"id", "service_name", "price", "user_id", "start_date", "end_date", "created_at", "updated_at"}).
		AddRow(uuid.New().String(), "ServiceA", 100, userID.String(), time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), nil, time.Now(), time.Now())

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "subscriptions" WHERE user_id = $1 AND start_date <= $2 AND (end_date IS NULL OR end_date >= $3)`)).
		WithArgs(userID.String(), periodEnd, periodStart).
		WillReturnRows(rows)

	cost, err := repo.GetTotalCost(userID, nil, periodStart, periodEnd)
	assert.NoError(t, err)
	assert.Greater(t, cost, 0)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestMonthIntersect(t *testing.T) {
	tests := []struct {
		name        string
		subStart    time.Time
		subEnd      *time.Time
		periodStart time.Time
		periodEnd   time.Time
		expected    int
	}{
		{
			name:        "full overlap",
			subStart:    time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			subEnd:      ptrTime(time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)),
			periodStart: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			periodEnd:   time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC),
			expected:    12,
		},
		{
			name:        "partial overlap",
			subStart:    time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC),
			subEnd:      ptrTime(time.Date(2025, 2, 28, 0, 0, 0, 0, time.UTC)),
			periodStart: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			periodEnd:   time.Date(2025, 3, 31, 0, 0, 0, 0, time.UTC),
			expected:    2,
		},
		{
			name:        "no overlap after",
			subStart:    time.Date(2025, 4, 1, 0, 0, 0, 0, time.UTC),
			subEnd:      ptrTime(time.Date(2025, 6, 30, 0, 0, 0, 0, time.UTC)),
			periodStart: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			periodEnd:   time.Date(2025, 3, 31, 0, 0, 0, 0, time.UTC),
			expected:    0,
		},
		{
			name:        "nil end_date (ongoing) - overlap",
			subStart:    time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			subEnd:      nil,
			periodStart: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
			periodEnd:   time.Date(2025, 8, 31, 0, 0, 0, 0, time.UTC),
			expected:    3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := monthIntersect(tt.subStart, tt.subEnd, tt.periodStart, tt.periodEnd)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func ptrTime(t time.Time) *time.Time {
	return &t
}
