# Contributing

Thanks for your interest in termdex! Bug reports, ideas, documentation fixes and code are
all welcome.

## Ways to help

- **Report a bug** or **suggest a feature** with the
  [issue templates](https://github.com/JoaoASimoes/termdex/issues/new/choose).
- **Fix an issue**: look for the `good first issue` and `help wanted` labels, and leave a
  comment so nobody works on the same thing twice.
- **Improve the docs**: anything unclear in the README or in [`docs/`](docs/) is a bug too.

Wrong game data (a stat, a move, an evolution) usually comes from
[PokeAPI](https://github.com/PokeAPI/pokeapi/issues); please check there first.

## Development setup

You need [Go](https://go.dev/dl/) 1.26 or newer (with an older Go, `GOTOOLCHAIN=auto`, the
default, downloads the right version) and `make`.

```bash
git clone https://github.com/JoaoASimoes/termdex.git
cd termdex
make build      # builds ./termdex
./termdex       # run it
```

Useful targets:

| Command          | What it does                                                   |
|------------------|----------------------------------------------------------------|
| `make check`     | Everything CI checks: formatting, `go vet`, tests with `-race` |
| `make test`      | Tests only                                                     |
| `make fmt`       | Format the code with `gofmt`                                   |
| `make cover`     | Coverage report in `coverage.html`                             |
| `make install`   | Install into `$(go env GOPATH)/bin`                            |
| `make dist`      | Cross-compile for every platform into `dist/`                  |

Use a separate config file while developing so your own settings stay untouched:
`./termdex --config /dev/null` starts with the defaults.

## Project layout

```
cmd/termdex/     entry point
internal/cli/    flags, subcommands, `config` command
internal/config/ config file, defaults (default.toml), validation, help texts
internal/tui/    Bubble Tea interface: screens, navigation, rendering
internal/dex/    PokeAPI client, cache and domain logic (no UI code)
internal/fuzzy/  typo-tolerant matching
internal/sprite/ pixel-art rendering of sprites
docs/            user documentation
```

Dependencies point one way: `cli → tui → dex / sprite / fuzzy`, with `config` used by `cli`
and `tui`. Keep PokeAPI and game logic in `dex`, and anything visual in `tui`.

## Guidelines

- **Format and vet**: `make check` must pass. CI runs it on Linux, macOS and Windows.
- **Tests**: add or update tests for logic changes (`internal/dex`, `internal/fuzzy`,
  `internal/config`, and the TUI model's behaviour in `internal/tui/*_test.go`). Tests must
  not use the network: use `dex.NewOfflineClient` / `dex.NewOfflineClientIDs`.
- **Try it in the terminal**: for UI changes, check a wide and a narrow terminal
  (e.g. 130×45 and 80×24) and mention what you checked in the pull request.
- **Be kind to PokeAPI**: it is free and run by volunteers. Cache everything you download
  (see `cached` in `internal/dex/client.go`), go through the client's request limiter, and
  avoid downloading data the user did not ask for.
- **New settings or keys** go in `internal/config/default.toml` *and* are copied to
  `configs/config.example.toml` (a test checks they are identical), with a description in
  `actionDesc` (`internal/config/config.go`) and in `docs/`.
- **Changelog**: add a line under "Unreleased" in [`CHANGELOG.md`](CHANGELOG.md) for
  user-visible changes.
- **Commits**: small and focused, with a message that says what changes and why. One topic
  per pull request makes review much faster.

## Pull requests

1. Fork the repository and create a branch from `main`.
2. Make your change, with tests and docs.
3. Run `make check`.
4. Open a pull request and fill in the template.

By contributing you agree that your contribution is licensed under the [MIT License](LICENSE).

## Releases (maintainers)

1. Move the "Unreleased" entries in `CHANGELOG.md` under a new version heading.
2. Tag and push: `git tag v0.2.0 && git push origin v0.2.0`.
3. The [release workflow](.github/workflows/release.yml) builds the binaries with
   GoReleaser and creates a **draft** release; review it and publish it.
