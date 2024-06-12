package assets

import (
	"fmt"
	"os"
	"popplio/state"
	"sync"

	"github.com/infinitybotlist/eureka/jsonimpl"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type mutLogger struct {
	sync.Mutex
	taskId string
}

func (m *mutLogger) add(p []byte) error {
	state.Logger.Info("add called", zap.String("taskId", m.taskId))
	defer m.Unlock()
	m.Lock()

	var data map[string]any

	err := jsonimpl.Unmarshal(p, &data)

	if err != nil {
		return fmt.Errorf("failed to unmarshal json: %w", err)
	}

	// For us, this is just an array append of the json
	_, err = state.Pool.Exec(state.Context, "UPDATE tasks SET statuses = array_append(statuses, $1) WHERE task_id = $2", data, m.taskId)

	if err != nil {
		return fmt.Errorf("failed to update statuses: %w", err)
	}

	return nil
}

func (m *mutLogger) Write(p []byte) (n int, err error) {
	state.Logger.Info("Write called", zap.String("taskId", m.taskId))
	err = m.add(p)

	if err != nil {
		state.Logger.Error("[dwWriter] Failed to add to buffer", zap.Error(err), zap.String("taskId", m.taskId))
	}

	return len(p), err
}

func (m *mutLogger) Sync() error {
	return nil
}

func newTaskLogger(taskId string) (*zap.Logger, *mutLogger) {
	ml := &mutLogger{
		taskId: taskId,
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
