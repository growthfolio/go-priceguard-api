package testutils

import (
	"context"
	"time"

	"github.com/google/uuid"
	appservices "github.com/growthfolio/go-priceguard-api/internal/application/services"
	"github.com/growthfolio/go-priceguard-api/internal/domain/entities"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/mock"
)

// MockAlertRepository implements the AlertRepository interface for testing
type MockAlertRepository struct {
	mock.Mock
}

func (m *MockAlertRepository) Create(ctx context.Context, alert *entities.Alert) error {
	args := m.Called(ctx, alert)
	return args.Error(0)
}

func (m *MockAlertRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.Alert, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.Alert), args.Error(1)
}

func (m *MockAlertRepository) GetByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]entities.Alert, error) {
	args := m.Called(ctx, userID, limit, offset)
	return args.Get(0).([]entities.Alert), args.Error(1)
}

func (m *MockAlertRepository) GetEnabled(ctx context.Context) ([]entities.Alert, error) {
	args := m.Called(ctx)
	return args.Get(0).([]entities.Alert), args.Error(1)
}

func (m *MockAlertRepository) Update(ctx context.Context, alert *entities.Alert) error {
	args := m.Called(ctx, alert)
	return args.Error(0)
}

func (m *MockAlertRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockAlertRepository) GetBySymbol(ctx context.Context, symbol string) ([]entities.Alert, error) {
	args := m.Called(ctx, symbol)
	return args.Get(0).([]entities.Alert), args.Error(1)
}

func (m *MockAlertRepository) MarkTriggered(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// MockNotificationRepository implements the NotificationRepository interface for testing
type MockNotificationRepository struct {
	mock.Mock
}

func (m *MockNotificationRepository) Create(ctx context.Context, notification *entities.Notification) error {
	args := m.Called(ctx, notification)
	return args.Error(0)
}

func (m *MockNotificationRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.Notification, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.Notification), args.Error(1)
}

func (m *MockNotificationRepository) GetByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]entities.Notification, error) {
	args := m.Called(ctx, userID, limit, offset)
	return args.Get(0).([]entities.Notification), args.Error(1)
}

func (m *MockNotificationRepository) GetUnread(ctx context.Context, userID uuid.UUID, limit, offset int) ([]entities.Notification, error) {
	args := m.Called(ctx, userID, limit, offset)
	return args.Get(0).([]entities.Notification), args.Error(1)
}

func (m *MockNotificationRepository) MarkAsRead(ctx context.Context, ids []uuid.UUID, userID uuid.UUID) error {
	args := m.Called(ctx, ids, userID)
	return args.Error(0)
}

func (m *MockNotificationRepository) Update(ctx context.Context, notification *entities.Notification) error {
	args := m.Called(ctx, notification)
	return args.Error(0)
}

func (m *MockNotificationRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockNotificationRepository) CountByUserID(ctx context.Context, userID uuid.UUID) (int64, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockNotificationRepository) CountUnreadByUserID(ctx context.Context, userID uuid.UUID) (int64, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockNotificationRepository) MarkAllAsReadByUserID(ctx context.Context, userID uuid.UUID) (int, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(int), args.Error(1)
}

// MockUserRepository implements the UserRepository interface for testing
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(ctx context.Context, user *entities.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.User), args.Error(1)
}

func (m *MockUserRepository) GetByEmail(ctx context.Context, email string) (*entities.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.User), args.Error(1)
}

func (m *MockUserRepository) GetByGoogleID(ctx context.Context, googleID string) (*entities.User, error) {
	args := m.Called(ctx, googleID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.User), args.Error(1)
}

func (m *MockUserRepository) Update(ctx context.Context, user *entities.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// MockPriceHistoryRepository implements the PriceHistoryRepository interface for testing
type MockPriceHistoryRepository struct {
	mock.Mock
}

func (m *MockPriceHistoryRepository) Create(ctx context.Context, history *entities.PriceHistory) error {
	args := m.Called(ctx, history)
	return args.Error(0)
}

func (m *MockPriceHistoryRepository) GetBySymbol(ctx context.Context, symbol, timeframe string, limit int) ([]entities.PriceHistory, error) {
	args := m.Called(ctx, symbol, timeframe, limit)
	return args.Get(0).([]entities.PriceHistory), args.Error(1)
}

func (m *MockPriceHistoryRepository) GetLatest(ctx context.Context, symbol, timeframe string) (*entities.PriceHistory, error) {
	args := m.Called(ctx, symbol, timeframe)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.PriceHistory), args.Error(1)
}

