package logger

import (
	"log"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var global *zap.Logger

// Init 初始化全局 zap 日志器，并桥接标准库 log。
func Init() error {
	cfg := zap.NewProductionConfig()
	cfg.Encoding = "json"
	cfg.OutputPaths = []string{"stdout"}
	cfg.ErrorOutputPaths = []string{"stderr"}
	cfg.EncoderConfig.TimeKey = "time"
	cfg.EncoderConfig.LevelKey = "level"
	cfg.EncoderConfig.MessageKey = "msg"
	cfg.EncoderConfig.CallerKey = "caller"
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	cfg.EncoderConfig.EncodeLevel = zapcore.LowercaseLevelEncoder
	cfg.EncoderConfig.EncodeDuration = zapcore.MillisDurationEncoder

	lg, err := cfg.Build(zap.AddCaller())
	if err != nil {
		return err
	}

	global = lg
	zap.ReplaceGlobals(lg)

	stdWriter := &zapStdWriter{logger: lg.Named("stdlog").WithOptions(zap.AddCallerSkip(1))}
	log.SetFlags(0)
	log.SetOutput(stdWriter)

	return nil
}

// L 返回全局日志器；若未初始化则返回可用的降级日志器。
func L() *zap.Logger {
	if global != nil {
		return global
	}
	return zap.NewNop()
}

// Sync 刷新日志缓冲。
func Sync() error {
	if global == nil {
		return nil
	}
	return global.Sync()
}

type zapStdWriter struct {
	logger *zap.Logger
}

func (w *zapStdWriter) Write(p []byte) (int, error) {
	msg := string(p)
	for len(msg) > 0 && (msg[len(msg)-1] == '\n' || msg[len(msg)-1] == '\r') {
		msg = msg[:len(msg)-1]
	}
	if msg != "" {
		w.logger.Info(msg)
	}
	return len(p), nil
}

func init() {
	if global == nil {
		global = zap.New(zapcore.NewCore(
			zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
			zapcore.AddSync(os.Stdout),
			zapcore.InfoLevel,
		))
	}
}
