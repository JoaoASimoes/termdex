# Pokédex

A terminal Pokédex. Look up any Pokémon to see its entry, base stats, abilities, type
weaknesses and full move pool, or check the weaknesses of any one- or two-type combination.

It has a full-screen interactive interface (TUI) and plain subcommands for quick lookups and
scripts. Data comes from [PokeAPI](https://pokeapi.co) and is cached locally, so after the
first lookup everything is instant and works offline.

```
 ◓ Pokédex

  Pokédex     Moves     Items     Natures     Type Matchups

╭────────────────────────────────╮ ╭──────────────────────────────────────────────────────────╮
│ Search Pokémon                 │ │ #0006 Charizard  Gen I                                   │
│ › charizrd                     │ │ Flame Pokémon                                            │
│                                │ │  Fire   Flying    1.7 m · 90.5 kg                        │
│ ▸ charizard                    │ │                                                          │
│                                │ │ Its wings can carry this Pokémon close to an altitude of │
│                                │ │ 4,600 feet. It blows out fire at very high temperatures. │
│                                │ │                                                          │
│                                │ │ Base stats                                               │
│                                │ │ HP         78 ██████████████░░░░░░░░░░░░░░░░░░░░░░       │
│                                │ │ Attack     84 ███████████████░░░░░░░░░░░░░░░░░░░░░       │
│                                │ │ Defense    78 ██████████████░░░░░░░░░░░░░░░░░░░░░░       │
│                                │ │ Sp. Atk   109 ███████████████████░░░░░░░░░░░░░░░░░       │
│                                │ │ Sp. Def    85 ███████████████░░░░░░░░░░░░░░░░░░░░░       │
│                                │ │ Speed     100 ██████████████████░░░░░░░░░░░░░░░░░░       │
│                                │ │ Total     534                                            │
│                                │ │                                                          │
│                                │ │ Abilities                                                │
│                                │ │ Blaze · Solar Power (hidden)                             │
│                                │ │                                                          │
│                                │ │ Weaknesses & resistances                                 │
│                                │ │ 4×  Very weak          Rock                              │
│                                │ │ 2×  Weak               Electric   Water                  │
│                                │ │ ½×  Resists            Fairy   Fighting   Fire   Steel   │
│                                │ │ ¼×  Strongly resists   Bug   Grass                       │
│                                │ │ 0×  Immune             Ground                            │
╰────────────────────────────────╯ ╰──────────────────────────────────────────────────────────╯
↑/↓ ctrl+k/j select • enter open entry • ctrl+h/l tabs • ctrl+t types • esc clear/quit
```

## Features

- **Pixel-art sprites** – every Pokémon's game sprite drawn right in the terminal with
  half-block characters (two pixels per cell, 24-bit colour).
- **Pokédex search** – every species and form known to PokeAPI (Mega, Gmax, regional forms, …),
  listed in National Pokédex order with each form right after its species. Filter with
  `type:fire`, `gen:1` and combinations (`type:fire type:flying gen:1 char`).
  The entry of the highlighted suggestion is shown live while you type or scroll: category,
  Pokédex description, height/weight, base stats, abilities (including the hidden one),
  evolutions (what it evolves from/into and how: level, item, trade, friendship, time of
  day, …) and weaknesses.
- **Full entry** (press Enter). A breadcrumb under the title shows where you are
  (`Search › Charmeleon › Charizard`) and where Esc takes you back to. Four tabs:
  - **Overview** – description, stats, weaknesses/resistances, abilities and the full
    evolution tree with the conditions for each step.
  - **Abilities** – what each ability does (short and full effect, in-game description).
  - **Moves** – the move pool per game (level-up, TM, tutor, egg moves…), with type and
    category. Select a move to see its power, accuracy, PP, priority, effect chance and
    effect. Filter with `/` and switch games with `[` / `]`.
  - **Family** – the evolution line, the other forms of the species (Mega, Gmax, regional,
    gender forms such as Meowstic or Indeedee) and the female look when it differs (e.g.
    Pikachu). Each one is previewed with its sprite and stats; Enter opens it, and Esc walks
    back through the Pokémon you came from.
- **Moves** – every move in the games (938), searchable with typos: type, category, power,
  accuracy, PP, priority, effect chance, effect, description and the Pokémon that learn it.
  Enter opens the list of those Pokémon (also from a Pokémon's Moves and Abilities tabs);
  from there Enter opens a Pokémon, and Esc walks back.
- **Items** – every item (2,200+): sprite, category, fling power, attributes, effect,
  description and the wild Pokémon that may hold it.
- **Natures** – what each nature does (+10% / −10% stats, liked and disliked berry flavors),
  the full nature chart and a short explanation. Search by name or by stat: `speed` lists
  the natures that raise Speed, `-spa` those that lower Special Attack.
- **Type matchups** – weaknesses, resistances and immunities of one or two types.
- **Typo tolerant** – misspelled names and types are matched by prefix, edit distance and
  substring (`charizrd` → charizard, `fary` → fairy).
- **Help screen** – `?` or `F1`, on any screen (even while typing), lists every key of the
  current screen.
- **Small terminals** – panels that do not fit scroll with `shift+↑/↓` (`pgup/pgdown` in the
  search).
- **Offline friendly** – everything already downloaded works without a connection; when a
  download fails you get a clear message and `ctrl+r` to retry.
- **Ability warnings** – flags abilities that change matchups (Levitate, Flash Fire,
  Thick Fat, Wonder Guard, …).
- **Subcommands** for quick lookups, with coloured output (disabled automatically when piped,
  or with `NO_COLOR=1`).

## Installation

Requires [Go](https://go.dev/dl/) 1.26 or newer. With an older Go installed, the `go`
command downloads the right toolchain automatically (`GOTOOLCHAIN=auto`, the default).

```bash
git clone git@github.com:JoaoASimoes/pokedex.git
cd pokedex
make build        # or: go build -o pokedex ./cmd/pokedex
```

Optionally move the binary somewhere on your `PATH`:

```bash
sudo install pokedex /usr/local/bin/
```

### Cross-compiling

```bash
make dist
```

Produces binaries for Linux, macOS and Windows (amd64/arm64) in `dist/`.

## Usage

```
pokedex [options]                         interactive Pokédex (TUI)
pokedex [options] pokemon|p <name>        weaknesses of a Pokémon
pokedex [options] type|t <type> [type2]   weaknesses of 1 or 2 types
pokedex [options] types                   list available types
pokedex [options] config path|init|show|check   manage the config file
```

Options go before the command: `--refresh` ignores the cache and downloads the data again,
`--config <file>` reads another config file.

```
$ pokedex type steel fary
'fary' not found; using 'fairy'.

Type(s): Steel / Fairy

  2×   Weak              Fire, Ground
  ½×   Resists           Dark, Fairy, Flying, Grass, Ice, Normal, Psychic, Rock
  ¼×   Strongly resists  Bug
  0×   Immune            Dragon, Poison
```

When a name is ambiguous, the subcommands list the candidates and ask you to pick one.

These are the default keys; all of them can be changed (see [Configuration](#configuration)).
The help line at the bottom of the TUI always shows the keys currently configured.

### Keys – search screen

| Key                            | Action                                                   |
|--------------------------------|----------------------------------------------------------|
| typing                         | update suggestions (the highlighted one is previewed)    |
| `↑` / `↓`, `ctrl+k` / `ctrl+j` | move through suggestions                                 |
| `enter`                        | Pokédex: open the full entry · Types: confirm the type   |
| `tab` / `shift+tab`            | next / previous box **in the same tab** (Type 1 ↔ Type 2); never changes tab |
| `ctrl+h` / `ctrl+l`            | previous / next tab (Pokédex, Moves, Items, Natures, Type Matchups) |
| `ctrl+f`                       | new Pokémon search (works on every screen)               |
| `ctrl+t`                       | go to Type Matchups (works on every screen)              |
| `esc`                          | clear the field; on an empty field, quit                 |
| `ctrl+c`                       | quit                                                     |

Enter always picks the highlighted type, also when the field is empty and you scrolled the
list. To look up a single type, leave Type 2 empty or pick "— none (single type) —" at the
top of its list.

### Keys – Pokémon entry

| Key                                  | Action                                    |
|--------------------------------------|-------------------------------------------|
| `ctrl+h` / `ctrl+l`, `←` / `→`, `h` / `l`, `1`–`4` | switch tab (Overview, Abilities, Moves, Family) |
| `enter` (Family tab)                 | open the selected evolution or form       |
| `↑` / `↓`, `j` / `k`, `ctrl+k/j`     | select ability / move                     |
| `pgup` / `pgdn`, `g` / `G`           | page / jump to start or end of the list   |
| `[` / `]`                            | previous / next game (Moves tab)          |
| `/`                                  | filter moves by name, type, category or method |
| `esc`                                | clear the filter, go back to the previous Pokémon, or back to search |
| `ctrl+f` / `ctrl+t`                  | new Pokémon search / Type Matchups        |

## Configuration

Optional TOML file at `~/.config/pokedex/config.toml` (Linux; see the docs for other
systems). It sets every key binding, the colours of the interface (accent, headings,
labels, type badges…) and a few general options: cache lifetime, number of simultaneous
requests, preview delay and sprites on/off.

```bash
pokedex config init     # write the commented defaults (same as configs/config.example.toml)
pokedex config check    # validate after editing
```

```toml
[general]
sprites = false

[colors]
accent = "#1E66F5"

[keys.search]
down = ["down", "ctrl+d"]
up = ["up", "ctrl+u"]
```

Full reference in [`docs/`](docs/README.md):
[configuration](docs/configuration.md) and [key bindings](docs/keybindings.md).

## Cache

Downloaded data is stored as JSON in your user cache directory
(`~/.cache/pokedex/` on Linux, `~/Library/Caches/pokedex/` on macOS,
`%LocalAppData%\pokedex\` on Windows) and refreshed after 30 days by default
(`general.cache_ttl_days`). Delete that folder or use `--refresh` to force an update.

## Project layout

Standard Go layout: one command in `cmd/`, the code in `internal/` packages.

```
cmd/pokedex/          entry point (main.go just calls cli.Run)
configs/              config.example.toml: the commented default configuration
docs/                 configuration and key binding reference
internal/
  cli/                argument parsing, subcommands, `config` command, plain-text output
  config/             config file: defaults (default.toml), loading, validation
  tui/                interactive Bubble Tea interface
    model.go          root model, messages and data loading
    search.go         search screen: Pokédex, Moves, Items, Natures, Type Matchups tabs
    detail.go         Pokémon entry: Overview, Abilities and Moves tabs
    family.go         Family tab: evolutions, forms and gender looks
    nav.go            page history, breadcrumb and the lists of related Pokémon
    dexorder.go       Pokédex order and the type:/gen: filters
    overlay.go        help screen
    scroll.go         clipping and scrolling of panels that do not fit
    render.go         styles, type badges, stat bars, Pokédex card, nature chart
    help.go           help lines built from the configured key bindings
  dex/                PokeAPI client, local cache and domain logic
    client.go         HTTP client, request limiter, cache, type chart, Pokémon list
    pokemon.go        Pokémon, species, ability and move models and loaders
    evolution.go      evolution chains and readable evolution conditions
    natures.go        natures, stat search and the nature chart
    matchups.go       damage multipliers, grouping, ability notes, name formatting
    sprite.go         sprite download and cache
  fuzzy/              name normalisation and typo-tolerant suggestions
  sprite/             half-block pixel-art rendering of sprites
```

Dependencies only point one way: `cmd → cli → tui → dex / sprite / fuzzy`, with `config`
used by `cli` and `tui`.

## Development

```bash
make test   # run unit tests (go test ./...)
make vet    # static checks
make fmt    # format code
```

## Notes

- Sprites come from the [PokeAPI sprites](https://github.com/PokeAPI/sprites) repository
  and are cached in `sprites/` inside the cache directory. They look best in a terminal with
  24-bit colour (most modern terminals); on others the colours are approximated. Large
  sprites are scaled down, keeping the most common colour of each block so outlines stay sharp.
- Ability warnings are informational; the multipliers always reflect types alone.
- Generation-specific type charts and Tera types are not modelled.
- Evolutions that need a specific place (which differs per game) are shown as
  "level up at a special location".
- The Moves tab opens on the newest game with a level-up learnset (spin-offs such as
  Pokémon Champions are still available with `[` / `]`).

## License

[MIT](LICENSE). Pokémon and Pokémon names are trademarks of Nintendo, Game Freak and
Creatures Inc. This project is not affiliated with them. Data provided by
[PokeAPI](https://pokeapi.co).
