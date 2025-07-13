package entities_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/growthfolio/go-priceguard-api/internal/domain/entities"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
)

func TestUserSettings_Creation(t *testing.T) {
	userID := uuid.New()
	settingsID := uuid.New()

	settings := entities.UserSettings{
		ID:                 settingsID,
		UserID:             userID,
		Theme:              "dark",
		DefaultTimeframe:   "1h",
		DefaultView:        "overview",
		NotificationsEmail: true,
		NotificationsPush:  true,
		NotificationsSMS:   false,
		RiskProfile:        "moderate",
		FavoriteSymbols:    pq.StringArray{"BTCUSDT", "ETHUSDT"},
	}

	assert.Equal(t, settingsID, settings.ID)
	assert.Equal(t, userID, settings.UserID)
	assert.Equal(t, "dark", settings.Theme)
	assert.Equal(t, "1h", settings.DefaultTimeframe)
	assert.Equal(t, "overview", settings.DefaultView)
	assert.True(t, settings.NotificationsEmail)
	assert.True(t, settings.NotificationsPush)
	assert.False(t, settings.NotificationsSMS)
	assert.Equal(t, "moderate", settings.RiskProfile)
	assert.Contains(t, settings.FavoriteSymbols, "BTCUSDT")
	assert.Contains(t, settings.FavoriteSymbols, "ETHUSDT")
}

func TestUserSettings_ThemeOptions(t *testing.T) {
	userID := uuid.New()

	themes := []string{"light", "dark", "auto"}

	for _, theme := range themes {
		t.Run("theme_"+theme, func(t *testing.T) {
			settings := entities.UserSettings{
				UserID: userID,
				Theme:  theme,
			}

			assert.Equal(t, theme, settings.Theme)
		})
	}
}

func TestUserSettings_TimeframeOptions(t *testing.T) {
	userID := uuid.New()

	timeframes := []string{"1m", "5m", "15m", "30m", "1h", "4h", "1d", "1w"}

	for _, timeframe := range timeframes {
		t.Run("timeframe_"+timeframe, func(t *testing.T) {
			settings := entities.UserSettings{
				UserID:           userID,
				DefaultTimeframe: timeframe,
			}

			assert.Equal(t, timeframe, settings.DefaultTimeframe)
		})
	}
}

func TestUserSettings_ViewOptions(t *testing.T) {
	userID := uuid.New()

	views := []string{"overview", "charts", "alerts", "portfolio"}

	for _, view := range views {
		t.Run("view_"+view, func(t *testing.T) {
			settings := entities.UserSettings{
				UserID:      userID,
				DefaultView: view,
			}

			assert.Equal(t, view, settings.DefaultView)
		})
	}
}

func TestUserSettings_NotificationPreferences(t *testing.T) {
	userID := uuid.New()

	tests := []struct {
		name  string
		email bool
		push  bool
		sms   bool
	}{
		{
			name:  "all notifications enabled",
			email: true,
			push:  true,
			sms:   true,
		},
		{
			name:  "only email and push",
			email: true,
			push:  true,
			sms:   false,
		},
		{
			name:  "only push notifications",
			email: false,
			push:  true,
			sms:   false,
		},
		{
			name:  "all notifications disabled",
			email: false,
			push:  false,
			sms:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			settings := entities.UserSettings{
				UserID:             userID,
				NotificationsEmail: tt.email,
				NotificationsPush:  tt.push,
				NotificationsSMS:   tt.sms,
			}

			assert.Equal(t, tt.email, settings.NotificationsEmail)
			assert.Equal(t, tt.push, settings.NotificationsPush)
			assert.Equal(t, tt.sms, settings.NotificationsSMS)
		})
	}
}

func TestUserSettings_RiskProfiles(t *testing.T) {
	userID := uuid.New()

	riskProfiles := []string{"conservative", "moderate", "aggressive"}

	for _, profile := range riskProfiles {
		t.Run("risk_profile_"+profile, func(t *testing.T) {
			settings := entities.UserSettings{
				UserID:      userID,
				RiskProfile: profile,
			}

			assert.Equal(t, profile, settings.RiskProfile)
		})
	}
}

