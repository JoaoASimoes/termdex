// Command termdex is a terminal Pokédex: entries, stats, abilities, moves, items,
// evolutions, natures and type matchups, with data from PokeAPI.
//
// Usage:
//
//	termdex                        interactive TUI
//	termdex pokemon <name>         weaknesses of a Pokémon
//	termdex type <type1> [type2]   weaknesses of 1 or 2 types
//	termdex types                  list available types
//	termdex config <command>       manage the config file
//	termdex version                print the version
package main

import (
	"os"

	"github.com/JoaoASimoes/termdex/internal/cli"
)

// version is set at build time by the release process:
// go build -ldflags "-X main.version=v1.2.3" ./cmd/termdex
var version = "dev"

func main() {
	cli.Version = version
	os.Exit(cli.Run(os.Args[0], os.Args[1:]))
}
