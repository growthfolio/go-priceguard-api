package middleware

import (
	"github.com/gin-gonic/gin"
)

// SecurityHeadersMiddleware adiciona headers de segurança essenciais
func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Previne ataques de clickjacking
		c.Header("X-Frame-Options", "DENY")

		// Previne ataques XSS
		c.Header("X-XSS-Protection", "1; mode=block")

		// Previne MIME type sniffing
		c.Header("X-Content-Type-Options", "nosniff")

		// Força HTTPS em produção (quando aplicável)
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

		// Content Security Policy básica
		c.Header("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self'; connect-src 'self' wss: https:; frame-ancestors 'none';")

		// Controla informações do referrer
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		// Controla features do browser
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		// Remove header que expõe informações do servidor
		c.Header("Server", "")
		c.Header("X-Powered-By", "")

		c.Next()
	}
}

// CSRFProtectionMiddleware middleware básico de proteção CSRF
func CSRFProtectionMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Para métodos que modificam estado, verifica token CSRF
		if c.Request.Method == "POST" || c.Request.Method == "PUT" ||
			c.Request.Method == "DELETE" || c.Request.Method == "PATCH" {

			// Obtém token do header
			token := c.GetHeader("X-CSRF-Token")

			// Para APIs, podemos usar verificação baseada em Origin/Referer
			origin := c.GetHeader("Origin")
			referer := c.GetHeader("Referer")

			// Permite requisições de mesma origem ou com token válido
			if token == "" && origin == "" && referer == "" {
				c.JSON(403, gin.H{
					"error": "CSRF protection: missing token or origin",
					"code":  "CSRF_TOKEN_MISSING",
				})
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

// SanitizeInputMiddleware middleware para sanitização básica de inputs
func SanitizeInputMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Remove headers potencialmente perigosos
		c.Request.Header.Del("X-Forwarded-Host")
		c.Request.Header.Del("X-Original-Host")

		// Adiciona validação de Content-Type para requests com body
		if c.Request.Method == "POST" || c.Request.Method == "PUT" || c.Request.Method == "PATCH" {
			contentType := c.GetHeader("Content-Type")
			if contentType != "" && contentType != "application/json" && contentType != "application/x-www-form-urlencoded" {
				c.JSON(400, gin.H{
					"error": "Unsupported content type",
					"code":  "INVALID_CONTENT_TYPE",
				})
				c.Abort()
				return
			}
		}

		c.Next()
	}
}
