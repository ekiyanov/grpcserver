package grpcserver

import (
	"os"

	"github.com/ekiyanov/logger"
	"go.uber.org/zap"
)

func osGetenv(envname, def string) string {

	var v = os.Getenv(envname)
	if v == "" {
		SLogger().Debugw("Unable to find envvar", "var", envname, "default", def)
		return def
	}

	return v
}
func Logger() *zap.Logger {
	return logger.Logger()
}

func SLogger() *zap.SugaredLogger {
	return logger.SLogger()
}
