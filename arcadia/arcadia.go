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
	"popplio/arcadia/dclient"
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

// Start brings up the staff bot's Discord connection, the panel API and, on
// production only, the background tasks.
//
// The Discord connection comes first: the panel validates staff position role
// ids against the gateway cache, and the tasks read guild members, so both want
// a connected client. Neither hard-fails without one - an uncached guild reads
// as "not found" - so a Discord failure degrades rather than taking the panel
// down with it.
//
// Background tasks are gated to non-staging environments exactly as upstream
// gates them.
func Start(parent context.Context) *Arcadia {
	ctx, cancel := context.WithCancel(parent)

	a := &Arcadia{
		panel:  panel.New(),
		cancel: cancel,
	}

	if err := dclient.Setup(ctx, bot.Listener(ctx)); err != nil {
		state.Logger.Error("Staff bot failed to start; the panel will run without Discord", zap.Error(err))
	}

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

// Stop drains the panel server, stops the task tickers and closes the staff
// bot's gateway connection.
//
// IMPROVEMENT (§14c): the Rust version had no graceful shutdown at all.
func (a *Arcadia) Stop(timeout time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := a.panel.Shutdown(ctx); err != nil {
		state.Logger.Error("Panel server shutdown failed", zap.Error(err))
	}

	a.cancel()

	defer dclient.Close(ctx)

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
