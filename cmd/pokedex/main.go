// Command pokedex is a terminal Pokédex: entries, stats, abilities, moves, items,
// evolutions, natures and type matchups, with data from PokeAPI.
//
// Usage:
//
//	pokedex                        interactive TUI
//	pokedex pokemon <name>         weaknesses of a Pokémon
//	pokedex type <type1> [type2]   weaknesses of 1 or 2 types
//	pokedex types                  list available types
//	pokedex config <command>       manage the config file
//	pokedex version                print the version
package main

import (
	"os"

	"github.com/JoaoASimoes/pokedex/internal/cli"
)

// version is set at build time by the release process:
// go build -ldflags "-X main.version=v1.2.3" ./cmd/pokedex
var version = "dev"

func main() {
	cli.Version = version
	os.Exit(cli.Run(os.Args[0], os.Args[1:]))
}
