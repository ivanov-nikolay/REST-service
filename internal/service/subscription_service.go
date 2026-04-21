package service

import (
	"errors"
	"strings"
	"time"

	"github.com/ivanov-nikolay/REST-service/internal/models"
	"github.com/ivanov-nikolay/REST-service/internal/repository"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type SubscriptionService interface {
	Create(sub *models.Subscription) error
	GetByID(id uuid.UUID) (*models.Subscription, error)
	Update(sub *models.Subscription) error
	Delete(id uuid.UUID) error
	List(page, pageSize int, filters map[string]interface{}) ([]models.Subscription, int64, error)
	GetTotalCost(userID uuid.UUID, serviceName *string, periodStart, periodEnd time.Time) (int, error)
}

type subscriptionService struct {
	repo   repository.SubscriptionRepository
	logger *logrus.Logger
}

func NewSubscriptionService(repo repository.SubscriptionRepository, logger *logrus.Logger) SubscriptionService {
	return &subscriptionService{
		repo:   repo,
		logger: logger,
	}
}

func (s *subscriptionService) Create(sub *models.Subscription) error {
	s.logger.WithFields(logrus.Fields{
		"service_name": sub.ServiceName,
		"user_id":      sub.UserID,
		"price":        sub.Price,
		"start_date":   sub.StartDate,
		"end_date":     sub.EndDate,
	}).Debug("creating subscription")

	if strings.TrimSpace(sub.ServiceName) == "" {
		err := errors.New("service_name cannot be empty")
		s.logger.WithError(err).Error("service_name is empty")
		return err
	}

	if sub.Price < 0 {
		err := errors.New("price cannot be negative")
		s.logger.WithError(err).Error("validation error")
		return err
	}

	if sub.EndDate != nil && sub.EndDate.Before(sub.StartDate) {
		err := errors.New("end_date must be after start_date")
		s.logger.WithError(err).Error("validation error")
		return err
	}

	if err := s.repo.Create(sub); err != nil {
		s.logger.WithError(err).Error("failed to create subscription")
		return err
	}

	s.logger.WithField("id", sub.ID).Debug("subscriptions to created successfully")

	return nil
}

func (s *subscriptionService) GetByID(id uuid.UUID) (*models.Subscription, error) {
	s.logger.WithField("id", id).Debug("getting subscription by id")

	sub, err := s.repo.GetByID(id)
	if err != nil {
		s.logger.WithError(err).WithField("id", id).Error("failed to get subscription by id")
		return nil, err
	}

	s.logger.WithField("id", id).Debug("subscription successfully retrieved")

	return sub, nil
}

func (s *subscriptionService) Update(sub *models.Subscription) error {
	s.logger.WithFields(logrus.Fields{
		"id":           sub.ID,
		"service_name": sub.ServiceName,
		"price":        sub.Price,
		"start_date":   sub.StartDate,
		"end_date":     sub.EndDate,
	}).Debug("updating subscription")

	existing, err := s.repo.GetByID(sub.ID)
	if err != nil {
		s.logger.WithError(err).WithField("id", sub.ID).Error("failed to find existing subscription")
		return err
	}

	sub.UserID = existing.UserID

	if strings.TrimSpace(sub.ServiceName) == "" {
		err := errors.New("service_name cannot be empty")
		s.logger.WithError(err).Error("service_name is empty")
		return err
	}

	if sub.Price < 0 {
		err := errors.New("price cannot be negative")
		s.logger.WithError(err).Error("validation failed")
		return err
	}

	if sub.EndDate != nil && sub.EndDate.Before(sub.StartDate) {
		err := errors.New("end_date must be after start_date")
		s.logger.WithError(err).Error("validation failed")
		return err
	}

	if err := s.repo.Update(sub); err != nil {
		s.logger.WithError(err).WithField("id", sub.ID).Error("failed to update subscription")
		return err
	}

	s.logger.WithField("id", sub.ID).Debug("subscription updated successfully")

	return nil
}

func (s *subscriptionService) Delete(id uuid.UUID) error {
	s.logger.WithField("id", id).Debug("deleting subscription")
	if err := s.repo.Delete(id); err != nil {
		s.logger.WithError(err).WithField("id", id).Error("failed to delete subscription")
		return err
	}

	s.logger.WithField("id", id).Debug("subscription deleted successfully")

	return nil
}

func (s *subscriptionService) List(page, pageSize int, filters map[string]interface{}) ([]models.Subscription, int64, error) {
	s.logger.WithFields(logrus.Fields{
		"page":      page,
		"page_size": pageSize,
		"filters":   filters,
	}).Debug("listing subscriptions")
	if page < 1 {
		page = 1
		s.logger.WithField("page", page).Debug("adjusted page to minimum")
	}

	if pageSize < 1 {
		pageSize = 10
		s.logger.WithField("page_size", pageSize).Debug("adjusted page size to default")
	}

	offset := (page - 1) * pageSize

	subs, total, err := s.repo.List(offset, pageSize, filters)
	if err != nil {
		s.logger.WithError(err).Error("failed to list subscriptions")
		return nil, 0, err
	}

	s.logger.WithFields(logrus.Fields{
		"count": len(subs),
		"total": total,
		"page":  page,
	}).Debug("subscriptions listed successfully")

	return subs, total, nil
}

func (s *subscriptionService) GetTotalCost(userID uuid.UUID, serviceName *string, periodStart, periodEnd time.Time) (int, error) {
	s.logger.WithFields(logrus.Fields{
		"user_id":      userID,
		"service_name": serviceName,
		"period_start": periodStart,
		"period_end":   periodEnd,
	}).Debug("calculating total cast")

	if periodStart.After(periodEnd) {
		err := errors.New("period_start must be before or equal to period_end")
		s.logger.WithError(err).Error("invalid period")
		return 0, err
	}

	periodStart = time.Date(periodStart.Year(), periodStart.Month(), 1, 0, 0, 0, 0, time.UTC)
	periodEnd = time.Date(periodEnd.Year(), periodEnd.Month()+1, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, -1)

	s.logger.WithFields(logrus.Fields{
		"normalized_period_start": periodStart,
		"normalized_period_end":   periodEnd,
	}).Debug("normalized period bounds")

	cost, err := s.repo.GetTotalCost(userID, serviceName, periodStart, periodEnd)
	if err != nil {
		s.logger.WithError(err).Error("failed to get total cost")
		return 0, err
	}

	s.logger.WithFields(logrus.Fields{
		"user_id":    userID,
		"total_cost": cost,
	}).Debug("total cost calculated")

	return cost, nil
}
