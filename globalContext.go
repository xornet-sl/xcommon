package xcommon

import (
	"context"
	"sync"
)

var (
	_ctx       context.Context
	_ctxCancel context.CancelFunc
)

func init() {
	ctx, cancel := context.WithCancel(context.Background())
	_ctx = ctx
	_ctxCancel = cancel
	go _runFinalizers()
}

func GetGlobalContext() (context.Context, context.CancelFunc) {
	return _ctx, contextStoper
}

func contextStoper() {
	defer _ctxCancel()
}

var _finalizers = []func(){}
var _finalizersMu sync.Mutex

// RegisterFinalizer registers functions that shoyld be run when global context is done
// i.e. when application is shutting down.
// There is a grace period that you can look at signals.go
// Also in rare cases you can extend grace period you need
func RegisterFinalizer(fn func()) {
	_finalizersMu.Lock()
	defer _finalizersMu.Unlock()
	_finalizers = append(_finalizers, fn)
}

func _runFinalizers() {
	select {
	case <-_ctx.Done():
		for idx := len(_finalizers) - 1; idx >= 0; idx-- {
			_finalizers[idx]()
		}
	}
}
