package xcommon

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/pflag"
)

var UnknownCommandlineCommandError = errors.New("Unknown command line command")
var CreateCommandWithoutHandlerError = func(cmd string) error {
	return errors.New(fmt.Sprintf("Every command should have a handler! '%s' doesn't have one!", cmd))
}

type CommandHandler func(ctx context.Context) error

type CmdlineCommandGroup struct {
	Description  string           // Can be used like a chapter in help. Can be empty. By default in help they won't be separated
	Commands     []CmdlineCommand // A list of commands
	GroupFlagSet *pflag.FlagSet   // Flags that are common for group
	Hidden       bool             // Is this group of commands are hidden from help (some debug/special commands)
}

type CmdlineCommand struct {
	Name           string         // Name of the command (case-insensitive)
	Description    string         // Description (can be displayed in help)
	Aliases        []string       // list of aliases for command (case-insensitive)
	CommandFlagSet *pflag.FlagSet // Flags for this command
	Handler        CommandHandler // Handler func for this command
	Hidden         bool           // Is this command hidden from help? (e.g. some debug or service functionality)
}

// CmdlineMap is a struct that can help you declare your cmdline interface
// Supported commandful and commandless interfaces
// If commands declared they should be specified by user as a first positional argument
type CmdlineMap struct {
	CommonFlagSet    *pflag.FlagSet        // All common flags (e.g. log levels, config sources, etc.)
	CommandGoups     []CmdlineCommandGroup // List of supported command groups. All commands are case-insensitive and must be uniq
	NoCommandFlagSet *pflag.FlagSet        // Flagset that is active when user don't specify command
	NoCommandHandler CommandHandler        // Handler func in case if there is no command specified
}

func parseCmdLine(cmdlineMap *CmdlineMap) (string, CommandHandler, error) {
	if cmdlineMap.CommonFlagSet != nil {
		pflag.CommandLine.AddFlagSet(cmdlineMap.CommonFlagSet)
	}

	if len(os.Args) < 2 {
		// No command line at all
		return "", cmdlineMap.NoCommandHandler, nil
		// if cmdlineMap.NoCommandHandler != nil {
		// 	return "", cmdlineMap.NoCommandHandler, nil
		// } else {
		// 	return "", GetDefaultNoCommandHandler(&cmdlineMap), nil
		// }
	}

	cmdFound := false
	foundCommand := ""
	var foundHandler CommandHandler = nil
	arg := strings.ToLower(os.Args[1])
	lower := strings.ToLower(arg)

	for _, cmdGroup := range cmdlineMap.CommandGoups {
		for _, cmd := range cmdGroup.Commands {
			if lower == strings.ToLower(cmd.Name) {
				cmdFound = true
			} else {
				for _, alias := range cmd.Aliases {
					if lower == strings.ToLower(alias) {
						cmdFound = true
						break
					}
				}
			}
			if cmdFound {
				foundCommand = cmd.Name
				foundHandler = cmd.Handler
				if cmdGroup.GroupFlagSet != nil {
					pflag.CommandLine.AddFlagSet(cmdGroup.GroupFlagSet)
				}
				if cmd.CommandFlagSet != nil {
					pflag.CommandLine.AddFlagSet(cmd.CommandFlagSet)
				}
				break
			}
		}
		if cmdFound {
			break
		}
	}
	if cmdFound && foundCommand == "" {
		// Unacceptable command
		return "", nil, UnknownCommandlineCommandError
	}
	if cmdFound && foundHandler == nil {
		return "", nil, CreateCommandWithoutHandlerError(foundCommand)
	}
	if !cmdFound {
		if cmdlineMap.NoCommandFlagSet != nil {
			pflag.CommandLine.AddFlagSet(cmdlineMap.NoCommandFlagSet)
		}
		foundCommand = ""
		if cmdlineMap.NoCommandHandler != nil {
			foundHandler = cmdlineMap.NoCommandHandler
		} else {
			// if pflag.CommandLine.Lookup("help") == nil {
			// 	pflag.BoolP("help", "h", false, "Display help")
			// }
			// foundHandler = GetDefaultNoCommandHandler(&cmdlineMap)
			foundHandler = nil
		}
	}

	updateUsages(cmdlineMap)

	pflag.Parse()
	return foundCommand, foundHandler, nil
}

// GetDefaultNoCommandHandler can process some general flags (like --help which just prints available commands).
// Please call it from your custom NoCommandHandler or implement it in your way
func GetDefaultNoCommandHandler(cmdlineMap *CmdlineMap) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		// if flag := pflag.Lookup("help"); flag != nil && flag.Changed && cmdlineMap != nil {
		// 	printGeneralHelp(cmdlineMap)
		// }
		return nil
	}
}
