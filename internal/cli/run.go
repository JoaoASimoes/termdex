package cli

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/JoaoASimoes/pokedex/internal/config"
	"github.com/JoaoASimoes/pokedex/internal/dex"
	"github.com/JoaoASimoes/pokedex/internal/tui"
)

// Run executes the program with the given arguments and returns the exit code.
func Run(prog string, args []string) int {
	fs := flag.NewFlagSet(prog, flag.ContinueOnError)
	refresh := fs.Bool("refresh", false, "ignore the local cache and download the data again")
	cfgPath := fs.String("config", "", "config file (default "+config.DefaultPath()+")")
	usage := func() {
		fmt.Fprintf(os.Stderr, `Terminal Pokédex and type weakness calculator (PokeAPI).

Usage:
  %[1]s [options]                         interactive TUI
  %[1]s [options] pokemon|p <name>        weaknesses of a Pokémon
  %[1]s [options] type|t <type> [type2]   weaknesses of 1 or 2 types
  %[1]s [options] types                   list available types
  %[1]s [options] config <command>        manage the config file:
        path               print where the config file is read from
        init [--force]     write the default config there, with comments
        show               print the effective configuration
        check              validate the config file

Options (before the command):
`, prog)
		fs.PrintDefaults()
		fmt.Fprintln(os.Stderr, "\nSee docs/configuration.md for every setting and key binding.")
	}
	fs.Usage = usage
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	args = fs.Args()

	path := *cfgPath
	if path == "" {
		path = config.DefaultPath()
	}
	if len(args) > 0 && args[0] == "config" {
		return runConfig(path, args[1:], usage)
	}

	cfg, err := config.Load(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, red("Config error: "+err.Error()))
		fmt.Fprintln(os.Stderr, "Fix the file (pokedex config check), or run with --config /dev/null to use the defaults.")
		return 2
	}

	d := dex.NewClient(dex.Options{
		Refresh:     *refresh,
		CacheTTL:    cfg.General.CacheTTL(),
		MaxRequests: cfg.General.MaxRequests,
	})
	ok := true
	if len(args) == 0 {
		if !isTerminal(os.Stdin) || !isTerminal(os.Stdout) {
			fmt.Fprintln(os.Stderr, "The interactive TUI needs a terminal; use a subcommand instead.")
			fmt.Fprintln(os.Stderr)
			usage()
			return 2
		}
		err = tui.Run(d, cfg)
	} else {
		d.Progress = os.Stderr // the TUI has its own spinner instead
		switch args[0] {
		case "pokemon", "p":
			if len(args) < 2 {
				usage()
				return 2
			}
			ok, err = cmdPokemon(d, strings.Join(args[1:], "-"), stdinPrompter(), cfg.General.Sprites)
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

// runConfig implements `pokedex config <command>`.
func runConfig(path string, args []string, usage func()) int {
	if len(args) == 0 {
		usage()
		return 2
	}
	switch args[0] {
	case "path":
		fmt.Println(path)
		if _, err := os.Stat(path); err != nil {
			fmt.Fprintln(os.Stderr, "(the file does not exist yet: defaults are used; create it with `config init`)")
		}
	case "init":
		force := len(args) > 1 && (args[1] == "--force" || args[1] == "-f")
		if err := config.Write(path, force); err != nil {
			fmt.Fprintln(os.Stderr, red("Error: "+err.Error()))
			return 1
		}
		fmt.Println("Wrote the default configuration to " + path)
	case "show":
		cfg, err := config.Load(path)
		if err != nil {
			fmt.Fprintln(os.Stderr, red("Config error: "+err.Error()))
			return 1
		}
		fmt.Printf("# Effective configuration (%s + defaults)\n", path)
		fmt.Print(cfg.TOML())
	case "check":
		if _, err := config.Load(path); err != nil {
			fmt.Fprintln(os.Stderr, red("Config error: "+err.Error()))
			return 1
		}
		if _, err := os.Stat(path); err != nil {
			fmt.Println("No config file at " + path + ": using the defaults.")
		} else {
			fmt.Println(path + ": OK")
		}
	default:
		fmt.Fprintf(os.Stderr, "Unknown config command '%s'.\n\n", args[0])
		usage()
		return 2
	}
	return 0
}
