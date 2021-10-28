package xcommon

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"

	"os"
	"sync"

	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
)

type GlobalState struct {
	subcommandName    atomic.Value // string
	subcommandHandler atomic.Value // CommandHandler
}

var StateMu sync.RWMutex
var State *GlobalState = &GlobalState{}

func (gs *GlobalState) GetSubcommandName() string {
	return gs.subcommandName.Load().(string)
}

type AppInitFuncType = func(ctx context.Context) error
type AppMainFuncType = func(ctx context.Context, commandHandler CommandHandler) error
type AppDownFuncType = func(ctx context.Context)

// AppRun is the entrypoint of your application. When it exits your main should exit too
// It creates a global context, configure your application, setup signal watcher and maintain lifecycle
// Please DO NOT underestimate using context.Context. Use it everywhere. Check it everywhere. Pass it everywhere.
// Select on this ctx in every worker goroutine
// By closing this context we notify the whole application about termination phase. When ctx is Done you have only one second
// (or more your code will request it). So please, gracefully shut down all your processes and goroutines
// You can pass to this function tree optional hooks
// AppInit: Run at very beginning. Before signal watcher is functional. This function whould configure the application
// at AppInit phase all errors are fatal. If this function returns an error the application will display it and exit
// AppInit default implementations just do nothing. AppInit sets that there is no command
// AppMain: a main function of your application. it can be your specified function or according to configuration plan it can be a subcommand/nocommand handler function
// AppMain default implementation calls subcommand/nocommand handler if any or just a DefaultNoCommandHandler that can react only on --help flag
// AppDown default is really just a no-op
func AppRun(appInitFn AppInitFuncType, appMainFn AppMainFuncType, appDownFn AppDownFuncType) {
	ctx, stop := GetGlobalContext()
	defer stop()

	// Initialization sequence
	var err error
	if appInitFn == nil {
		err = stubAppInit(ctx)
	} else {
		err = appInitFn(ctx)
	}
	if err != nil {
		log.WithError(err).Fatal("Initialization failed")
		os.Exit(1)
	}
	SetupSignalWatcher(ctx)
	log.Debug("Initialization's done")

	// Running sequence
	subcommandName := State.subcommandName.Load().(string)
	subcommandHandler := State.subcommandHandler.Load().(CommandHandler)
	if subcommandHandler == nil {
		log.WithError(errors.New("subcommand can not be nil in internal!")).Fatal("Initialization failed")
	}
	if appMainFn != nil {
		err = appMainFn(ctx, subcommandHandler)
	} else {
		err = subcommandHandler(ctx)
	}
	if err != nil {
		_log := log.WithError(err)
		if subcommandName != "" {
			_log = _log.WithField("subcommand", subcommandName)
		}
		if appMainFn == nil {
			_log.Errorf("Subcommand returned error")
		} else {
			_log.Errorf("AppMain returned error")
		}
	}

	// Finalizing sequence
	if appDownFn != nil {
		appDownFn(ctx)
	}
}

func stubAppInit(ctx context.Context) error {
	State.subcommandName.Store("")
	State.subcommandHandler.Store(DefaultNoCommandHandler)
	return nil
}

// DefaultNoCommandHandler can process some general flags (like --help which just prints available commands).
// Please call it from your custom NoCommandHandler or implement it in your way
func DefaultNoCommandHandler(ctx context.Context) error {
	if flag := flag.Lookup("help"); flag != nil && flag.Changed && _cmdlineMap != nil {
		printGeneralHelp()
	}
	return nil
}

func printGeneralHelp() {
	fmt.Fprintln(os.Stderr, "Available commands:")
	cmdIndent := strings.Repeat(" ", 4)
	descriptionIndent := strings.Repeat(" ", 8)
	cmds := _getSubcommandList()
	for _, cmd := range cmds {
		fmt.Fprintf(os.Stderr, "%s%s\n", cmdIndent, cmd.Name)
		if cmd.Description != "" {
			// TODO: word-wrap description
			fmt.Fprintf(os.Stderr, "%s%s\n", descriptionIndent, cmd.Description)
		}
		fmt.Fprintln(os.Stderr)
	}
	fmt.Fprintln(os.Stderr, "For each command --help is available for more information")
}
