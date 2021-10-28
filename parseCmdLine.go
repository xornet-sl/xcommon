package xcommon

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	flag "github.com/spf13/pflag"
)

var UnknownCommandlineCommandError = errors.New("Unknown command line command")

type CommandHandler func(ctx context.Context) error

type CmdlineCommand struct {
	Name           string         // Name of the command (case-insensitive)
	Description    string         // Description (can be displayed in help)
	Aliases        []string       // list of aliases for command (case-insensitive)
	CommandFlagSet *flag.FlagSet  // Flags for this command
	Handler        CommandHandler // Handler func for this command
	Hidden         bool           // Is this command hidden from help? (e.g. some debug or service functionality)
}

// CmdlineMap is a struct that can help you declare your cmdline interface
// Supported commandful and commandless interfaces
// If commands declared they should be specified by user as a first positional argument
type CmdlineMap struct {
	CommonFlagSet    *flag.FlagSet    // All common flags (e.g. log levels, config sources, etc.)
	Commands         []CmdlineCommand // List of supported commands. All commands are case-insensitive
	NoCommandFlagSet *flag.FlagSet    // Flagset that is active when user don't specify command
	NoCommandHandler CommandHandler   // Handler func in case if there is no command specified
}

var _cmdlineMap *CmdlineMap = nil

// parseCmdLine parses the commandline string. Returns found command or error. All parsed flags accessible using flag package
// If no command given then function will select which handler to use. User specified or DefaultNoCommandHandler
// DefaultNoCommandHandler by default can only display help using flag --help
// if no arguments given at all then using DefaultNoCommandHandler which will do nothing because it takes at least --help flag
// Users can specify their NoCommandHandler but it is highly recommended that user-specified handler will call DefaultNoCommandHandler to support default no-command --help
func parseCmdLine(cmdlineMap CmdlineMap) (string, CommandHandler, error) {
	if cmdlineMap.CommonFlagSet != nil {
		flag.CommandLine.AddFlagSet(cmdlineMap.CommonFlagSet)
	}

	if len(os.Args) < 2 {
		// no command line at all
		if cmdlineMap.NoCommandHandler != nil {
			return "", cmdlineMap.NoCommandHandler, nil
		} else {
			return "", DefaultNoCommandHandler, nil
		}
	}

	cmdFound := false
	foundCommand := ""
	var foundHandler CommandHandler = nil
	arg := strings.ToLower(os.Args[1])
	lower := strings.ToLower(arg)
	for _, cmd := range cmdlineMap.Commands {
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
			if cmd.CommandFlagSet != nil {
				flag.CommandLine.AddFlagSet(cmd.CommandFlagSet)
			}
			break
		}
	}
	if cmdFound && foundCommand == "" {
		// Unacceptable command
		return "", nil, UnknownCommandlineCommandError
	}
	if cmdFound && foundHandler == nil {
		return "", nil, errors.New(fmt.Sprintf("Every command should have a handler! '%s' Doesn't have one!", foundCommand))
	}
	if !cmdFound {
		if cmdlineMap.NoCommandFlagSet != nil {
			flag.CommandLine.AddFlagSet(cmdlineMap.NoCommandFlagSet)
		}
		foundCommand = ""
		if cmdlineMap.NoCommandHandler != nil {
			foundHandler = cmdlineMap.NoCommandHandler
		} else {
			if flag.CommandLine.Lookup("help") == nil {
				flag.BoolP("help", "h", false, "Display help")
			}
			foundHandler = DefaultNoCommandHandler
		}
	}
	flag.Parse()
	_cmdlineMap = &cmdlineMap
	return foundCommand, foundHandler, nil
}

type subcommandShort struct {
	Name        string
	Description string
}

func _getSubcommandList() []subcommandShort {
	if _cmdlineMap == nil {
		return []subcommandShort{}
	}
	ret := make([]subcommandShort, 0, len(_cmdlineMap.Commands))
	for _, cmd := range _cmdlineMap.Commands {
		if !cmd.Hidden {
			ret = append(ret, subcommandShort{
				Name:        cmd.Name,
				Description: cmd.Description,
			})
		}
	}
	return ret
}
