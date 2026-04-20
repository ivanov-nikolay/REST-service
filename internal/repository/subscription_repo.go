package repository

import (
	"time"

	"github.com/ivanov-nikolay/REST-service/internal/models"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type SubscriptionRepository interface {
	Create(sub *models.Subscription) error
	GetByID(id uuid.UUID) (*models.Subscription, error)
	Update(sub *models.Subscription) error
	Delete(id uuid.UUID) error
	List(offset, limit int, filters map[string]interface{}) ([]models.Subscription, int64, error)
	GetTotalCost(userID uuid.UUID, serviceName *string, periodStart, periodEnd time.Time) (int, error)
}

type subscriptionRepo struct {
	db     *gorm.DB
	logger *logrus.Logger
}

func NewSubscriptionRepo(db *gorm.DB, logger *logrus.Logger) SubscriptionRepository {
	return &subscriptionRepo{
		db:     db,
		logger: logger,
	}
}

func (r *subscriptionRepo) Create(sub *models.Subscription) error {
	r.logger.WithFields(logrus.Fields{
		"service_name": sub.ServiceName,
		"user_id":      sub.UserID,
		"price":        sub.Price,
		"start_date":   sub.StartDate,
		"end_date":     sub.EndDate,
	}).Debug("creating subscription")

	err := r.db.Create(sub).Error
	if err != nil {
		r.logger.WithError(err).Error("failed to create subscription")
		return err
	}

	r.logger.WithField("id", sub.ID).Debug("subscription created successfully")

	return nil
}

func (r *subscriptionRepo) GetByID(id uuid.UUID) (*models.Subscription, error) {
	r.logger.WithField("id", id).Debug("getting subscription by id")

	var sub models.Subscription
	err := r.db.First(&sub, "id = ?", id).Error
	if err != nil {
		r.logger.WithError(err).WithField("id", id).Error("failed to get subscription by id")
		return nil, err
	}

	r.logger.WithField("id", id).Debug("subscription retrieved successfully")

	return &sub, nil
}

func (r *subscriptionRepo) Update(sub *models.Subscription) error {
	r.logger.WithFields(logrus.Fields{
		"id":           sub.ID,
		"service_name": sub.ServiceName,
		"price":        sub.Price,
		"start_date":   sub.StartDate,
		"end_date":     sub.EndDate,
	}).Debug("updating subscription")

	err := r.db.Save(sub).Error
	if err != nil {
		r.logger.WithError(err).WithField("id", sub.ID).Error("failed to update subscription")
		return err
	}

	r.logger.WithField("id", sub.ID).Debug("subscriptions updated successfully")

	return nil
}

func (r *subscriptionRepo) Delete(id uuid.UUID) error {
	r.logger.WithField("id", id).Debug("deleting subscription")

	err := r.db.Delete(&models.Subscription{}, "id = ?", id).Error
	if err != nil {
		r.logger.WithError(err).WithField("id", id).Error("failed to delete subscription")
		return err
	}

	r.logger.WithField("id", id).Debug("subscription deleted successfully")

	return nil

}

func (r *subscriptionRepo) List(offset, limit int, filters map[string]interface{}) ([]models.Subscription, int64, error) {
	r.logger.WithFields(logrus.Fields{
		"offset":  offset,
		"limit":   limit,
		"filters": filters,
	}).Debug("listing subscriptions")

	var subs []models.Subscription
	var total int64

	query := r.db.Model(&models.Subscription{})

	if userID, ok := filters["user_id"]; ok {
		query = query.Where("user_id = ?", userID)
	}
	if serviceName, ok := filters["service_name"]; ok {
		query = query.Where("service_name = ?", serviceName)
	}
	if startFrom, ok := filters["start_date_from"]; ok {
		query = query.Where("start_date >= ?", startFrom)
	}
	if startTo, ok := filters["start_date_to"]; ok {
		query = query.Where("start_date <= ?", startTo)
	}
	if endFrom, ok := filters["end_date_from"]; ok {
		query = query.Where("end_date >= ?", endFrom)
	}
	if endTo, ok := filters["end_date_to"]; ok {
		query = query.Where("end_date <= ?", endTo)
	}

	if err := query.Count(&total).Error; err != nil {
		r.logger.WithError(err).Error("failed to count subscriptions")
		return nil, 0, err
	}

	err := query.
		Offset(offset).Limit(limit).
		Order("start_date ASC").
		Find(&subs).Error
	if err != nil {
		r.logger.WithError(err).Error("failed to list subscriptions")
		return nil, 0, err
	}

	r.logger.WithFields(logrus.Fields{
		"count": len(subs),
		"total": total,
	}).Debug("list subscriptions successfully")

	return subs, total, nil
}

func (r *subscriptionRepo) GetTotalCost(userID uuid.UUID, serviceName *string, periodStart, periodEnd time.Time) (int, error) {
	r.logger.WithFields(logrus.Fields{
		"user_id":      userID,
		"service_name": serviceName,
		"period_start": periodStart,
		"period_end":   periodEnd,
	}).Debug("calculating total cost for subscriptions")

	query := r.db.Model(&models.Subscription{}).
		Where("user_id = ?", userID).
		Where("start_date <= ?", periodEnd).
		Where("end_date IS NULL OR end_date >= ?", periodStart)

	if serviceName != nil {
		query = query.Where("service_name = ?", *serviceName)
	}

	var subs []models.Subscription
	if err := r.db.Find(&subs).Error; err != nil {
		r.logger.WithError(err).Error("failed to query subscriptions for cost calculation")
		return 0, err
	}

	totalCost := 0
	for _, sub := range subs {
		months := monthIntersect(sub.StartDate, sub.EndDate, periodStart, periodEnd)
		contribution := sub.Price * months
		totalCost += contribution
		r.logger.WithFields(logrus.Fields{
			"subscription_id": sub.ID,
			"price":           sub.Price,
			"months":          months,
			"contribution":    contribution,
		}).Debug("subscription contribution to total cost")
	}

	r.logger.WithField("total_cost", totalCost).Debug("total cost calculated successfully")

	return totalCost, nil
}

func monthIntersect(subStart time.Time, subEnd *time.Time, periodStart, periodEnd time.Time) int {
	start := subStart
	end := subEnd
	now := time.Now()

	currentMonthEnd := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, -1)
	if end == nil {
		end = &currentMonthEnd
	}

	intersectStart := maxTime(start, periodStart)
	intersectEnd := minTime(*end, periodEnd)

	if intersectStart.After(intersectEnd) {
		return 0
	}

	months := (intersectEnd.Year()-intersectStart.Year())*12 + int(intersectEnd.Month()) - int(intersectStart.Month()+1)
	if months < 0 {
		months = 0
	}

	return months
}

func maxTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}