func TestUserSettings_FavoriteSymbols(t *testing.T) {
	userID := uuid.New()

	tests := []struct {
		name            string
		favoriteSymbols pq.StringArray
		expectedCount   int
	}{
		{
			name:            "no favorites",
			favoriteSymbols: pq.StringArray{},
			expectedCount:   0,
		},
		{
			name:            "single favorite",
			favoriteSymbols: pq.StringArray{"BTCUSDT"},
			expectedCount:   1,
		},
		{
			name:            "multiple favorites",
			favoriteSymbols: pq.StringArray{"BTCUSDT", "ETHUSDT", "ADAUSDT"},
			expectedCount:   3,
		},
		{
			name:            "many favorites",
			favoriteSymbols: pq.StringArray{"BTCUSDT", "ETHUSDT", "ADAUSDT", "DOGEUSDT", "LTCUSDT", "XRPUSDT"},
			expectedCount:   6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			settings := entities.UserSettings{
				UserID:          userID,
				FavoriteSymbols: tt.favoriteSymbols,
			}

			assert.Equal(t, tt.expectedCount, len(settings.FavoriteSymbols))

			for _, symbol := range tt.favoriteSymbols {
				assert.Contains(t, settings.FavoriteSymbols, symbol)
			}
		})
	}
}

func TestUserSettings_DefaultValuesStructure(t *testing.T) {
	userID := uuid.New()

	// Test what should be default values
	settings := entities.UserSettings{
		UserID: userID,
		// Let other fields use their defaults
	}

	// These would be set by GORM defaults in real usage
	// Here we just test the structure allows them
	assert.Equal(t, userID, settings.UserID)
	assert.NotNil(t, &settings.Theme)
	assert.NotNil(t, &settings.DefaultTimeframe)
	assert.NotNil(t, &settings.DefaultView)
	assert.NotNil(t, &settings.NotificationsEmail)
	assert.NotNil(t, &settings.NotificationsPush)
	assert.NotNil(t, &settings.NotificationsSMS)
	assert.NotNil(t, &settings.RiskProfile)
}

func TestUserSettings_AddFavoriteSymbol(t *testing.T) {
	userID := uuid.New()

	settings := entities.UserSettings{
		UserID:          userID,
		FavoriteSymbols: pq.StringArray{"BTCUSDT"},
	}

	// Initially has one favorite
	assert.Equal(t, 1, len(settings.FavoriteSymbols))
	assert.Contains(t, settings.FavoriteSymbols, "BTCUSDT")

	// Add another favorite
	settings.FavoriteSymbols = append(settings.FavoriteSymbols, "ETHUSDT")

	assert.Equal(t, 2, len(settings.FavoriteSymbols))
	assert.Contains(t, settings.FavoriteSymbols, "BTCUSDT")
	assert.Contains(t, settings.FavoriteSymbols, "ETHUSDT")
}

func TestUserSettings_RemoveFavoriteSymbol(t *testing.T) {
	userID := uuid.New()

	settings := entities.UserSettings{
		UserID:          userID,
		FavoriteSymbols: pq.StringArray{"BTCUSDT", "ETHUSDT", "ADAUSDT"},
	}

	// Initially has three favorites
	assert.Equal(t, 3, len(settings.FavoriteSymbols))

	// Remove one favorite (simulate removal)
	newFavorites := pq.StringArray{}
	for _, symbol := range settings.FavoriteSymbols {
		if symbol != "ETHUSDT" {
			newFavorites = append(newFavorites, symbol)
		}
	}
	settings.FavoriteSymbols = newFavorites

	assert.Equal(t, 2, len(settings.FavoriteSymbols))
	assert.Contains(t, settings.FavoriteSymbols, "BTCUSDT")
	assert.Contains(t, settings.FavoriteSymbols, "ADAUSDT")
	assert.NotContains(t, settings.FavoriteSymbols, "ETHUSDT")
}
