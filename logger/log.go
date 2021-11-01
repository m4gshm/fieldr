package logger

import (
	"flag"
	"log"
	"os"
	"strings"

	"go.uber.org/zap"
)

var logger *zap.SugaredLogger

func init() {
	debugFlag := flag.Bool("debug", false, "enable debug logging")
	var developmentConfig zap.Config
	debug := *debugFlag

	if !debug {
		envDebug := os.Getenv("FIELDR_DEBUG")
		if len(envDebug) > 0 && !(strings.ToLower(envDebug) == "disable" || strings.ToLower(envDebug) == "false") {
			debug = true
		}
	}

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

func Debugw(msg string, keysAndValues ...interface{}) {
	logger.Debugw(msg, keysAndValues...)
}

func Debugf(template string, args ...interface{}) {
	logger.Debugf(template, args...)
}
