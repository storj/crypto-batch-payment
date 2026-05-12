package main

import (
	"os"
	"path/filepath"
	"time"

	"github.com/zeebo/errs"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func openLog(dataDir string) (*zap.Logger, error) {
	// Ensure a logs directory exists
	logsDir := filepath.Join(dataDir, "logs")
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return nil, errs.Wrap(err)
	}

	// Name the log based on the current timestamp to millisecond precision
	logName := time.Now().UTC().Format("2006.01.02.15.04.05.000Z") + ".json"

	// Convert to an absolute path for the file URI passed to zap
	logsPath, err := filepath.Abs(filepath.Join(logsDir, logName))
	if err != nil {
		return nil, errs.Wrap(err)
	}

	stderrLog, err := openConsoleLog()
	if err != nil {
		return nil, err
	}

	// Send debug to file as JSON
	fileEncoder := zap.NewProductionEncoderConfig()
	fileEncoder.EncodeTime = zapcore.ISO8601TimeEncoder
	fileLog, err := (zap.Config{
		Level:         zap.NewAtomicLevelAt(zap.DebugLevel),
		Encoding:      "json",
		EncoderConfig: fileEncoder,
		OutputPaths:   []string{"file://" + logsPath},
	}).Build()
	if err != nil {
		return nil, errs.Wrap(err)
	}

	log := zap.New(zapcore.NewTee(stderrLog.Core(), fileLog.Core()))

	// Overwrite the latest symlink
	if err := os.Symlink(logName, filepath.Join(logsDir, ".latest")); err != nil {
		return nil, errs.Wrap(err)
	}
	if err := os.Rename(filepath.Join(logsDir, ".latest"), filepath.Join(logsDir, "latest")); err != nil {
		return nil, errs.Wrap(err)
	}

	return log, nil
}

// openConsoleLog creates a logger using info level + console.
func openConsoleLog() (*zap.Logger, error) {
	stderrEncoder := zap.NewDevelopmentEncoderConfig()
	stderrEncoder.EncodeLevel = zapcore.CapitalColorLevelEncoder
	stderrLog, err := (zap.Config{
		Level:         zap.NewAtomicLevelAt(zap.InfoLevel),
		Encoding:      "console",
		EncoderConfig: stderrEncoder,
		OutputPaths:   []string{"stderr"},
	}).Build()
	return stderrLog, errs.Wrap(err)
}
