# Key bindings

Every key in the TUI is an *action* that you can rebind in the config file (see
[configuration.md](configuration.md) for where the file lives). The help line at the
bottom of the screen always shows the keys that are currently configured.

## How to change a key

Each action takes a **list** of keys. Setting an action **replaces** all of its default
keys, so list every key you want to keep. Actions you do not mention keep their defaults.

```toml
# ~/.config/pokedex/config.toml
[keys.search]
down = ["down", "ctrl+d"]     # ctrl+j / ctrl+n no longer move down in the search list
up = ["up", "ctrl+u"]

[keys.global]
type_matchups = ["ctrl+y"]
```

To disable an action, give it an empty list: `filter = []`.

Then run `pokedex config check`. It reports typos in action names and keys bound to two
actions on the same screen.

## Key names

Keys are written as Bubble Tea names them:

| Kind              | Names                                                                 |
|-------------------|-----------------------------------------------------------------------|
| Letters, digits, symbols | `a`, `G` (shift+g), `1`, `/`, `[`, `>` …                       |
| Special keys      | `enter`, `esc`, `tab`, `space`, `backspace`, `delete`, `insert`       |
| Arrows and paging | `up`, `down`, `left`, `right`, `pgup`, `pgdown`, `home`, `end`        |
| Function keys     | `f1` … `f12`                                                          |
| With modifiers    | `ctrl+` / `alt+` / `shift+` prefixes, combined in that order: `ctrl+k`, `alt+enter`, `shift+tab`, `ctrl+alt+x` |

Notes:

- An upper-case letter is written as itself (`G`), not `shift+g`.
- Some terminals cannot tell certain combinations apart: `ctrl+i` is `tab`,
  `ctrl+m` is `enter`, `ctrl+[` is `esc`, and on a few terminals `ctrl+h` is `backspace`.
  Prefer other combinations if a binding does not react.
- On the **search screen** you are typing in a text box, so do not bind plain letters,
  digits or symbols there: they could no longer be typed. Use `ctrl+`/`alt+` combinations
  or special keys.

## Actions and default keys

### `[keys.global]` – every screen

| Action          | Default     | What it does                                    |
|-----------------|-------------|-------------------------------------------------|
| `quit`          | `ctrl+c`    | Quit.                                           |
| `search`        | `ctrl+f`    | Open a new, empty Pokémon search.               |
| `type_matchups` | `ctrl+t`    | Go to the Type Matchups tab (Type 1 box).       |
| `help`          | `f1`, `?`   | Show / hide the help screen with every key of the current screen. `?` works while typing too: no Pokémon, move, item or nature has a `?` in its name. |
| `retry`         | `ctrl+r`    | Retry what failed after a network error.        |

### `[keys.search]` – search screen (Pokédex, Moves, Items, Natures and Type Matchups tabs)

| Action       | Default                        | What it does |
|--------------|--------------------------------|--------------|
| `up`         | `up`, `ctrl+k`, `ctrl+p`       | Previous suggestion (its result is shown at once). |
| `down`       | `down`, `ctrl+j`, `ctrl+n`     | Next suggestion. |
| `confirm`    | `enter`                        | Pokédex: open the full entry. Moves/Items: list the Pokémon that learn / hold it. Natures/Types: pick the highlighted entry. |
| `clear`      | `esc`                          | Clear the box; on an empty box, quit. |
| `next_field` | `tab`                          | Next box **in the same tab** (Type 1 ↔ Type 2). Does nothing in tabs with one box. |
| `prev_field` | `shift+tab`                    | Previous box in the same tab. |
| `next_tab`   | `ctrl+l`                       | Next tab (Pokédex → Moves → Items → Natures → Type Matchups → Pokédex). |
| `prev_tab`   | `ctrl+h`                       | Previous tab. |
| `scroll_up`  | `pgup`, `shift+up`             | Scroll the right panel up (when it does not fit the terminal). |
| `scroll_down`| `pgdown`, `shift+down`         | Scroll the right panel down. |

### `[keys.entry]` – Pokémon entry (Overview, Abilities, Moves, Family) and Pokémon lists

The same keys work in the lists of Pokémon opened from a move, an item or an ability.

| Action      | Default                          | What it does |
|-------------|----------------------------------|--------------|
| `up`        | `up`, `k`, `ctrl+k`, `ctrl+p`    | Previous ability / move / family member. |
| `down`      | `down`, `j`, `ctrl+j`, `ctrl+n`  | Next one. |
| `page_up`   | `pgup`                           | Up 10 rows. |
| `page_down` | `pgdown`                         | Down 10 rows. |
| `top`       | `home`, `g`                      | First row. |
| `bottom`    | `end`, `G`                       | Last row. |
| `next_tab`  | `ctrl+l`, `right`, `l`           | Next tab. |
| `prev_tab`  | `ctrl+h`, `left`, `h`            | Previous tab. |
| `overview`  | `1`                              | Overview tab. |
| `abilities` | `2`                              | Abilities tab. |
| `moves`     | `3`                              | Moves tab. |
| `family`    | `4`, `f`                         | Family tab. |
| `next_game` | `]`, `>`                         | Moves tab: next game. |
| `prev_game` | `[`, `<`                         | Moves tab: previous game. |
| `filter`    | `/`                              | Moves tab: start typing a filter. |
| `open`      | `enter`                          | Family: open the evolution or form. Moves: list the Pokémon that learn the move. Abilities: list the Pokémon with the ability. Lists: open the Pokémon. |
| `back`      | `esc`, `backspace`, `q`          | Back to the previous page, or to the search. |
| `scroll_up` | `shift+up`, `ctrl+u`             | Scroll the panels up (when they do not fit the terminal). |
| `scroll_down` | `shift+down`, `ctrl+d`         | Scroll the panels down. |

`tab` is not bound here on purpose: the entry screen has no boxes to move between, and tabs
are switched with `next_tab` / `prev_tab` or the number keys.

### `[keys.filter]` – while typing the move filter

| Action     | Default                    | What it does |
|------------|----------------------------|--------------|
| `up`       | `up`, `ctrl+k`, `ctrl+p`   | Previous move. |
| `down`     | `down`, `ctrl+j`, `ctrl+n` | Next move. |
| `apply`    | `enter`                    | Keep the filter and leave the box. |
| `cancel`   | `esc`                      | Clear the filter and leave the box. |
| `next_tab` | `ctrl+l`                   | Next tab. |
| `prev_tab` | `ctrl+h`                   | Previous tab. |

The global keys work here too. Every other key is typed into the filter.

## Examples

**Vim-style everywhere** (letters are fine in the entry screen, not in the search):

```toml
[keys.search]
up = ["up", "ctrl+k"]
down = ["down", "ctrl+j"]

[keys.entry]
up = ["k", "up"]
down = ["j", "down"]
prev_tab = ["h", "left"]
next_tab = ["l", "right"]
back = ["esc", "q"]
```

**Arrow keys only, tabs with alt**:

```toml
[keys.search]
up = ["up"]
down = ["down"]
next_tab = ["alt+right"]
prev_tab = ["alt+left"]

[keys.entry]
up = ["up"]
down = ["down"]
next_tab = ["right", "alt+right"]
prev_tab = ["left", "alt+left"]
```

**Function keys for the main places**:

```toml
[keys.global]
search = ["f2", "ctrl+f"]
type_matchups = ["f3", "ctrl+t"]
```
