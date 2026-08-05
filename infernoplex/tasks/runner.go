package tasks

import (
	"context"
	"runtime/debug"
	"sync"
	"time"

	"popplio/state"

	"go.uber.org/zap"
)

type Task struct {
	Name        string
	Description string
	Enabled     bool
	Interval    time.Duration
	Run         func(ctx context.Context) error
}

func All() []Task {
	return []Task{
		{
			Name:        "serversync",
			Description: "Syncs every listed server's icon, plus opted-in servers' emojis/stickers, into the database",
			Enabled:     true,
			Interval:    30 * time.Minute,
			Run:         ServerSync,
		},
		{
			Name:        "teamcleanup",
			Description: "Removes team members who've left their server or lost Administrator, via REST polling instead of the privileged Server Members intent",
			Enabled:     true,
			Interval:    15 * time.Minute,
			Run:         TeamCleanup,
		},
	}
}

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
			state.Logger.Info("Infernoplex task stopping", zap.String("task", task.Name))
			return
		case <-ticker.C:
			runOnce(ctx, task)
		}
	}
}

func runOnce(ctx context.Context, task Task) {
	defer func() {
		if rec := recover(); rec != nil {
			state.Logger.Error("Infernoplex task panicked",
				zap.String("task", task.Name),
				zap.Any("panic", rec),
				zap.ByteString("stack", debug.Stack()),
			)
		}
	}()

	state.Logger.Info("Running Infernoplex task", zap.String("task", task.Name), zap.String("description", task.Description))

	start := time.Now()

	if err := task.Run(ctx); err != nil {
		state.Logger.Error("Infernoplex task failed", zap.String("task", task.Name), zap.Error(err), zap.Duration("took", time.Since(start)))
		return
	}

	state.Logger.Info("Infernoplex task finished", zap.String("task", task.Name), zap.Duration("took", time.Since(start)))
}
