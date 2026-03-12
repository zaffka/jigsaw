package logger

import (
	"fmt"
	"io"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Opts struct {
	Service    string
	Version    string
	Level      string
	UseJSONFmt bool
}

// New creates a new logger instance.
// At DEV environment logger prints colored capitalized log level.
// At PROD environment logger prints logs in a JSON format.
func New(writer io.Writer, opts Opts) (*zap.Logger, error) {
	lvl, err := zapcore.ParseLevel(opts.Level)
	if err != nil {
		return nil, fmt.Errorf("failed to parse log level: %w", err)
	}

	conf := zap.NewProductionEncoderConfig()
	conf.EncodeTime = zapcore.RFC3339TimeEncoder

	var encoder zapcore.Encoder

	if opts.UseJSONFmt {
		encoder = zapcore.NewJSONEncoder(conf)
	} else {
		conf.EncodeLevel = zapcore.CapitalColorLevelEncoder
		encoder = zapcore.NewConsoleEncoder(conf)
	}
	core := zapcore.NewCore(
		encoder,
		zapcore.AddSync(writer),
		lvl,
	)

	logger := zap.New(core, zap.AddCaller()).With(
		zap.String("service", opts.Service),
		zap.String("version", opts.Version),
	)
	zap.RedirectStdLog(logger)

	return logger, nil
}
