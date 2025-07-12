package handlers

import (
	"net/http"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// HealthHandler handler para health checks e métricas
type HealthHandler struct {
	db  *gorm.DB
	rdb *redis.Client
}

// NewHealthHandler cria uma nova instância do HealthHandler
func NewHealthHandler(db *gorm.DB, rdb *redis.Client) *HealthHandler {
	return &HealthHandler{
		db:  db,
		rdb: rdb,
	}
}

// HealthCheck resposta do health check
type HealthCheck struct {
	Status    string            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
	Version   string            `json:"version"`
	Services  map[string]string `json:"services"`
	Uptime    string            `json:"uptime"`
}

// Metrics resposta de métricas
type Metrics struct {
	Timestamp time.Time `json:"timestamp"`
	System    struct {
		GoVersion    string `json:"go_version"`
		NumGoroutine int    `json:"num_goroutine"`
		NumCPU       int    `json:"num_cpu"`
		MemStats     struct {
			Alloc       uint64 `json:"alloc"`
			TotalAlloc  uint64 `json:"total_alloc"`
			Sys         uint64 `json:"sys"`
			NumGC       uint32 `json:"num_gc"`
			HeapObjects uint64 `json:"heap_objects"`
		} `json:"mem_stats"`
	} `json:"system"`
	Services struct {
		Database string `json:"database"`
		Redis    string `json:"redis"`
	} `json:"services"`
}

var startTime = time.Now()

// Health endpoint para health check
func (h *HealthHandler) Health(c *gin.Context) {
	services := make(map[string]string)

	// Verifica status do banco de dados
	if h.db != nil {
		sqlDB, err := h.db.DB()
		if err != nil {
			services["database"] = "error"
		} else {
			err = sqlDB.Ping()
			if err != nil {
				services["database"] = "down"
			} else {
				services["database"] = "up"
			}
		}
	} else {
		services["database"] = "not_configured"
	}

	// Verifica status do Redis
	if h.rdb != nil {
		_, err := h.rdb.Ping(c.Request.Context()).Result()
		if err != nil {
			services["redis"] = "down"
		} else {
			services["redis"] = "up"
		}
	} else {
		services["redis"] = "not_configured"
	}

	// Determina status geral
	status := "healthy"
	for _, serviceStatus := range services {
		if serviceStatus == "down" || serviceStatus == "error" {
			status = "unhealthy"
			break
		}
	}

	health := HealthCheck{
		Status:    status,
		Timestamp: time.Now(),
		Version:   "1.0.0", // Versão da aplicação
		Services:  services,
		Uptime:    time.Since(startTime).String(),
	}

	statusCode := http.StatusOK
	if status == "unhealthy" {
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, health)
}

// Metrics endpoint para métricas do sistema
func (h *HealthHandler) Metrics(c *gin.Context) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	metrics := Metrics{
		Timestamp: time.Now(),
	}

	// Métricas do sistema
	metrics.System.GoVersion = runtime.Version()
	metrics.System.NumGoroutine = runtime.NumGoroutine()
	metrics.System.NumCPU = runtime.NumCPU()
	metrics.System.MemStats.Alloc = m.Alloc
	metrics.System.MemStats.TotalAlloc = m.TotalAlloc
	metrics.System.MemStats.Sys = m.Sys
	metrics.System.MemStats.NumGC = m.NumGC
	metrics.System.MemStats.HeapObjects = m.HeapObjects

	// Status dos serviços
	if h.db != nil {
		sqlDB, err := h.db.DB()
		if err != nil {
			metrics.Services.Database = "error"
		} else {
			err = sqlDB.Ping()
			if err != nil {
				metrics.Services.Database = "down"
			} else {
				metrics.Services.Database = "up"
			}
		}
	}

	if h.rdb != nil {
		_, err := h.rdb.Ping(c.Request.Context()).Result()
		if err != nil {
			metrics.Services.Redis = "down"
		} else {
			metrics.Services.Redis = "up"
		}
	}

	c.JSON(http.StatusOK, metrics)
}

// Ready endpoint para readiness probe
func (h *HealthHandler) Ready(c *gin.Context) {
	// Verifica se todos os serviços críticos estão prontos
	ready := true
	services := make(map[string]bool)

	// Verifica banco de dados
	if h.db != nil {
		sqlDB, err := h.db.DB()
		if err != nil || sqlDB.Ping() != nil {
			ready = false
			services["database"] = false
		} else {
			services["database"] = true
		}
	}

	// Verifica Redis
	if h.rdb != nil {
		_, err := h.rdb.Ping(c.Request.Context()).Result()
		if err != nil {
			ready = false
			services["redis"] = false
		} else {
			services["redis"] = true
		}
	}

	response := gin.H{
		"ready":     ready,
		"timestamp": time.Now(),
		"services":  services,
	}

	if ready {
		c.JSON(http.StatusOK, response)
	} else {
		c.JSON(http.StatusServiceUnavailable, response)
	}
}

// Live endpoint para liveness probe
func (h *HealthHandler) Live(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"alive":     true,
		"timestamp": time.Now(),
		"uptime":    time.Since(startTime).String(),
	})
}
