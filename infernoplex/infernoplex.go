package infernoplex

import (
	"context"
	"sync"
	"time"

	"popplio/config"
	"popplio/infernoplex/bot"
	"popplio/infernoplex/dclient"
	"popplio/infernoplex/sorbet"
	"popplio/infernoplex/tasks"
	"popplio/state"

	"go.uber.org/zap"
)

type Infernoplex struct {
	sorbet *sorbet.Server
	cancel context.CancelFunc
	tasks  *sync.WaitGroup
}

func Start(parent context.Context) *Infernoplex {
	ctx, cancel := context.WithCancel(parent)

	i := &Infernoplex{
		sorbet: sorbet.New(),
		cancel: cancel,
	}

	if err := dclient.Setup(ctx, bot.Listener(ctx)); err != nil {
		state.Logger.Error("Infernoplex's Discord bot failed to start; Sorbet will run without it", zap.Error(err))
	}

	go func() {
		if err := i.sorbet.Start(); err != nil {
			state.Logger.Error("Sorbet server stopped with an error", zap.Error(err))
		}
	}()

	if config.CurrentEnv == config.CurrentEnvProd {
		i.tasks = tasks.Start(ctx)
	} else {
		state.Logger.Info("Skipping Infernoplex background tasks outside of production", zap.String("env", config.CurrentEnv))
	}

	return i
}

func (i *Infernoplex) Stop(timeout time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := i.sorbet.Shutdown(ctx); err != nil {
		state.Logger.Error("Sorbet server shutdown failed", zap.Error(err))
	}

	i.cancel()

	defer dclient.Close(ctx)

	if i.tasks != nil {
		done := make(chan struct{})

		go func() {
			i.tasks.Wait()
			close(done)
		}()

		select {
		case <-done:
		case <-ctx.Done():
			state.Logger.Warn("Timed out waiting for Infernoplex tasks to stop")
		}
	}
}
