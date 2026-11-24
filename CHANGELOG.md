# Changelog

All notable changes to this project are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the project uses
[Semantic Versioning](https://semver.org/).

## [Unreleased]

## [0.1.0] - 2026-11-24

First public release.

### Added

- Interactive terminal interface (Bubble Tea) with five search tabs:
  - **Pokédex**: every Pokémon and form in National Pokédex order, typo-tolerant search,
    `type:` and `gen:` filters, and a live entry preview (sprite, description, stats,
    abilities, evolutions, weaknesses).
  - **Moves**: all 938 moves with their details and the Pokémon that learn them.
  - **Items**: all items with sprite, category, effect and the wild Pokémon that hold them.
  - **Natures**: the nature chart, berry flavours and search by stat.
  - **Type Matchups**: weaknesses, resistances and immunities of one or two types.
- Pokémon entries with Overview, Abilities, Moves (per game, with filter) and Family
  (evolutions, forms, gender differences) tabs.
- Links between pages: the Pokémon that learn a move, hold an item or have an ability;
  breadcrumb and back history.
- Pixel-art sprites drawn with half-block characters.
- Help screen (`?` or `F1`), scrolling for panels that do not fit, clear offline messages
  with retry.
- Config file (`~/.config/pokedex/config.toml`) for every key binding, the colours and
  general options; `pokedex config path|init|show|check`.
- Subcommands for quick lookups: `pokemon`, `type`, `types`, `version`.
- Local cache of all PokeAPI data; works offline for anything already seen.

[Unreleased]: https://github.com/JoaoASimoes/pokedex/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/JoaoASimoes/pokedex/releases/tag/v0.1.0
