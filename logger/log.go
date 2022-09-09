package logger

import (
	"log"

	"go.uber.org/zap"
)

var logger *zap.SugaredLogger

func Init(debug bool) {
	var developmentConfig zap.Config

	if debug {
		developmentConfig = zap.NewDevelopmentConfig()
	} else {
		developmentConfig = zap.NewProductionConfig()
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

// Warnf writes a warn message to the output.
func Warnf(template string, args ...interface{}) {
	logger.Warnf(template, args...)
}
