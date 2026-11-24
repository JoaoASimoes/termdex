// Command pokedex is a terminal Pokédex: entries, stats, abilities, moves, evolutions,
// natures and type matchups, with data from PokeAPI.
//
// Usage:
//
//	pokedex                        interactive TUI
//	pokedex pokemon <name>         weaknesses of a Pokémon
//	pokedex type <type1> [type2]   weaknesses of 1 or 2 types
//	pokedex types                  list available types
//	pokedex --refresh ...          ignore the cache and download the data again
package main

import (
	"os"

	"github.com/JoaoASimoes/pokedex/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[0], os.Args[1:]))
}
