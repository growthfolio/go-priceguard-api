package infrastructure

import (
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// LoggerConfig configuração do logger
type LoggerConfig struct {
	Level       string   `json:"level"`
	Environment string   `json:"environment"`
	OutputPaths []string `json:"output_paths"`
}

// NewLogger cria uma nova instância do logger zap
func NewLogger(config LoggerConfig) (*zap.Logger, error) {
	// Configuração baseada no ambiente
	var zapConfig zap.Config

	if config.Environment == "production" {
		zapConfig = zap.NewProductionConfig()
	} else {
		zapConfig = zap.NewDevelopmentConfig()
		zapConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	// Define o nível de log
	level := parseLogLevel(config.Level)
	zapConfig.Level = zap.NewAtomicLevelAt(level)

	// Define outputs
	if len(config.OutputPaths) > 0 {
		zapConfig.OutputPaths = config.OutputPaths
	}

	// Configurações adicionais
	zapConfig.EncoderConfig.TimeKey = "timestamp"
	zapConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	zapConfig.EncoderConfig.CallerKey = "caller"
	zapConfig.EncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder

	// Adiciona campos padrão
	zapConfig.InitialFields = map[string]interface{}{
		"service": "priceguard-api",
	}

	logger, err := zapConfig.Build()
	if err != nil {
		return nil, err
	}

	// Adiciona caller info
	logger = logger.WithOptions(zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))

	return logger, nil
}

// parseLogLevel converte string para zapcore.Level
func parseLogLevel(level string) zapcore.Level {
	switch strings.ToLower(level) {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn", "warning":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	case "fatal":
		return zapcore.FatalLevel
	case "panic":
		return zapcore.PanicLevel
	default:
		return zapcore.InfoLevel
	}
}

// NewDefaultLogger cria um logger com configuração padrão
func NewDefaultLogger() (*zap.Logger, error) {
	env := os.Getenv("GIN_MODE")
	if env == "" {
		env = "development"
	}

	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		if env == "production" {
			logLevel = "info"
		} else {
			logLevel = "debug"
		}
	}

	config := LoggerConfig{
		Level:       logLevel,
		Environment: env,
		OutputPaths: []string{"stdout"},
	}

	return NewLogger(config)
}

// NewFileLogger cria um logger com rotação de arquivos
func NewFileLogger(filename string) (*zap.Logger, error) {
	env := os.Getenv("GIN_MODE")
	if env == "" {
		env = "development"
	}

	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}

	// Configuração do lumberjack para rotação de logs
	logRotator := &lumberjack.Logger{
		Filename:   filename,
		MaxSize:    100, // MB
		MaxBackups: 5,   // Número de backups
		MaxAge:     30,  // Dias
		Compress:   true,
	}

	// Configuração do core do zap
	var encoder zapcore.Encoder
	if env == "production" {
		encoder = zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
	} else {
		encoder = zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())
	}

	level := parseLogLevel(logLevel)
	core := zapcore.NewCore(
		encoder,
		zapcore.AddSync(logRotator),
		level,
	)

	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))

	// Adiciona campos padrão
	logger = logger.With(zap.String("service", "priceguard-api"))

	return logger, nil
}

// NewMultiOutputLogger cria um logger que escreve em múltiplos outputs
func NewMultiOutputLogger(outputs []string) (*zap.Logger, error) {
	env := os.Getenv("GIN_MODE")
	if env == "" {
		env = "development"
	}

	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}

	var cores []zapcore.Core
	level := parseLogLevel(logLevel)

	for _, output := range outputs {
		var writer zapcore.WriteSyncer
		var encoder zapcore.Encoder

		switch output {
		case "stdout":
			writer = zapcore.AddSync(os.Stdout)
			if env == "production" {
				encoder = zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
			} else {
				config := zap.NewDevelopmentEncoderConfig()
				config.EncodeLevel = zapcore.CapitalColorLevelEncoder
				encoder = zapcore.NewConsoleEncoder(config)
			}
		case "stderr":
			writer = zapcore.AddSync(os.Stderr)
			encoder = zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
		default:
			// Assume que é um arquivo
			logRotator := &lumberjack.Logger{
				Filename:   output,
				MaxSize:    100,
				MaxBackups: 5,
				MaxAge:     30,
				Compress:   true,
			}
			writer = zapcore.AddSync(logRotator)
			encoder = zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
		}

		core := zapcore.NewCore(encoder, writer, level)
		cores = append(cores, core)
	}

	// Combina todos os cores
	combinedCore := zapcore.NewTee(cores...)
	logger := zap.New(combinedCore, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))

	// Adiciona campos padrão
	logger = logger.With(zap.String("service", "priceguard-api"))

	return logger, nil
}
