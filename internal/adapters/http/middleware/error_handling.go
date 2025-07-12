package middleware

import (
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ErrorResponse estrutura padrão de resposta de erro
type ErrorResponse struct {
	Error     string      `json:"error"`
	Message   string      `json:"message"`
	Code      string      `json:"code,omitempty"`
	Details   interface{} `json:"details,omitempty"`
	RequestID string      `json:"request_id,omitempty"`
}

// ErrorHandlingMiddleware middleware para tratamento centralizado de erros
func ErrorHandlingMiddleware(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Obtém Request ID se disponível
				requestID, _ := c.Get("request_id")

				// Log do panic
				logger.Error("Panic recovered",
					zap.Any("error", err),
					zap.String("path", c.Request.URL.Path),
					zap.String("method", c.Request.Method),
					zap.Any("request_id", requestID),
					zap.String("stack", string(debug.Stack())),
				)

				// Resposta de erro 500
				response := ErrorResponse{
					Error:   "Internal Server Error",
					Message: "An unexpected error occurred",
					Code:    "INTERNAL_ERROR",
				}

				if requestID != nil {
					response.RequestID = requestID.(string)
				}

				c.JSON(http.StatusInternalServerError, response)
				c.Abort()
			}
		}()

		c.Next()

		// Verifica se houve erros durante o processamento
		if len(c.Errors) > 0 {
			err := c.Errors.Last()
			requestID, _ := c.Get("request_id")

			// Log do erro
			logger.Error("Request error",
				zap.Error(err.Err),
				zap.String("path", c.Request.URL.Path),
				zap.String("method", c.Request.Method),
				zap.Any("request_id", requestID),
			)

			// Se já foi enviada uma resposta, não faz nada
			if c.Writer.Written() {
				return
			}

			// Determina status code baseado no tipo de erro
			statusCode := http.StatusInternalServerError
			errorCode := "INTERNAL_ERROR"
			message := "An error occurred while processing your request"

			// Personaliza baseado no tipo de erro
			switch err.Type {
			case gin.ErrorTypeBind:
				statusCode = http.StatusBadRequest
				errorCode = "VALIDATION_ERROR"
				message = "Invalid request data"
			case gin.ErrorTypePublic:
				statusCode = http.StatusBadRequest
				errorCode = "BAD_REQUEST"
				message = err.Error()
			}

			response := ErrorResponse{
				Error:   http.StatusText(statusCode),
				Message: message,
				Code:    errorCode,
			}

			if requestID != nil {
				response.RequestID = requestID.(string)
			}

			c.JSON(statusCode, response)
		}
	}
}

// ValidationErrorMiddleware middleware específico para erros de validação
func ValidationErrorMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Verifica se existe erro de binding/validação
		if len(c.Errors) > 0 {
			for _, err := range c.Errors {
				if err.Type == gin.ErrorTypeBind {
					requestID, _ := c.Get("request_id")

					response := ErrorResponse{
						Error:   "Validation Error",
						Message: "Invalid request data",
						Code:    "VALIDATION_ERROR",
						Details: err.Error(),
					}

					if requestID != nil {
						response.RequestID = requestID.(string)
					}

					c.JSON(http.StatusBadRequest, response)
					c.Abort()
					return
				}
			}
		}
	}
}
