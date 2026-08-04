// Package bgtasks holds Popplio's own periodic background jobs — distinct
// from arcadia/tasks, which are the staff bot's jobs and only run when
// Arcadia is configured. Jobs here run regardless, since they're core
// platform features rather than staff tooling.
package bgtasks

import (
	"context"
	"runtime/debug"
	"sync"
	"time"

	"popplio/state"

	"go.uber.org/zap"
)

// Task is one periodic job.
type Task struct {
	Name        string
	Description string
	Enabled     bool
	Interval    time.Duration
	Run         func(ctx context.Context) error
}

// All returns every task, with their intervals.
func All() []Task {
	return []Task{
		{
			Name:        "bot_uptime_check",
			Description: "Checking every listed bot's presence in the main server for uptime tracking",
			Enabled:     true,
			Interval:    5 * time.Minute,
			Run:         BotUptimeCheck,
		},
	}
}

// Start launches every enabled task and returns a WaitGroup that completes
// once they have all stopped. Cancelling ctx stops them.
func Start(ctx context.Context) *sync.WaitGroup {
	var wg sync.WaitGroup

	for _, task := range All() {
		if !task.Enabled {
			continue
		}

		wg.Add(1)

		go func(task Task) {
			defer wg.Done()
			runLoop(ctx, task)
		}(task)
	}

	return &wg
}

func runLoop(ctx context.Context, task Task) {
	ticker := time.NewTicker(task.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			state.Logger.Info("Task stopping", zap.String("task", task.Name))
			return
		case <-ticker.C:
			runOnce(ctx, task)
		}
	}
}

func runOnce(ctx context.Context, task Task) {
	defer func() {
		if rec := recover(); rec != nil {
			state.Logger.Error("Task panicked",
				zap.String("task", task.Name),
				zap.Any("panic", rec),
				zap.ByteString("stack", debug.Stack()),
			)
		}
	}()

	state.Logger.Info("Running task", zap.String("task", task.Name), zap.String("description", task.Description))

	start := time.Now()

	if err := task.Run(ctx); err != nil {
		state.Logger.Error("Task failed", zap.String("task", task.Name), zap.Error(err), zap.Duration("took", time.Since(start)))
		return
	}

	state.Logger.Info("Task finished", zap.String("task", task.Name), zap.Duration("took", time.Since(start)))
}
