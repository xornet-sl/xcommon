package xcommon

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	gracefullShutdownTimeout = 1500 * time.Millisecond
	prolongedShutdownTimeout = 10 * time.Second
)

type OSSignalHandler func(ctx context.Context, fastCtx context.Context, sig os.Signal) error
type OSSigTermHandler func(ctx context.Context, sig os.Signal)

var OSSignalStopPropagationError = errors.New("Do not propagate signal handlers")
var SlowShutdownWithoutTermination = errors.New("You can slow shutdown process only during termination")
var SlowShutdownNeedAReason = errors.New("You MUST specify a reason for slow shutdown")
var SlowShutdownRequestedAlready = errors.New("Slow shutdown has been requested already. Sorry")

var _signalChan chan os.Signal
var _signals = map[os.Signal][]*OSSignalHandler{}
var _signalsMutex sync.RWMutex
var _defaultSigTermHandler = defaultSigTermHandler
var _defaultTermSignals = []os.Signal{
	os.Interrupt,
	syscall.SIGTERM,
}
var _termination = struct {
	mu        sync.RWMutex
	Started   bool
	StartedAt time.Time
	Prolonged bool
	Deadline  time.Time
}{}

func init() {
	_signalChan = make(chan os.Signal, 5)
}

func dispatchSignals(ctx context.Context, signal os.Signal) {
	// Should be kept under 100ms
	fastCtx, fastCancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer fastCancel()
	_signalsMutex.RLock()
	defer _signalsMutex.RUnlock()

	if _, ok := _signals[signal]; !ok {
		return
	}

	sigHandlers := _signals[signal]
	for idx := len(sigHandlers) - 1; idx >= 0; idx-- {
		sigHandler := *sigHandlers[idx]
		err := sigHandler(ctx, fastCtx, signal)

		if errors.Is(err, OSSignalStopPropagationError) {
			break
		}

		timeIsUp := false
		select {
		case <-fastCtx.Done():
			timeIsUp = true
		default:
		}
		if timeIsUp {
			break
		}
	}
}

func checkTermSignals(ctx context.Context, sig os.Signal) bool {
	for _, termSign := range _defaultTermSignals {
		if termSign == sig {
			log.Info("Shutdown requested by receiving signal ", sig.String())
			_terminate(ctx, sig)
			return true
		}
	}
	return false
}

func signalWatcher(ctx context.Context, signalChannel chan os.Signal) {
	defer func() {
		signal.Stop(signalChannel)
		close(signalChannel)
	}()
	ignoreSignals := false
	for {
		select {
		case sig := <-signalChannel:
			dispatchSignals(ctx, sig)
			ignoreSignals = checkTermSignals(ctx, sig)
			if ignoreSignals {
				signal.Ignore()
			}
		case <-ctx.Done():
			return
		}
	}
}

// SetupSignalWatcher enables signal watching for application
// It also dealing with graceful shutdowns
func SetupSignalWatcher(ctx context.Context) {
	signal.Notify(_signalChan, os.Interrupt, syscall.SIGTERM)
	go signalWatcher(ctx, _signalChan)
}

// Terminate launches this application termination process gracefully
// It simply executes current termination handler if any (if no handler
// then oops, go will just os.Exit() us)
func Terminate(ctx context.Context) {
	log.Info("Internal shutdown requested")
	_terminate(ctx, nil)
}

func _terminate(ctx context.Context, sig os.Signal) {
	if _defaultSigTermHandler != nil {
		_defaultSigTermHandler(ctx, sig)
	}
}

// defaultSigTermHandler is the default termination handler that handles
// by default two signals: os.Interrupt and syscall.SIGTERM
func defaultSigTermHandler(ctx context.Context, sig os.Signal) {
	waitFn := func(waitCtx context.Context) {
		select {
		case <-waitCtx.Done():
		}
	}

	waitCtx := context.Background()
	var cancel context.CancelFunc
	func() {
		_termination.mu.Lock()
		defer _termination.mu.Unlock()
		_termination.Started = true
		_termination.Prolonged = false
		_termination.StartedAt = time.Now()
		_termination.Deadline = _termination.StartedAt.Add(gracefullShutdownTimeout)
		waitCtx, cancel = context.WithDeadline(waitCtx, _termination.Deadline)
	}()
	_ctxCancel() // This is the signal to the whole application to start finalize everything

	waitFn(waitCtx)
	cancel()

	func() {
		_termination.mu.RLock()
		defer _termination.mu.RUnlock()
		if _termination.Prolonged {
			waitCtx, cancel = context.WithDeadline(context.Background(), _termination.Deadline)
		}
	}()

	waitFn(waitCtx)
	cancel()

	os.Exit(100)
}

// RequestSlowTermination request some time during termination process
// When Shutdown process starts application has only one second or so
// to finish everything gracefully or will be killed
// In some rare cases you can call this function once and get an extra time
// But in the end application will be killed anyway
func RequestSlowTermination(reason string) error {
	if reason == "" {
		return SlowShutdownNeedAReason
	}
	_termination.mu.Lock()
	defer _termination.mu.Unlock()
	if !_termination.Started {
		return SlowShutdownWithoutTermination
	}
	if _termination.Prolonged {
		return SlowShutdownRequestedAlready
	}
	_termination.Prolonged = true
	_termination.Deadline = _termination.Deadline.Add(prolongedShutdownTimeout)
	log.Warnf("Slow shutdown has beed requested with reason '%s'", reason)
	return nil
}

// GetTerminationDeadline return deadline time when application will be killed
// If Termination process hasn't been launched it will be IsZero
func GetTerminationDeadline() time.Time {
	_termination.mu.RLock()
	defer _termination.mu.RUnlock()
	if !_termination.Started {
		return time.Time{}
	}
	return _termination.Deadline
}

