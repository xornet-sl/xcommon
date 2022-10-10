package xcommon

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/pflag"
)

type subcommandGroupsShort struct {
	Description string
	Commands    []subcommandShort
}

type subcommandShort struct {
	Name        string
	Description string
}

func updateUsages(cmdlineMap *CmdlineMap) {
	pflag.Usage = func() {}
}

func printGeneralHelp(cmdlineMap *CmdlineMap) {
	// print(cmdlineMap.NoCommandFlagSet.FlagUsages())
}

func _printGeneralHelp(cmdlineMap *CmdlineMap) {
	fmt.Fprintln(os.Stderr, "Available commands:")
	cmdGroupIndent := ""
	cmdIndent := ">>  "
	descriptionIndent := strings.Repeat(" ", 8)
	groups := _getSubcommandGroupList(cmdlineMap)
	emptyIdx := -1
	var cmdGroup subcommandGroupsShort
	for idx := 0; idx < len(groups); idx++ {
		cmdGroup = groups[idx]
		if groups[idx].Description == "" && idx+1 < len(groups) {
			emptyIdx = idx
			continue
		} else if emptyIdx > 0 && idx+1 == len(groups) || groups[idx].Description == "" {
			if emptyIdx > 0 {
				cmdGroup = groups[emptyIdx]
			}
			fmt.Fprintf(os.Stderr, "%s%s:\n", cmdGroupIndent, "Other commands")
		} else {
			fmt.Fprintf(os.Stderr, "%s%s:\n", cmdGroupIndent, strings.TrimRight(cmdGroup.Description, ":"))
		}
		for _, cmd := range cmdGroup.Commands {
			fmt.Fprintf(os.Stderr, "%s%s\n", cmdIndent, cmd.Name)
			if cmd.Description != "" {
				// TODO: word-wrap description
				fmt.Fprintf(os.Stderr, "%s%s\n", descriptionIndent, cmd.Description)
			}
			fmt.Fprintln(os.Stderr)
		}
		if idx+1 < len(groups) {
			fmt.Fprintln(os.Stderr)
		}
	}
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "For each command --help is available for more information")
}

func _getSubcommandGroupList(cmdlineMap *CmdlineMap) []subcommandGroupsShort {
	if cmdlineMap == nil {
		return []subcommandGroupsShort{}
	}
	ret := make([]subcommandGroupsShort, 0, len(cmdlineMap.CommandGoups))
	for _, cmdGroup := range cmdlineMap.CommandGoups {
		if cmdGroup.Hidden {
			continue
		}
		retSubcommands := make([]subcommandShort, 0, len(cmdGroup.Commands))
		for _, cmd := range cmdGroup.Commands {
			if !cmd.Hidden {
				retSubcommands = append(retSubcommands, subcommandShort{
					Name:        cmd.Name,
					Description: cmd.Description,
				})
			}
		}
		if len(retSubcommands) > 0 {
			ret = append(ret, subcommandGroupsShort{
				Description: cmdGroup.Description,
				Commands:    retSubcommands,
			})
		}
	}
	return ret
}