func (m *MockPriceHistoryRepository) BulkInsert(ctx context.Context, histories []entities.PriceHistory) error {
	args := m.Called(ctx, histories)
	return args.Error(0)
}

func (m *MockPriceHistoryRepository) DeleteOld(ctx context.Context, symbol, timeframe string, keepDays int) error {
	args := m.Called(ctx, symbol, timeframe, keepDays)
	return args.Error(0)
}

// MockTechnicalIndicatorRepository implements the TechnicalIndicatorRepository interface for testing
type MockTechnicalIndicatorRepository struct {
	mock.Mock
}

func (m *MockTechnicalIndicatorRepository) Create(ctx context.Context, indicator *entities.TechnicalIndicator) error {
	args := m.Called(ctx, indicator)
	return args.Error(0)
}

func (m *MockTechnicalIndicatorRepository) GetBySymbol(ctx context.Context, symbol, timeframe, indicatorType string, limit int) ([]entities.TechnicalIndicator, error) {
	args := m.Called(ctx, symbol, timeframe, indicatorType, limit)
	return args.Get(0).([]entities.TechnicalIndicator), args.Error(1)
}

func (m *MockTechnicalIndicatorRepository) GetLatest(ctx context.Context, symbol, timeframe, indicatorType string) (*entities.TechnicalIndicator, error) {
	args := m.Called(ctx, symbol, timeframe, indicatorType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.TechnicalIndicator), args.Error(1)
}

func (m *MockTechnicalIndicatorRepository) BulkInsert(ctx context.Context, indicators []entities.TechnicalIndicator) error {
	args := m.Called(ctx, indicators)
	return args.Error(0)
}

func (m *MockTechnicalIndicatorRepository) DeleteOld(ctx context.Context, symbol, timeframe string, keepDays int) error {
	args := m.Called(ctx, symbol, timeframe, keepDays)
	return args.Error(0)
}

// MockAlertWebSocketService implements the AlertWebSocketService interface for testing
type MockAlertWebSocketService struct {
	mock.Mock
}

var _ appservices.AlertWebSocketService = (*MockAlertWebSocketService)(nil)

func (m *MockAlertWebSocketService) BroadcastAlertTriggered(ctx context.Context, alert *entities.Alert, result *appservices.AlertEvaluationResult) error {
	args := m.Called(ctx, alert, result)
	return args.Error(0)
}

func (m *MockAlertWebSocketService) BroadcastNotificationUpdate(ctx context.Context, notification *entities.Notification) error {
	args := m.Called(ctx, notification)
	return args.Error(0)
}

func (m *MockAlertWebSocketService) BroadcastCryptoDataUpdate(ctx context.Context, symbol string, data map[string]interface{}) error {
	args := m.Called(ctx, symbol, data)
	return args.Error(0)
}

func (m *MockAlertWebSocketService) BroadcastSystemAlert(ctx context.Context, alertType, title, message string, data map[string]interface{}) error {
	args := m.Called(ctx, alertType, title, message, data)
	return args.Error(0)
}

func (m *MockAlertWebSocketService) NotifyAlertEvaluation(ctx context.Context, userID uuid.UUID, results []appservices.AlertEvaluationResult) error {
	args := m.Called(ctx, userID, results)
	return args.Error(0)
}

func (m *MockAlertWebSocketService) GetConnectedUsersStats(ctx context.Context) (map[string]interface{}, error) {
	args := m.Called(ctx)
	if args.Get(0) != nil {
		return args.Get(0).(map[string]interface{}), args.Error(1)
	}
	return nil, args.Error(1)
}

// MockRedisClient implements the RedisClientInterface for testing
type MockRedisClient struct {
	mock.Mock
}

func (m *MockRedisClient) LPush(ctx context.Context, key string, values ...interface{}) *redis.IntCmd {
	args := m.Called(ctx, key, values)
	cmd := redis.NewIntCmd(ctx)
	cmd.SetVal(args.Get(0).(int64))
	return cmd
}

