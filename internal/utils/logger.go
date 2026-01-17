package utils

import (
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Logger wraps logrus.Logger with additional functionality
type Logger struct {
	*logrus.Logger
}

// NewLogger creates a new logger instance
func NewLogger() *Logger {
	logger := logrus.New()
	
	// Set formatter
	logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339Nano,
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime:  "timestamp",
			logrus.FieldKeyLevel: "level",
			logrus.FieldKeyMsg:   "message",
		},
	})
	
	// Set log level from environment
	level := os.Getenv("LSCC_LOG_LEVEL")
	switch level {
	case "debug":
		logger.SetLevel(logrus.DebugLevel)
	case "info":
		logger.SetLevel(logrus.InfoLevel)
	case "warn":
		logger.SetLevel(logrus.WarnLevel)
	case "error":
		logger.SetLevel(logrus.ErrorLevel)
	default:
		logger.SetLevel(logrus.InfoLevel)
	}
	
	// Set output
	output := os.Getenv("LSCC_LOG_OUTPUT")
	if output == "file" {
		logDir := os.Getenv("LSCC_LOG_DIR")
		if logDir == "" {
			logDir = "./logs"
		}
		
		// Create log directory if it doesn't exist
		if err := os.MkdirAll(logDir, 0755); err != nil {
			logger.Warnf("Failed to create log directory: %v", err)
		} else {
			// Use lumberjack for log rotation
			logFile := &lumberjack.Logger{
				Filename:   filepath.Join(logDir, "lscc.log"),
				MaxSize:    100, // MB
				MaxBackups: 3,
				MaxAge:     28, // days
				Compress:   true,
			}
			
			// Write to both file and stdout
			multiWriter := io.MultiWriter(os.Stdout, logFile)
			logger.SetOutput(multiWriter)
		}
	} else {
		logger.SetOutput(os.Stdout)
	}
	
	return &Logger{Logger: logger}
}

// LogBlockchain logs blockchain-specific information
func (l *Logger) LogBlockchain(action string, fields logrus.Fields) {
	if fields == nil {
		fields = logrus.Fields{}
	}
	fields["component"] = "blockchain"
	fields["action"] = action
	fields["timestamp"] = time.Now().UTC()
	
	l.WithFields(fields).Info("Blockchain operation")
}

// LogConsensus logs consensus-specific information
func (l *Logger) LogConsensus(algorithm string, action string, fields logrus.Fields) {
	if fields == nil {
		fields = logrus.Fields{}
	}
	fields["component"] = "consensus"
	fields["algorithm"] = algorithm
	fields["action"] = action
	fields["timestamp"] = time.Now().UTC()
	
	l.WithFields(fields).Info("Consensus operation")
}

// LogSharding logs sharding-specific information
func (l *Logger) LogSharding(shardID int, action string, fields logrus.Fields) {
	if fields == nil {
		fields = logrus.Fields{}
	}
	fields["component"] = "sharding"
	fields["shard_id"] = shardID
	fields["action"] = action
	fields["timestamp"] = time.Now().UTC()
	
	l.WithFields(fields).Info("Sharding operation")
}

// LogNetwork logs network-specific information
func (l *Logger) LogNetwork(action string, fields logrus.Fields) {
	if fields == nil {
		fields = logrus.Fields{}
	}
	fields["component"] = "network"
	fields["action"] = action
	fields["timestamp"] = time.Now().UTC()
	
	l.WithFields(fields).Info("Network operation")
}

// LogPerformance logs performance metrics
func (l *Logger) LogPerformance(metric string, value interface{}, fields logrus.Fields) {
	if fields == nil {
		fields = logrus.Fields{}
	}
	fields["component"] = "performance"
	fields["metric"] = metric
	fields["value"] = value
	fields["timestamp"] = time.Now().UTC()
	
	l.WithFields(fields).Info("Performance metric")
}

// LogError logs error with context
func (l *Logger) LogError(component string, action string, err error, fields logrus.Fields) {
	if fields == nil {
		fields = logrus.Fields{}
	}
	fields["component"] = component
	fields["action"] = action
	fields["error"] = err.Error()
	fields["timestamp"] = time.Now().UTC()
	
	l.WithFields(fields).Error("Operation failed")
}

// LogDebug logs debug information with structured data
func (l *Logger) LogDebug(component string, message string, fields logrus.Fields) {
	if fields == nil {
		fields = logrus.Fields{}
	}
	fields["component"] = component
	fields["timestamp"] = time.Now().UTC()
	
	l.WithFields(fields).Debug(message)
}

// LogTransaction logs transaction-related information
func (l *Logger) LogTransaction(txID string, action string, fields logrus.Fields) {
	if fields == nil {
		fields = logrus.Fields{}
	}
	fields["component"] = "transaction"
	fields["tx_id"] = txID
	fields["action"] = action
	fields["timestamp"] = time.Now().UTC()
	
	l.WithFields(fields).Info("Transaction operation")
}

// LogValidation logs validation-related information
func (l *Logger) LogValidation(validator string, action string, success bool, fields logrus.Fields) {
	if fields == nil {
		fields = logrus.Fields{}
	}
	fields["component"] = "validation"
	fields["validator"] = validator
	fields["action"] = action
	fields["success"] = success
	fields["timestamp"] = time.Now().UTC()
	
	if success {
		l.WithFields(fields).Info("Validation successful")
	} else {
		l.WithFields(fields).Warn("Validation failed")
	}
}

// LogCrossShard logs cross-shard communication
func (l *Logger) LogCrossShard(fromShard, toShard int, messageType string, fields logrus.Fields) {
	if fields == nil {
		fields = logrus.Fields{}
	}
	fields["component"] = "cross_shard"
	fields["from_shard"] = fromShard
	fields["to_shard"] = toShard
	fields["message_type"] = messageType
	fields["timestamp"] = time.Now().UTC()
	
	l.WithFields(fields).Info("Cross-shard communication")
}

// GetContextLogger returns a logger with predefined context
func (l *Logger) GetContextLogger(component string, extraFields logrus.Fields) *logrus.Entry {
	fields := logrus.Fields{
		"component": component,
		"timestamp": time.Now().UTC(),
	}
	
	// Merge extra fields
	for k, v := range extraFields {
		fields[k] = v
	}
	
	return l.WithFields(fields)
}
