package logger_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zaffka/jigsaw/pkg/logger"
	"go.uber.org/zap"
)

func TestNew(t *testing.T) {
	t.Run("dev env with debug", func(t *testing.T) {
		buf := &bytes.Buffer{}
		log, err := logger.New(buf, logger.Opts{
			Service:    "test",
			Level:      "debug",
			UseJSONFmt: false,
		})
		require.NoError(t, err)
		require.NotNil(t, log)

		log.Debug("test", zap.String("str", "xxx"))

		res := buf.String()
		require.Contains(t, res, "\x1b[35mDEBUG\x1b[0m")
	})

	t.Run("prod env", func(t *testing.T) {
		buf := &bytes.Buffer{}
		log, err := logger.New(buf, logger.Opts{
			Service:    "test",
			Level:      "info",
			UseJSONFmt: true,
		})
		require.NoError(t, err)
		require.NotNil(t, log)

		log.Info("test", zap.String("str", "xxx"))

		res := buf.String()
		require.Contains(t, res, `{"level":"info","ts"`)
		require.NotContains(t, res, "INFO")
	})
}
