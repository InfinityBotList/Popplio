package notifications

import (
	"popplio/state"
	"reflect"
	"runtime"
	"time"

	"go.uber.org/zap"
)

var tasks = []func(){
	premiumCheck,
	vrCheck,
}

func taskMgr(f func()) {
	funcName := runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
	for {
		state.Logger.With(
			zap.String("task", funcName),
		).Info("Running task")
		f()
		time.Sleep(10 * time.Second)
	}
}

func startTaskMgr() {
	for _, task := range tasks {
		go taskMgr(task)
		time.Sleep(3 * time.Second)
	}
}
