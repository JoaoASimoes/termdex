# Pokédex documentation

- [Configuration](configuration.md) – the config file, `pokedex config` commands,
  general options, colours, command-line flags, environment variables and the cache.
- [Key bindings](keybindings.md) – every action of the TUI, its default keys, key names
  and how to rebind them.

The commented default configuration is
[`configs/config.example.toml`](../configs/config.example.toml): copy it to your config
location, or run `pokedex config init`, which writes exactly that file. (It is kept
identical to the defaults built into the binary, `internal/config/default.toml`; a test
checks this.)
