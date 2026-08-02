// Package arcadia wires the ported staff panel API, staff Discord bot and
// background tasks into Popplio's startup.
//
// It owns three long-lived goroutines - the panel HTTP server, the task runners
// and (indirectly) the Discord event listeners - and shuts all of them down
// cleanly.
package arcadia

import (
	"context"
	"sync"
	"time"

	"popplio/arcadia/bot"
	"popplio/arcadia/panel"
	"popplio/arcadia/tasks"
	"popplio/config"
	"popplio/state"

	"go.uber.org/zap"
)

// Arcadia is the running staff subsystem.
type Arcadia struct {
	panel  *panel.Server
	cancel context.CancelFunc
	tasks  *sync.WaitGroup
}

// Start brings up the panel API, registers the Discord bot's listeners and, on
// production only, starts the background tasks.
//
// Background tasks are gated to non-staging environments exactly as upstream
// gates them.
func Start(parent context.Context) *Arcadia {
	ctx, cancel := context.WithCancel(parent)

	a := &Arcadia{
		panel:  panel.New(),
		cancel: cancel,
	}

	bot.Setup(ctx)

	go func() {
		if err := a.panel.Start(ctx); err != nil {
			state.Logger.Error("Panel server stopped with an error", zap.Error(err))
		}
	}()

	if config.CurrentEnv != config.CurrentEnvStaging {
		a.tasks = tasks.Start(ctx)
	} else {
		state.Logger.Info("Skipping arcadia background tasks on staging")
	}

	return a
}

// Stop drains the panel server and stops the task tickers.
//
// IMPROVEMENT (§14c): the Rust version had no graceful shutdown at all.
func (a *Arcadia) Stop(timeout time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := a.panel.Shutdown(ctx); err != nil {
		state.Logger.Error("Panel server shutdown failed", zap.Error(err))
	}

	a.cancel()

	if a.tasks != nil {
		done := make(chan struct{})

		go func() {
			a.tasks.Wait()
			close(done)
		}()

		select {
		case <-done:
		case <-ctx.Done():
			state.Logger.Warn("Timed out waiting for arcadia tasks to stop")
		}
	}
}
