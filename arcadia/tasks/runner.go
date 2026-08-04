// Package tasks holds the staff bot's periodic background jobs.
//
// Each task runs on its own goroutine with its own ticker: the intervals differ
// by two orders of magnitude and a slow task must not delay the others. Every
// iteration has panic recovery, and an error is logged without stopping the
// loop, matching botox::taskman's semantics.
package tasks

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

// All returns every task, with the upstream intervals.
func All() []Task {
	return []Task{
		{
			Name:        "auto_unclaim",
			Description: "Checking for claimed bots greater than 1 hour claim interval",
			Enabled:     true,
			Interval:    60 * time.Second,
			Run:         AutoUnclaim,
		},
		{
			Name:        "bans_sync",
			Description: "Syncing bans",
			Enabled:     true,
			Interval:    300 * time.Second,
			Run:         BansSync,
		},
		{
			Name:        "deleted_bots",
			Description: "Cleaning up deleted bots",
			Enabled:     true,
			Interval:    500 * time.Second,
			Run:         DeletedBots,
		},
		{
			Name:        "generic_cleaner",
			Description: "Cleaning up orphaned generic entities",
			Enabled:     true,
			Interval:    400 * time.Second,
			Run:         GenericCleaner,
		},
		{
			Name:        "premium_remove",
			Description: "Removing expired subscriptions",
			Enabled:     true,
			Interval:    75 * time.Second,
			Run:         PremiumRemove,
		},
		{
			Name:        "spec_role_sync",
			Description: "Syncing special roles",
			Enabled:     true,
			Interval:    50 * time.Second,
			Run:         SpecRoleSync,
		},
		{
			Name:        "staff_resync",
			Description: "Resyncing staff permissions",
			Enabled:     true,
			Interval:    45 * time.Second,
			Run:         StaffResync,
		},
		{
			Name:        "team_cleaner",
			Description: "Fixing up empty/invalid teams",
			Enabled:     true,
			Interval:    300 * time.Second,
			Run:         TeamCleaner,
		},
		{
			Name:        "vote_resetter",
			Description: "Resetting votes",
			Enabled:     true,
			Interval:    600 * time.Second,
			Run:         VoteResetter,
		},
		{
			Name:        "topreviewer_sync",
			Description: "Sync top reviewers",
			Enabled:     true,
			Interval:    7 * 24 * time.Hour,
			Run:         TopReviewerSync,
		},
		{
			Name:        "japi_updater",
			Description: "JAPI Updater",
			Enabled:     true,
			Interval:    2 * time.Minute,
			Run:         JapiUpdater,
		},
	}
}

// Start launches every enabled task and returns a WaitGroup that completes once
// they have all stopped. Cancelling ctx stops them.
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