func (m *MockRedisClient) BRPop(ctx context.Context, timeout time.Duration, keys ...string) *redis.StringSliceCmd {
	args := m.Called(ctx, timeout, keys)
	cmd := redis.NewStringSliceCmd(ctx)
	if args.Get(0) != nil {
		cmd.SetVal(args.Get(0).([]string))
	}
	return cmd
}

func (m *MockRedisClient) Ping(ctx context.Context) *redis.StatusCmd {
	args := m.Called(ctx)
	cmd := redis.NewStatusCmd(ctx)
	cmd.SetVal(args.String(0))
	return cmd
}

func (m *MockRedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
	args := m.Called(ctx, key, value, expiration)
	cmd := redis.NewStatusCmd(ctx)
	cmd.SetVal(args.String(0))
	return cmd
}

func (m *MockRedisClient) Get(ctx context.Context, key string) *redis.StringCmd {
	args := m.Called(ctx, key)
	cmd := redis.NewStringCmd(ctx)
	if err := args.Error(1); err != nil {
		cmd.SetErr(err)
	} else if args.Get(0) != nil {
		cmd.SetVal(args.String(0))
	}
	return cmd
}

func (m *MockRedisClient) TTL(ctx context.Context, key string) *redis.DurationCmd {
	args := m.Called(ctx, key)
	cmd := redis.NewDurationCmd(ctx, 0)
	if args.Get(0) != nil {
		cmd.SetVal(args.Get(0).(time.Duration))
	}
	return cmd
}

func (m *MockRedisClient) Del(ctx context.Context, keys ...string) *redis.IntCmd {
	args := m.Called(ctx, keys)
	cmd := redis.NewIntCmd(ctx)
	cmd.SetVal(args.Get(0).(int64))
	return cmd
}

func (m *MockRedisClient) Exists(ctx context.Context, keys ...string) *redis.IntCmd {
	args := m.Called(ctx, keys)
	cmd := redis.NewIntCmd(ctx)
	cmd.SetVal(args.Get(0).(int64))
	return cmd
}

func (m *MockRedisClient) Expire(ctx context.Context, key string, expiration time.Duration) *redis.BoolCmd {
	args := m.Called(ctx, key, expiration)
	cmd := redis.NewBoolCmd(ctx)
	cmd.SetVal(args.Bool(0))
	return cmd
}

func (m *MockRedisClient) ZAdd(ctx context.Context, key string, members ...redis.Z) *redis.IntCmd {
	args := m.Called(ctx, key, members)
	cmd := redis.NewIntCmd(ctx)
	if args.Get(0) != nil {
		cmd.SetVal(args.Get(0).(int64))
	}
	return cmd
}

func (m *MockRedisClient) ZCard(ctx context.Context, key string) *redis.IntCmd {
	args := m.Called(ctx, key)
	cmd := redis.NewIntCmd(ctx)
	if args.Get(0) != nil {
		cmd.SetVal(args.Get(0).(int64))
	}
	return cmd
}

func (m *MockRedisClient) ZRem(ctx context.Context, key string, members ...interface{}) *redis.IntCmd {
	args := m.Called(ctx, key, members)
	cmd := redis.NewIntCmd(ctx)
	if args.Get(0) != nil {
		cmd.SetVal(args.Get(0).(int64))
	}
	return cmd
}

func (m *MockRedisClient) ZRemRangeByScore(ctx context.Context, key, min, max string) *redis.IntCmd {
	args := m.Called(ctx, key, min, max)
	cmd := redis.NewIntCmd(ctx)
	if args.Get(0) != nil {
		cmd.SetVal(args.Get(0).(int64))
	}
	return cmd
}

func (m *MockRedisClient) ZRangeByScoreWithScores(ctx context.Context, key string, opt *redis.ZRangeBy) *redis.ZSliceCmd {
	args := m.Called(ctx, key, opt)
	cmd := redis.NewZSliceCmd(ctx)
	if args.Get(0) != nil {
		cmd.SetVal(args.Get(0).([]redis.Z))
	}
	return cmd
}
