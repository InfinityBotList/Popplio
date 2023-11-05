package assets

import (
	"bytes"
	"os"
	"popplio/state"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type mutLogger struct {
	sync.Mutex
	taskId string
	buf    *bytes.Buffer
}

func (m *mutLogger) pushBuf() {
	defer m.Unlock()
	m.Lock()

	// Get existing error
	existing, err := state.Redis.Get(state.Context, "task:status:"+m.taskId).Result()

	if err != nil {
		state.Logger.Error("Failed to get existing status", zap.Error(err), zap.String("task_id", m.taskId))
		existing = ""
	}

	// Append new status
	existing += "\n" + m.buf.String()

	if err := state.Redis.Set(state.Context, "task:status:"+m.taskId, existing, time.Hour*4).Err(); err != nil {
		state.Logger.Error("Failed to set status", zap.Error(err), zap.String("task_id", m.taskId))
	}

	// Reset buffer
	m.buf.Reset()
}

func (m *mutLogger) Write(p []byte) (n int, err error) {
	m.Lock()
	defer m.Unlock()
	return m.buf.Write(p)
}

func (m *mutLogger) Sync() error {
	m.pushBuf()
	return nil
}

func newTaskLogger(taskId string) (*zap.Logger, *mutLogger) {
	buf := bytes.NewBuffer([]byte{})

	ml := &mutLogger{
		taskId: taskId,
		buf:    buf,
	}

	logger := zap.New(zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.NewMultiWriteSyncer(
			ml,
			os.Stdout,
		),
		zapcore.DebugLevel,
	))
	return logger, ml
}
