# Type Calc

A small command-line tool that shows the type weaknesses, resistances and immunities of a
Pokémon, or of any combination of one or two types.

Data comes from [PokeAPI](https://pokeapi.co) and is cached locally, so after the first run
lookups are instant and work offline.

```
$ type_calc pokemon garchomp

#445 Garchomp
Type(s): Dragon / Ground

  4×   Very weak         Ice
  2×   Weak              Dragon, Fairy
  ½×   Resists           Fire, Poison, Rock
  0×   Immune            Electric
```

## Features

- **Search by Pokémon** – every species and form known to PokeAPI (Mega, Gmax, regional forms, …).
- **Search by type** – one or two types, e.g. `fire flying`.
- **Typo tolerant** – misspelled names and types are matched by prefix, edit distance and
  substring. A single good match is used automatically; otherwise you get a numbered list to
  pick from.
  ```
  $ type_calc type steel fary
  'fary' not found; using 'fairy'.

  $ type_calc pokemon giratina
  'giratina' not found. Did you mean:
    1. giratina-origin
    2. giratina-altered
  Number (Enter to cancel):
  ```
- **Interactive mode** with TAB autocompletion for commands, Pokémon names and types, plus
  command history (↑/↓).
- **Ability warnings** – flags abilities that change matchups (Levitate, Flash Fire,
  Thick Fat, Wonder Guard, …).
- Coloured output (disabled automatically when piped, or with `NO_COLOR=1`).

## Installation

Requires [Go](https://go.dev/dl/) 1.22 or newer.

```bash
git clone git@github.com:JoaoASimoes/type_calc.git
cd type_calc
make build        # or: go build -o type_calc .
```

Optionally move the binary somewhere on your `PATH`:

```bash
sudo install type_calc /usr/local/bin/
```

### Cross-compiling

```bash
make dist
```

Produces binaries for Linux, macOS and Windows (amd64/arm64) in `dist/`.

## Usage

```
type_calc [--refresh]                         interactive mode
type_calc [--refresh] pokemon|p <name>        weaknesses of a Pokémon
type_calc [--refresh] type|t <type> [type2]   weaknesses of 1 or 2 types
type_calc [--refresh] types                   list available types
```

`--refresh` ignores the cache and downloads the data again. It must come before the command.

### Interactive mode

Run `type_calc` without arguments:

```
> p pikachu
> t fire flying
> types
> help
> quit
```

Press TAB to complete a command, Pokémon name or type; press it twice to list all options.

## Cache

Downloaded data is stored as JSON in your user cache directory
(`~/.cache/type_calc/` on Linux, `~/Library/Caches/type_calc/` on macOS,
`%LocalAppData%\type_calc\` on Windows) and refreshed after 30 days. Delete that folder or
use `--refresh` to force an update.

## Project layout

| File          | Purpose                                                  |
|---------------|----------------------------------------------------------|
| `main.go`     | CLI, interactive mode, autocompletion and output         |
| `pokeapi.go`  | PokeAPI client and local cache                           |
| `fuzzy.go`    | Name normalisation and typo-tolerant suggestions         |
| `*_test.go`   | Unit tests (`make test`)                                 |

## Development

```bash
make test   # run unit tests
make vet    # static checks
make fmt    # format code
```

## Notes

- Abilities are only shown as warnings; the multipliers always reflect types alone.
- Generation-specific mechanics (e.g. older type charts) and Tera types are not modelled.

## License

[MIT](LICENSE). Pokémon and Pokémon names are trademarks of Nintendo, Game Freak and
Creatures Inc. This project is not affiliated with them. Data provided by
[PokeAPI](https://pokeapi.co).
