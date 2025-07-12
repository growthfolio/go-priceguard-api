package middleware

import (
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
)

// CompressionMiddleware middleware de compressão de respostas
func CompressionMiddleware() gin.HandlerFunc {
	return gzip.Gzip(gzip.DefaultCompression, gzip.WithExcludedExtensions([]string{
		".png", ".gif", ".jpeg", ".jpg", // Imagens já comprimidas
		".woff", ".woff2", ".ttf", // Fontes já comprimidas
		".mp4", ".avi", ".mov", // Vídeos já comprimidos
		".zip", ".tar", ".gz", // Arquivos já comprimidos
	}))
}
