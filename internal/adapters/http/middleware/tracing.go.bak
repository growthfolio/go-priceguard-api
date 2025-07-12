package middleware

import (
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// TracingMiddleware middleware para distributed tracing
func TracingMiddleware(serviceName string) gin.HandlerFunc {
	return otelgin.Middleware(serviceName)
}

// CustomTracingMiddleware middleware customizado para adicionar atributos específicos
func CustomTracingMiddleware(serviceName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Usa o middleware padrão do otelgin
		otelgin.Middleware(serviceName)(c)

		// Adiciona atributos customizados ao span
		span := trace.SpanFromContext(c.Request.Context())
		if span.IsRecording() {
			// Adiciona informações do usuário se disponível
			if userID, exists := c.Get("user_id"); exists {
				span.SetAttributes(attribute.String("user.id", userID.(string)))
			}

			// Adiciona request ID
			if requestID, exists := c.Get("request_id"); exists {
				span.SetAttributes(attribute.String("request.id", requestID.(string)))
			}

			// Adiciona informações da requisição
			span.SetAttributes(
				attribute.String("http.client_ip", c.ClientIP()),
				attribute.String("http.user_agent", c.Request.UserAgent()),
			)
		}

		c.Next()
	}
}
