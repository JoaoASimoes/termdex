# Configuration

Everything that can be configured: the config file, command-line options and environment
variables. Key bindings have their own page: [keybindings.md](keybindings.md).

## The config file

Pokédex works without a config file. When one exists, it is read at startup, on top of the
built-in defaults: you only write the settings you want to change.

| Platform | Default location                                         |
|----------|----------------------------------------------------------|
| Linux    | `$XDG_CONFIG_HOME/pokedex/config.toml` (usually `~/.config/pokedex/config.toml`) |
| macOS    | `~/Library/Application Support/pokedex/config.toml`      |
| Windows  | `%AppData%\pokedex\config.toml`                          |

A different file can be used with `--config <file>` or the `POKEDEX_CONFIG` environment
variable (the flag wins).

The format is [TOML](https://toml.io). The easiest way to start is to write the default
file, which contains every setting with a comment, and edit it. The same file is in the
repository as [`configs/config.example.toml`](../configs/config.example.toml).

```bash
pokedex config init      # writes the defaults to the config location
$EDITOR "$(pokedex config path)"
pokedex config check     # validate your changes
```

### `pokedex config` commands

| Command                        | What it does                                                    |
|--------------------------------|-----------------------------------------------------------------|
| `pokedex config path`          | Print the config file location (and whether it exists).        |
| `pokedex config init`          | Write the commented default config there. Refuses to overwrite an existing file. |
| `pokedex config init --force`  | Same, overwriting the existing file.                            |
| `pokedex config show`          | Print the effective configuration (your file + defaults).      |
| `pokedex config check`         | Validate the file and report problems. Exit code 1 on error.   |

All of them respect `--config` and `POKEDEX_CONFIG`, e.g.
`pokedex --config ./my.toml config check`.

### Validation

The file is checked every time Pokédex starts. It refuses to start (exit code 2) and tells
you what is wrong when:

- the TOML is invalid;
- a setting is unknown (usually a typo, e.g. `[keys.serch]` or `donw = [...]`);
- a value is out of range (see the table below);
- the same key is bound to two actions that are active on the same screen (for example
  `ctrl+f` for both `keys.global.search` and `keys.search.down`).

To start with the defaults while you fix the file: `pokedex --config /dev/null`.

## `[general]` options

| Setting            | Type    | Default | Allowed   | Description |
|--------------------|---------|---------|-----------|-------------|
| `cache_ttl_days`   | integer | `30`    | 1 or more | Days before cached PokeAPI data (Pokémon, moves, abilities, natures, type chart…) is downloaded again. Sprites never expire. |
| `max_requests`     | integer | `6`     | 1–20      | Maximum number of requests sent to PokeAPI at the same time. Higher loads move lists faster the first time; please keep it low to be kind to PokeAPI, which is free. |
| `preview_delay_ms` | integer | `150`   | 0–5000    | Pause before loading a Pokémon you are scrolling past in the search list. Already-loaded Pokémon always show at once. Lower feels snappier, higher sends fewer requests. |
| `sprites`          | boolean | `true`  |           | Show pixel-art sprites in the TUI and in `pokedex pokemon`. Turn off for terminals without 24-bit colour, or to save space. |

Example:

```toml
[general]
cache_ttl_days = 7
max_requests = 4
preview_delay_ms = 80
sprites = false
```

## `[keys.*]` – key bindings

Every key in the TUI can be changed, per screen. See [keybindings.md](keybindings.md) for
the list of actions, their default keys, key names and examples.

```toml
[keys.search]
down = ["ctrl+d", "down"]
up = ["ctrl+u", "up"]
```

## Command-line options

Options go **before** the command (`pokedex --refresh pokemon mew`).

| Option             | Description                                                         |
|--------------------|---------------------------------------------------------------------|
| `--config <file>`  | Read the configuration from this file instead of the default location. |
| `--refresh`        | Ignore the cache for this run and download everything again (the new data is cached). |
| `-h`, `--help`     | Show the usage.                                                     |

## Environment variables

| Variable          | Effect                                                                 |
|-------------------|------------------------------------------------------------------------|
| `POKEDEX_CONFIG`  | Path of the config file (overridden by `--config`).                    |
| `XDG_CONFIG_HOME` | Base directory of the default config location on Linux.               |
| `XDG_CACHE_HOME`  | Base directory of the cache on Linux (`$XDG_CACHE_HOME/pokedex`).      |
| `NO_COLOR`        | Any value disables colours (and sprites) in the subcommands' output. The TUI keeps its colours. |
| `GOTOOLCHAIN`     | Only when building: leave it at `auto` so Go can fetch the 1.26 toolchain. |

## Cache

Not a setting, but useful to know: downloaded data lives in the user cache directory
(`~/.cache/pokedex/` on Linux, `~/Library/Caches/pokedex/` on macOS,
`%LocalAppData%\pokedex\` on Windows), with sprites in its `sprites/` folder. It is safe to
delete at any time; it is rebuilt on demand. Its lifetime is set by
`general.cache_ttl_days`, and `--refresh` bypasses it for one run.
