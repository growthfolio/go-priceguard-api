package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/growthfolio/go-priceguard-api/internal/adapters/http/handlers"
	"github.com/growthfolio/go-priceguard-api/internal/domain/entities"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
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

func (m *MockAlertRepository) GetBySymbol(ctx context.Context, symbol string) ([]entities.Alert, error) {
	args := m.Called(ctx, symbol)
	return args.Get(0).([]entities.Alert), args.Error(1)
}

func (m *MockAlertRepository) MarkTriggered(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func TestAlertHandler_Creation(t *testing.T) {
	// Setup mock
	mockRepo := &MockAlertRepository{}

	// Create handler
	handler := handlers.NewAlertHandler(mockRepo, nil, nil)

	assert.NotNil(t, handler)
}

func TestAlertHandler_GetAlerts_NoAuth(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	mockRepo := &MockAlertRepository{}
	handler := handlers.NewAlertHandler(mockRepo, nil, nil)

	// Create router and route
	router := gin.New()
	router.GET("/alerts", handler.GetAlerts)

	// Create request without authentication
	req, _ := http.NewRequest("GET", "/alerts", nil)
	w := httptest.NewRecorder()

	// Execute
	router.ServeHTTP(w, req)

	// Assert - should return 401 or 400 due to missing user_id
	assert.True(t, w.Code == http.StatusUnauthorized || w.Code == http.StatusBadRequest)
}

func TestAlertHandler_GetAlerts_WithAuth(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	mockRepo := &MockAlertRepository{}
	handler := handlers.NewAlertHandler(mockRepo, nil, nil)

	userID := uuid.New()

	// Create test alerts
	alerts := []entities.Alert{
		{
			ID:            uuid.New(),
			UserID:        userID,
			Symbol:        "BTCUSDT",
			AlertType:     "price",
			ConditionType: "above",
			TargetValue:   50000.0,
			Timeframe:     "1h",
			Enabled:       true,
		},
		{
			ID:            uuid.New(),
			UserID:        userID,
			Symbol:        "ETHUSDT",
			AlertType:     "price",
			ConditionType: "below",
			TargetValue:   3000.0,
			Timeframe:     "4h",
			Enabled:       true,
		},
	}

	// Setup mock expectations
	mockRepo.On("GetByUserID", mock.Anything, userID, 50, 0).Return(alerts, nil)

	// Create router and add middleware to set user_id
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	})
	router.GET("/alerts", handler.GetAlerts)

	// Create request
	req, _ := http.NewRequest("GET", "/alerts", nil)
	w := httptest.NewRecorder()

	// Execute
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Check response structure
	data, exists := response["data"]
	assert.True(t, exists)

	alertsData, ok := data.([]interface{})
	assert.True(t, ok)
	assert.Len(t, alertsData, 2)

	mockRepo.AssertExpectations(t)
}

