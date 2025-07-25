package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/growthfolio/go-priceguard-api/internal/domain/entities"
	"github.com/growthfolio/go-priceguard-api/internal/domain/repositories"
)

type notificationRepository struct {
	db *gorm.DB
}

// NewNotificationRepository creates a new notification repository
func NewNotificationRepository(db *gorm.DB) repositories.NotificationRepository {
	return &notificationRepository{
		db: db,
	}
}

func (r *notificationRepository) Create(ctx context.Context, notification *entities.Notification) error {
	if notification.ID == uuid.Nil {
		notification.ID = uuid.New()
	}
	notification.CreatedAt = time.Now()

	return r.db.WithContext(ctx).Create(notification).Error
}

func (r *notificationRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.Notification, error) {
	var notification entities.Notification
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&notification).Error
	if err != nil {
		return nil, err
	}
	return &notification, nil
}

func (r *notificationRepository) GetByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]entities.Notification, error) {
	var notifications []entities.Notification
	query := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	err := query.Find(&notifications).Error
	return notifications, err
}

func (r *notificationRepository) GetUnread(ctx context.Context, userID uuid.UUID, limit, offset int) ([]entities.Notification, error) {
	var notifications []entities.Notification
	query := r.db.WithContext(ctx).Where("user_id = ? AND read_at IS NULL", userID).Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	err := query.Find(&notifications).Error
	return notifications, err
}

func (r *notificationRepository) MarkAsRead(ctx context.Context, ids []uuid.UUID, userID uuid.UUID) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&entities.Notification{}).
		Where("id IN ? AND user_id = ?", ids, userID).
		Update("read_at", &now).Error
}

func (r *notificationRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&entities.Notification{}, id).Error
}

func (r *notificationRepository) CountByUserID(ctx context.Context, userID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entities.Notification{}).Where("user_id = ?", userID).Count(&count).Error
	return count, err
}

func (r *notificationRepository) CountUnreadByUserID(ctx context.Context, userID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entities.Notification{}).Where("user_id = ? AND read_at IS NULL", userID).Count(&count).Error
	return count, err
}

func (r *notificationRepository) MarkAllAsReadByUserID(ctx context.Context, userID uuid.UUID) (int, error) {
	now := time.Now()
	result := r.db.WithContext(ctx).Model(&entities.Notification{}).
		Where("user_id = ? AND read_at IS NULL", userID).
		Update("read_at", &now)

	return int(result.RowsAffected), result.Error
}

func (r *notificationRepository) Update(ctx context.Context, notification *entities.Notification) error {
	return r.db.WithContext(ctx).Save(notification).Error
}
