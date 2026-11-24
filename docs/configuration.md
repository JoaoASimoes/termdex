# Configuration

Everything that can be configured: the config file, command-line options and environment
variables. Key bindings have their own page: [keybindings.md](keybindings.md).

## The config file

termdex works without a config file. When one exists, it is read at startup, on top of the
built-in defaults: you only write the settings you want to change.

| Platform | Default location                                         |
|----------|----------------------------------------------------------|
| Linux    | `$XDG_CONFIG_HOME/termdex/config.toml` (usually `~/.config/termdex/config.toml`) |
| macOS    | `~/Library/Application Support/termdex/config.toml`      |
| Windows  | `%AppData%\termdex\config.toml`                          |

A different file can be used with `--config <file>` or the `TERMDEX_CONFIG` environment
variable (the flag wins).

The format is [TOML](https://toml.io). The easiest way to start is to write the default
file, which contains every setting with a comment, and edit it. The same file is in the
repository as [`configs/config.example.toml`](../configs/config.example.toml).

```bash
termdex config init      # writes the defaults to the config location
$EDITOR "$(termdex config path)"
termdex config check     # validate your changes
```

### `termdex config` commands

| Command                        | What it does                                                    |
|--------------------------------|-----------------------------------------------------------------|
| `termdex config path`          | Print the config file location (and whether it exists).        |
| `termdex config init`          | Write the commented default config there. Refuses to overwrite an existing file. |
| `termdex config init --force`  | Same, overwriting the existing file.                            |
| `termdex config show`          | Print the effective configuration (your file + defaults).      |
| `termdex config check`         | Validate the file and report problems. Exit code 1 on error.   |

All of them respect `--config` and `TERMDEX_CONFIG`, e.g.
`termdex --config ./my.toml config check`.

### Validation

The file is checked every time termdex starts. It refuses to start (exit code 2) and tells
you what is wrong when:

- the TOML is invalid;
- a setting is unknown (usually a typo, e.g. `[keys.serch]` or `donw = [...]`);
- a value is out of range (see the table below), or a colour is not `#RRGGBB`/`0`–`255`;
- the same key is bound to two actions that are active on the same screen (for example
  `ctrl+f` for both `keys.global.search` and `keys.search.down`).

To start with the defaults while you fix the file: `termdex --config /dev/null`.

## `[general]` options

| Setting            | Type    | Default | Allowed   | Description |
|--------------------|---------|---------|-----------|-------------|
| `cache_ttl_days`   | integer | `30`    | 1 or more | Days before cached PokeAPI data (Pokémon, moves, abilities, natures, type chart…) is downloaded again. Sprites never expire. |
| `max_requests`     | integer | `6`     | 1–20      | Maximum number of requests sent to PokeAPI at the same time. Higher loads move lists faster the first time; please keep it low to be kind to PokeAPI, which is free. |
| `preview_delay_ms` | integer | `150`   | 0–5000    | Pause before loading a Pokémon you are scrolling past in the search list. Already-loaded Pokémon always show at once. Lower feels snappier, higher sends fewer requests. |
| `sprites`          | boolean | `true`  |           | Show pixel-art sprites in the TUI and in `termdex pokemon`. Turn off for terminals without 24-bit colour, or to save space. |

Example:

```toml
[general]
cache_ttl_days = 7
max_requests = 4
preview_delay_ms = 80
sprites = false
```

## `[colors]` – colours of the interface

Every colour of the TUI can be changed. Values are either `"#RRGGBB"` (24-bit colour) or an
ANSI colour number `"0"`–`"255"` (useful for terminals without 24-bit colour, or to follow
your terminal's palette: `"1"` is its red, `"4"` its blue…).

| Setting       | Default   | Used for |
|---------------|-----------|----------|
| `accent`      | `#E3350D` | Title bar, active tab, selected row, focused panel border, the back-key badge, the selected nature in the chart, the loading spinner. |
| `accent_text` | `#FFFFFF` | Text drawn on the accent colour (title bar, active tab, back-key badge). |
| `heading`     | `#FFCB05` | Section titles ("Base stats", "Evolution", "In short"…). |
| `muted`       | `#6C6C6C` | Labels, help line, inactive tabs, panel borders. |
| `error`       | `#FF5F5F` | Error messages. |
| `warning`     | `#FFD75F` | Warnings: abilities that change matchups, Legendary/Mythical tags. |
| `flavor`      | `#A8A8A8` | Descriptions of Pokémon, moves and items (in italics). |

### `[colors.types]` – type badges

The background of each type badge: `normal`, `fire`, `water`, `electric`, `grass`, `ice`,
`fighting`, `poison`, `ground`, `flying`, `psychic`, `bug`, `rock`, `ghost`, `dragon`,
`dark`, `steel`, `fairy`. Types you do not set keep their default (the classic colours from
the games, listed in the default config). With `"#RRGGBB"` values the badge text is made
dark or light automatically for contrast; with ANSI numbers the default choice is kept.

```toml
# A blue theme that follows the terminal palette for the basics
[colors]
accent = "#1E66F5"
heading = "#40A0FF"
muted = "8"          # the terminal's "bright black"

[colors.types]
fire = "#FF6B35"
```

Not configurable: the multiplier colours (4×, 2×, ½×…), the stat bar gradient, the
physical/special/status colours and sprite colours. The subcommands' plain-text output
(`termdex pokemon …`) uses the terminal's 256-colour palette and is not affected by
`[colors]`; turn colours off there with `NO_COLOR=1`.

## `[keys.*]` – key bindings

Every key in the TUI can be changed, per screen. See [keybindings.md](keybindings.md) for
the list of actions, their default keys, key names and examples.

```toml
[keys.search]
down = ["ctrl+d", "down"]
up = ["ctrl+u", "up"]
```

## Command-line options

Options go **before** the command (`termdex --refresh pokemon mew`).

| Option             | Description                                                         |
|--------------------|---------------------------------------------------------------------|
| `--config <file>`  | Read the configuration from this file instead of the default location. |
| `--refresh`        | Ignore the cache for this run and download everything again (the new data is cached). |
| `-h`, `--help`     | Show the usage.                                                     |

## Environment variables

| Variable          | Effect                                                                 |
|-------------------|------------------------------------------------------------------------|
| `TERMDEX_CONFIG`  | Path of the config file (overridden by `--config`).                    |
| `XDG_CONFIG_HOME` | Base directory of the default config location on Linux.               |
| `XDG_CACHE_HOME`  | Base directory of the cache on Linux (`$XDG_CACHE_HOME/termdex`).      |
| `NO_COLOR`        | Any value disables colours (and sprites) in the subcommands' output. The TUI keeps its colours. |
| `GOTOOLCHAIN`     | Only when building: leave it at `auto` so Go can fetch the 1.26 toolchain. |

## Cache

Not a setting, but useful to know: downloaded data lives in the user cache directory
(`~/.cache/termdex/` on Linux, `~/Library/Caches/termdex/` on macOS,
`%LocalAppData%\termdex\` on Windows), with sprites in its `sprites/` folder. It is safe to
delete at any time; it is rebuilt on demand. Its lifetime is set by
`general.cache_ttl_days`, and `--refresh` bypasses it for one run.