func TestAlertHandler_CreateAlert_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := &MockAlertRepository{}
	handler := handlers.NewAlertHandler(mockRepo, nil, nil)

	router := gin.New()
	router.POST("/alerts", func(c *gin.Context) {
		c.Set("user_id", uuid.New())
		handler.CreateAlert(c)
	})

	alertData := map[string]interface{}{
		"symbol":         "BTCUSDT",
		"alert_type":     "price",
		"condition_type": "above",
		"target_value":   50000.0,
		"timeframe":      "1h",
		"enabled":        true,
	}

	jsonData, _ := json.Marshal(alertData)

	mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*entities.Alert")).Return(nil)

	req, _ := http.NewRequest("POST", "/alerts", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestAlertHandler_CreateAlert_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := &MockAlertRepository{}
	handler := handlers.NewAlertHandler(mockRepo, nil, nil)

	router := gin.New()
	router.POST("/alerts", func(c *gin.Context) {
		c.Set("user_id", uuid.New())
		handler.CreateAlert(c)
	})

	invalidJSON := `{"symbol": "BTCUSDT", "alert_type": "price"`

	req, _ := http.NewRequest("POST", "/alerts", bytes.NewBufferString(invalidJSON))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAlertHandler_CreateAlert_MissingFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := &MockAlertRepository{}
	handler := handlers.NewAlertHandler(mockRepo, nil, nil)

	router := gin.New()
	router.POST("/alerts", func(c *gin.Context) {
		c.Set("user_id", uuid.New())
		handler.CreateAlert(c)
	})

	alertData := map[string]interface{}{
		"symbol": "BTCUSDT",
		// Missing required fields
	}

	jsonData, _ := json.Marshal(alertData)

	req, _ := http.NewRequest("POST", "/alerts", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAlertHandler_CreateAlert_InvalidConditionType(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := &MockAlertRepository{}
	handler := handlers.NewAlertHandler(mockRepo, nil, nil)

	router := gin.New()
	router.POST("/alerts", func(c *gin.Context) {
		c.Set("user_id", uuid.New())
		handler.CreateAlert(c)
	})

	alertData := map[string]interface{}{
		"symbol":         "BTCUSDT",
		"alert_type":     "price",
		"condition_type": "invalid_condition",
		"target_value":   50000.0,
		"timeframe":      "1h",
	}

	jsonData, _ := json.Marshal(alertData)

	req, _ := http.NewRequest("POST", "/alerts", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAlertHandler_CreateAlert_RepositoryError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := &MockAlertRepository{}
	handler := handlers.NewAlertHandler(mockRepo, nil, nil)

	router := gin.New()
	router.POST("/alerts", func(c *gin.Context) {
		c.Set("user_id", uuid.New())
		handler.CreateAlert(c)
	})

	alertData := map[string]interface{}{
		"symbol":         "BTCUSDT",
		"alert_type":     "price",
		"condition_type": "above",
		"target_value":   50000.0,
		"timeframe":      "1h",
	}

	jsonData, _ := json.Marshal(alertData)

	mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*entities.Alert")).Return(assert.AnError)

	req, _ := http.NewRequest("POST", "/alerts", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestAlertHandler_UpdateAlert_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := &MockAlertRepository{}
	handler := handlers.NewAlertHandler(mockRepo, nil, nil)

	router := gin.New()
	router.PUT("/alerts/:id", func(c *gin.Context) {
		userID := uuid.New()
		c.Set("user_id", userID)
		handler.UpdateAlert(c)
	})

	alertID := uuid.New()
	userID := uuid.New()

	existingAlert := &entities.Alert{
		ID:      alertID,
		UserID:  userID,
		Symbol:  "BTCUSDT",
		Enabled: true,
	}

	updateData := map[string]interface{}{
		"enabled": false,
	}

	jsonData, _ := json.Marshal(updateData)

	mockRepo.On("GetByID", mock.Anything, alertID).Return(existingAlert, nil)
	mockRepo.On("Update", mock.Anything, mock.AnythingOfType("*entities.Alert")).Return(nil)

	req, _ := http.NewRequest("PUT", "/alerts/"+alertID.String(), bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Set the same userID that owns the alert
	router.Use(func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	})

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestAlertHandler_UpdateAlert_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := &MockAlertRepository{}
	handler := handlers.NewAlertHandler(mockRepo, nil, nil)

	router := gin.New()
	router.PUT("/alerts/:id", func(c *gin.Context) {
		c.Set("user_id", uuid.New())
		handler.UpdateAlert(c)
	})

	alertID := uuid.New()

	updateData := map[string]interface{}{
		"enabled": false,
	}

	jsonData, _ := json.Marshal(updateData)

	mockRepo.On("GetByID", mock.Anything, alertID).Return((*entities.Alert)(nil), assert.AnError)

	req, _ := http.NewRequest("PUT", "/alerts/"+alertID.String(), bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestAlertHandler_DeleteAlert_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := &MockAlertRepository{}
	handler := handlers.NewAlertHandler(mockRepo, nil, nil)

	router := gin.New()
	router.DELETE("/alerts/:id", func(c *gin.Context) {
		userID := uuid.New()
		c.Set("user_id", userID)
		handler.DeleteAlert(c)
	})

	alertID := uuid.New()
	userID := uuid.New()

	existingAlert := &entities.Alert{
		ID:     alertID,
		UserID: userID,
		Symbol: "BTCUSDT",
	}

	mockRepo.On("GetByID", mock.Anything, alertID).Return(existingAlert, nil)
	mockRepo.On("Delete", mock.Anything, alertID).Return(nil)

	req, _ := http.NewRequest("DELETE", "/alerts/"+alertID.String(), nil)
	w := httptest.NewRecorder()

	// Set the same userID that owns the alert
	router.Use(func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	})

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestAlertHandler_DeleteAlert_Forbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := &MockAlertRepository{}
	handler := handlers.NewAlertHandler(mockRepo, nil, nil)

	router := gin.New()
	router.DELETE("/alerts/:id", func(c *gin.Context) {
		c.Set("user_id", uuid.New()) // Different user
		handler.DeleteAlert(c)
	})

	alertID := uuid.New()
	userID := uuid.New()

	existingAlert := &entities.Alert{
		ID:     alertID,
		UserID: userID, // Owned by different user
		Symbol: "BTCUSDT",
	}

	mockRepo.On("GetByID", mock.Anything, alertID).Return(existingAlert, nil)

	req, _ := http.NewRequest("DELETE", "/alerts/"+alertID.String(), nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestAlertHandler_DeleteAlert_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := &MockAlertRepository{}
	handler := handlers.NewAlertHandler(mockRepo, nil, nil)

	router := gin.New()
	router.DELETE("/alerts/:id", func(c *gin.Context) {
		c.Set("user_id", uuid.New())
		handler.DeleteAlert(c)
	})

	req, _ := http.NewRequest("DELETE", "/alerts/invalid-uuid", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
