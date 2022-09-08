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
	owner := in()
	logger.Debugw(msg, keysAndValues...)
	out(owner)
}

// Debugf writes a debug message to the output.
func Debugf(template string, args ...interface{}) {
	owner := in()
	logger.Debugf(template, args...)
	out(owner)
}

// Warnf writes a warn message to the output.
func Warnf(template string, args ...interface{}) {
	owner := in()
	logger.Warnf(template, args...)
	out(owner)
}

var inLogContext bool

func IsInLogContext() bool {
	return inLogContext
}

func in() bool {
	owner := false
	if !inLogContext {
		inLogContext = true
		owner = true
	}
	return owner
}

func out(owner bool) {
	if owner {
		inLogContext = false
	}
}
