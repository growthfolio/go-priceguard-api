package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// LoggingMiddleware middleware de logging estruturado
func LoggingMiddleware(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Gera um Request ID único
		requestID := uuid.New().String()
		c.Header("X-Request-ID", requestID)
		c.Set("request_id", requestID)

		// Captura informações da requisição
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery
		method := c.Request.Method
		userAgent := c.Request.UserAgent()
		clientIP := c.ClientIP()

		// Skip logging para health checks e metrics
		if path == "/health" || path == "/metrics" {
			c.Next()
			return
		}

		// Captura o body da requisição para logs (apenas para métodos POST/PUT/PATCH)
		var requestBody []byte
		if method == "POST" || method == "PUT" || method == "PATCH" {
			if c.Request.Body != nil {
				requestBody, _ = io.ReadAll(c.Request.Body)
				c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
			}
		}

		// Continua processamento
		c.Next()

		// Calcula tempo de resposta
		latency := time.Since(start)
		statusCode := c.Writer.Status()

		// Obtém informações do usuário se autenticado
		var userID interface{}
		if uid, exists := c.Get("user_id"); exists {
			userID = uid
		}

		// Campos base do log
		logFields := []zap.Field{
			zap.String("request_id", requestID),
			zap.String("method", method),
			zap.String("path", path),
			zap.String("query", query),
			zap.Int("status_code", statusCode),
			zap.Duration("latency", latency),
			zap.String("client_ip", clientIP),
			zap.String("user_agent", userAgent),
		}

		// Adiciona user_id se disponível
		if userID != nil {
			logFields = append(logFields, zap.Any("user_id", userID))
		}

		// Adiciona body da requisição se não for muito grande e não contiver dados sensíveis
		if len(requestBody) > 0 && len(requestBody) < 1024 && !containsSensitiveData(string(requestBody)) {
			logFields = append(logFields, zap.String("request_body", string(requestBody)))
		}

		// Log com nível baseado no status code
		message := method + " " + path

		switch {
		case statusCode >= 500:
			logger.Error(message, logFields...)
		case statusCode >= 400:
			logger.Warn(message, logFields...)
		case statusCode >= 300:
			logger.Info(message, logFields...)
		default:
			logger.Info(message, logFields...)
		}
	}
}

// RequestIDMiddleware adiciona Request ID se não existir
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Verifica se já tem Request ID no header
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}

		c.Header("X-Request-ID", requestID)
		c.Set("request_id", requestID)
		c.Next()
	}
}

// containsSensitiveData verifica se o body contém dados sensíveis
func containsSensitiveData(body string) bool {
	sensitiveFields := []string{
		"password",
		"token",
		"secret",
		"key",
		"authorization",
		"credit_card",
		"ssn",
	}

	// Parse JSON para verificar campos sensíveis
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(body), &data); err == nil {
		for key := range data {
			for _, sensitive := range sensitiveFields {
				if key == sensitive {
					return true
				}
			}
		}
	}

	return false
}
