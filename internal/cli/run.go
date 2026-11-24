package cli

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/JoaoASimoes/pokedex/internal/dex"
	"github.com/JoaoASimoes/pokedex/internal/tui"
)

// Run executes the program with the given arguments and returns the exit code.
func Run(prog string, args []string) int {
	fs := flag.NewFlagSet(prog, flag.ContinueOnError)
	refresh := fs.Bool("refresh", false, "ignore the local cache")
	usage := func() {
		fmt.Fprintf(os.Stderr, `Terminal Pokédex and type weakness calculator (PokeAPI).

Usage:
  %[1]s [--refresh]                       interactive TUI
  %[1]s [--refresh] pokemon|p <name>       weaknesses of a Pokémon
  %[1]s [--refresh] type|t <type> [type2]  weaknesses of 1 or 2 types
  %[1]s [--refresh] types                  list available types

Options:
`, prog)
		fs.PrintDefaults()
	}
	fs.Usage = usage
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	args = fs.Args()

	d := dex.NewClient(*refresh)
	ok := true
	var err error
	if len(args) == 0 {
		if !isTerminal(os.Stdin) || !isTerminal(os.Stdout) {
			fmt.Fprintln(os.Stderr, "The interactive TUI needs a terminal; use a subcommand instead.")
			fmt.Fprintln(os.Stderr)
			usage()
			return 2
		}
		err = tui.Run(d)
	} else {
		d.Progress = os.Stderr // the TUI has its own spinner instead
		switch args[0] {
		case "pokemon", "p":
			if len(args) < 2 {
				usage()
				return 2
			}
			ok, err = cmdPokemon(d, strings.Join(args[1:], "-"), stdinPrompter())
		case "type", "t":
			ok, err = cmdType(d, args[1:], stdinPrompter())
		case "types":
			err = cmdTypes(d)
		case "help":
			usage()
		default:
			fmt.Fprintf(os.Stderr, "Unknown command '%s'.\n\n", args[0])
			usage()
			return 2
		}
	}
	if err != nil {
		if errors.Is(err, dex.ErrNotFound) {
			err = errors.New("not found on PokeAPI")
		}
		fmt.Fprintln(os.Stderr, red("Error: "+err.Error()))
		return 1
	}
	if !ok {
		return 1
	}
	return 0
}
