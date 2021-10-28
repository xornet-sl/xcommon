package xcommon

// TODO: test signal receiving and gracefull shutdown

import (
	"context"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGroupSignals(t *testing.T) {
	t.Run("BindUnbindsHandlers", SubtestBindUnbindsHandlers)
	t.Run("SignalWatcher", SubtestSignalWatcher)
}

func SubtestSignalWatcher(t *testing.T) {
	t.Parallel()
	ctx, cancel := GetGlobalContext()
	defer cancel()
	pid := os.Getpid()
	SetupSignalWatcher(ctx)
	watcherChan := make(chan string, 10)
	watcherTimeoutChan := make(chan string, 10)
	unwatchChan := make(chan string, 10)
	handlerResponseChan := make(chan string, 10)

	go sigWatcher(ctx, t, watcherChan, watcherTimeoutChan, unwatchChan)

	sideEffects := map[string]bool{}

	var simpleHandler1 OSSignalHandler = func(ctx, fastCtx context.Context, sig os.Signal) error {
		name := "simpleHandler1"
		sideEffects[name] = true
		time.Sleep(10 * time.Millisecond)
		handlerResponseChan <- name
		return nil
	}
	var simpleHandlerWithStop OSSignalHandler = func(ctx, fastCtx context.Context, sig os.Signal) error {
		name := "simpleHandlerWithStopPropagation"
		sideEffects[name] = true
		time.Sleep(20 * time.Millisecond)
		handlerResponseChan <- name
		return OSSignalStopPropagationError
	}
	var simpleHandler2 OSSignalHandler = func(ctx, fastCtx context.Context, sig os.Signal) error {
		name := "simpleHandler2"
		sideEffects[name] = true
		time.Sleep(15 * time.Millisecond)
		handlerResponseChan <- name
		return nil
	}
	var longHandler OSSignalHandler = func(ctx, fastCtx context.Context, sig os.Signal) error {
		name := "longHandler"
		sideEffects[name] = true
		select {
		case <-fastCtx.Done():
		}
		time.Sleep(15 * time.Millisecond)
		handlerResponseChan <- name
		return nil
	}

	var (
		Success = "Success"
		Timeout = "Timeout"
		Fail    = "Fail"
	)
	type TestInfo struct {
		Name           string
		fn             *OSSignalHandler
		ExpectedResult string
		sig            os.Signal
	}
	testplan := []TestInfo{
		{"simpleHandler1", &simpleHandler1, Fail, syscall.SIGUSR1}, // Because of stop propagation
		{"simpleHandlerWithStopPropagation", &simpleHandlerWithStop, Success, syscall.SIGUSR1},
		{"simpleHandler2", &simpleHandler2, Success, syscall.SIGUSR1},
		{"longHandler", &longHandler, Timeout, syscall.SIGUSR2},
	}

	for _, info := range testplan {
		BindHandlerToSignals(info.fn, info.sig)
	}
	for _, info := range testplan {
		watcherChan <- info.Name
	}
	syscall.Kill(pid, syscall.SIGUSR1)
	time.Sleep(2 * time.Millisecond)
	syscall.Kill(pid, syscall.SIGUSR2)

	tasks := len(testplan) - 1

	results := map[string]bool{}

	for {
		select {
		case <-time.After(15 * time.Second):
			require.Fail(t, "Too many time spent in awaiting signal handlers response")
		case task := <-watcherTimeoutChan:
			results[task] = false
		case task := <-handlerResponseChan:
			unwatchChan <- task
			tasks--
			if _, ok := results[task]; !ok {
				results[task] = true
			}
		}
		if tasks == 0 {
			break
		}
	}

	for _, info := range testplan {
		var result string
		if !sideEffects[info.Name] {
			result = Fail
		} else {
			res, ok := results[info.Name]
			if !ok {
				require.Fail(t, "Test malfunction", "Unreachable branch")
			}
			if res {
				result = Success
			} else {
				result = Timeout
			}
		}
		assert.Equalf(t, info.ExpectedResult, result, "Wrong signal handler '%s' status", info.Name)
	}
}

func sigWatcher(ctx context.Context, t *testing.T,
	watcherChan chan string,
	watcherTimeoutChan chan string,
	unwatchChan chan string) {

	ticker := time.NewTicker(10 * time.Millisecond)
	tasksInProgress := map[string]<-chan time.Time{}
	for {
		select {
		case <-ticker.C:
			for task, taskTimer := range tasksInProgress {
				select {
				case <-taskTimer:
					watcherTimeoutChan <- task
					delete(tasksInProgress, task)
				default:
				}
			}
		case task := <-unwatchChan:
			delete(tasksInProgress, task)
		case task := <-watcherChan:
			tasksInProgress[task] = time.After(100 * time.Millisecond)
		case <-ctx.Done():
			return
		}
	}
}

func SubtestBindUnbindsHandlers(t *testing.T) {
	expectedSignals := make(map[os.Signal][]*OSSignalHandler)

	remove := func(handlers []*OSSignalHandler, handler *OSSignalHandler) []*OSSignalHandler {
		if handlers == nil || len(handlers) == 0 {
			return []*OSSignalHandler{}
		}
		ret := make([]*OSSignalHandler, 0, len(handlers)-1)
		for _, h := range handlers {
			if h != handler {
				ret = append(ret, h)
			}
		}
		return ret
	}

	var sigHandler1 OSSignalHandler = func(ctx context.Context, fastCtx context.Context, sig os.Signal) error {
		return nil
	}
	var sigHandler2 OSSignalHandler = func(ctx context.Context, fastCtx context.Context, sig os.Signal) error {
		return nil
	}
	var sigHandler3 OSSignalHandler = func(ctx context.Context, fastCtx context.Context, sig os.Signal) error {
		return nil
	}
	var sigHandlerNonExisted OSSignalHandler = func(ctx context.Context, fastCtx context.Context, sig os.Signal) error {
		return nil
	}

	// Null addings
	BindHandlerToSignals(nil, nil)
	BindHandlerToSignals(nil)
	BindHandlerToSignals(&sigHandler1, nil)
	assert.Equal(t, expectedSignals, _signals, "Adding nil bindings")
	// one handler - one signal with double check
	BindHandlerToSignals(&sigHandler1, syscall.SIGABRT)
	expectedSignals[syscall.SIGABRT] = append(expectedSignals[syscall.SIGABRT], &sigHandler1)
	assert.Equal(t, expectedSignals, _signals, "Adding single handler to single siglal")
	BindHandlerToSignals(&sigHandler1, syscall.SIGABRT)
	assert.Equal(t, expectedSignals, _signals, "Adding duplicated handler to signal")
	// add other two signals to existing handler
	BindHandlerToSignals(&sigHandler1, os.Interrupt, syscall.SIGCHLD)
	expectedSignals[os.Interrupt] = append(expectedSignals[os.Interrupt], &sigHandler1)
	expectedSignals[syscall.SIGCHLD] = append(expectedSignals[syscall.SIGCHLD], &sigHandler1)
	assert.Equal(t, expectedSignals, _signals, "Adding another signal to existing handler")
	// add another handler to tree signal. one signal is already having a handler
	BindHandlerToSignals(&sigHandler2, syscall.SIGABRT, syscall.SIGALRM, syscall.SIGHUP)
	expectedSignals[syscall.SIGABRT] = append(expectedSignals[syscall.SIGABRT], &sigHandler2)
	expectedSignals[syscall.SIGALRM] = append(expectedSignals[syscall.SIGALRM], &sigHandler2)
	expectedSignals[syscall.SIGHUP] = append(expectedSignals[syscall.SIGHUP], &sigHandler2)
	assert.Equal(t, expectedSignals, _signals, "Adding a new handler to two signals where one signal is already having a handler")
	// add third handler for all three signals
	BindHandlerToSignals(&sigHandler3, syscall.SIGABRT, os.Interrupt, syscall.SIGALRM)
	expectedSignals[syscall.SIGABRT] = append(expectedSignals[syscall.SIGABRT], &sigHandler3)
	expectedSignals[os.Interrupt] = append(expectedSignals[os.Interrupt], &sigHandler3)
	expectedSignals[syscall.SIGALRM] = append(expectedSignals[syscall.SIGALRM], &sigHandler3)
	assert.Equal(t, expectedSignals, _signals, "Adding third handler for all three signals")

	// test wrong input
	UnbindSignals()
	UnbindHandlerFromSignals(nil)
	UnbindHandlerFromSignals(&sigHandler1, nil)
	UnbindHandlers()
	UnbindHandlers(nil)
	UnbindHandlerFromSignals(&sigHandler1, syscall.SIGSTOP)
	UnbindHandlers(&sigHandlerNonExisted)
	assert.Equal(t, expectedSignals, _signals, "Wrong input shouldn't affect anything")

	// delete h2 from all signals
	UnbindHandlers(&sigHandler2)
	expectedSignals[syscall.SIGABRT] = remove(expectedSignals[syscall.SIGABRT], &sigHandler2)
	expectedSignals[syscall.SIGALRM] = remove(expectedSignals[syscall.SIGALRM], &sigHandler2)
	delete(expectedSignals, syscall.SIGHUP)
	assert.Equal(t, expectedSignals, _signals, "Removing handler2 from all two signals")
	// delete h3 from ABRT and INT
	UnbindHandlerFromSignals(&sigHandler3, syscall.SIGABRT, os.Interrupt)
	expectedSignals[syscall.SIGABRT] = remove(expectedSignals[syscall.SIGABRT], &sigHandler3)
	expectedSignals[os.Interrupt] = remove(expectedSignals[os.Interrupt], &sigHandler3)
	assert.Equal(t, expectedSignals, _signals, "Removing handler3 from only two signals")
	// delete all ABRT handlers
	BindHandlerToSignals(&sigHandler3, syscall.SIGABRT) // add more to ABRT
	UnbindSignals(syscall.SIGABRT)
	delete(expectedSignals, syscall.SIGABRT)
	assert.Equal(t, expectedSignals, _signals, "Clear SIGABRT")
	// delete h3 from alarm
	UnbindHandlerFromSignals(&sigHandler3, syscall.SIGALRM)
	delete(expectedSignals, syscall.SIGALRM)
	assert.Equal(t, expectedSignals, _signals, "Removing last handler removes signal")
	// delete everything
	BindHandlerToSignals(&sigHandler1, syscall.SIGCHLD) // just for fun
	UnbindAllSignals()
	assert.Equal(t, map[os.Signal][]*OSSignalHandler{}, _signals, "Unbinding all signals and handlers")
}