// BindHandlerToSignals attaches `handler` pointer to specified list of signals.
// If specified handler function is already added to particular signal then nothig will happen
// handler will be called in reverse order
// You have a control to stop signal handlers chaining just returning OSSignalStopPropagationError  from your handler
// All other error will be ignored.
// You can attach to SIGTERM and os.Interrupt but can not stop propagating them. These signals are treated differently
// And here can be used as instant notification
// It is not recommended to launch anything heavy inside a signal handler.
// For this reason in your function you will have TWO context.Context arguments. first is a global one but
// second is set to only 100ms for all handlers on current signal(!). You need to do your work or create goroutine (in global context)
// If signal handler don't want or ignore local ctx 100ms timer, it is very sad. Nobody kills you but
// if this second ctx is in Done state it means you are hanging the system and this situation will be logged
// One more point. Handler will receive a global context as the first parameter. If it is in Done state you MUST
// imidiately close all your work and all your goroutines work. This context is in Done state when whole application is terminating.
// But we want to terminate gracefully so you have one second and moreover you can request more time
// using common.RequestSlowTermination() but only once and not for a long time too
// If you don't check this context and/or ignore then application-wide watchdog just kill everything.
func BindHandlerToSignals(handler *OSSignalHandler, signals ...os.Signal) {
	if handler == nil || len(signals) == 0 {
		return
	}

	_signalsMutex.Lock()
	defer _signalsMutex.Unlock()

	for _, sig := range signals {
		if sig == nil {
			continue
		}
		handlerFound := false
		for _, h := range _signals[sig] {
			if h == handler {
				handlerFound = true
				break
			}
		}
		if !handlerFound {
			if _, ok := _signals[sig]; !ok {
				_carefullyNotifySignal(sig)
			}
			_signals[sig] = append(_signals[sig], handler)
		}
	}
}

func isTerminationSignal(sig os.Signal) bool {
	for _, termSignal := range _defaultTermSignals {
		if sig == termSignal {
			return true
		}
	}
	return false
}

func _carefullyNotifySignal(sig os.Signal) {
	if !isTerminationSignal(sig) {
		signal.Notify(_signalChan, sig)
	}
}

func _carefullyResetSignal(sig os.Signal) {
	if !isTerminationSignal(sig) {
		signal.Reset(sig)
	}
}

// UnbindHandlers detaches a list of handlers from any attached signals
func UnbindHandlers(handlers ...*OSSignalHandler) {
	if len(handlers) == 0 {
		return
	}

	_signalsMutex.Lock()
	defer _signalsMutex.Unlock()

	signalsToDelete := []os.Signal{}
	for sig, sigHandlers := range _signals {
		_signals[sig] = _removeFromSignalsHandlers(sigHandlers, handlers)
		if len(_signals[sig]) == 0 {
			signalsToDelete = append(signalsToDelete, sig)
		}
	}
	for _, sig := range signalsToDelete {
		_carefullyResetSignal(sig)
		delete(_signals, sig)
	}
}

// UnbindHandlerFromSignals detaches particular handler from specified list
// of signals
func UnbindHandlerFromSignals(handler *OSSignalHandler, signals ...os.Signal) {
	if handler == nil || len(signals) == 0 {
		return
	}

	_signalsMutex.Lock()
	defer _signalsMutex.Unlock()

	for _, sig := range signals {
		if sig == nil {
			continue
		}
		sigHandlers, sigFound := _signals[sig]
		if !sigFound {
			continue
		}
		_signals[sig] = _removeFromSignalsHandlers(sigHandlers, []*OSSignalHandler{handler})
		if len(_signals[sig]) == 0 {
			_carefullyResetSignal(sig)
			delete(_signals, sig)
		}
	}
}

// Must be run under mutex protection
func _removeFromSignalsHandlers(origin []*OSSignalHandler, removed []*OSSignalHandler) []*OSSignalHandler {
	updated := false
	newHandlers := make([]*OSSignalHandler, 0, cap(origin))
	for _, originHandler := range origin {
		for _, removingHandler := range removed {
			if removingHandler == nil {
				continue
			}
			if removingHandler != originHandler {
				newHandlers = append(newHandlers, originHandler)
			} else {
				updated = true
			}
		}
	}
	if updated {
		return newHandlers
	} else {
		return origin
	}
}

// UnbindSignals detaches all the handlers from specified list of signals
func UnbindSignals(signals ...os.Signal) {
	if len(signals) == 0 {
		return
	}

	_signalsMutex.Lock()
	defer _signalsMutex.Unlock()

	for _, sig := range signals {
		_carefullyResetSignal(sig)
		delete(_signals, sig)
	}
}

// UnbindAllSignals resets all signals to theirs defaults
func UnbindAllSignals() {
	_signalsMutex.Lock()
	defer _signalsMutex.Unlock()

	_signals = make(map[os.Signal][]*OSSignalHandler)
}

// EnableDefaultSigTermHandler enables default termination handler for
// Any custom termination handler registered before will be dropped
// signals os.Interrupt and syscall.SIGTERM
func EnableDefaultSigTermHandler() {
	_defaultSigTermHandler = defaultSigTermHandler
}

// SetSigTermHandler sets a custom termination handler
// CAUTION: the second parameter (os.Signal can be nil in case if it is an internal termination request)
func SetSigTermHandler(fn OSSigTermHandler) {
	_defaultSigTermHandler = fn
}

// DisableSigTermHandler disables default termination handler for termination
// signals such as os.Interrupt and syscall.SIGTERM or internal termination request
// In this case only default GO behaviour remains (just os.Exit())
func DisableSigTermHandler() {
	_defaultSigTermHandler = nil
}
