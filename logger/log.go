package logger

import (
	"log"

	"go.uber.org/zap"
)

var logger *zap.SugaredLogger

func Init(debug bool) {
	developmentConfig := zap.NewDevelopmentConfig()
	if !debug {
		developmentConfig.Development = false
		developmentConfig.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	}
	l, err := developmentConfig.Build()
	if err != nil {
		log.Fatal(err)
	}
	zap.ReplaceGlobals(l)
	logger = zap.S()
}

// Debugw writes a debug message to the output.
func Debugw(msg string, keysAndValues ...interface{}) {
	logger.Debugw(msg, keysAndValues...)
}

// Debugf writes a debug message to the output.
func Debugf(template string, args ...interface{}) {
	logger.Debugf(template, args...)
}

// Infof writes a warn message to the output.
func Infof(template string, args ...interface{}) {
	logger.Infof(template, args...)
}
