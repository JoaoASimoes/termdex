# Security policy

## Supported versions

Security fixes are made on the latest release only. Please update before reporting.

## Reporting a vulnerability

Please **do not open a public issue** for security problems. Instead, report them privately
through GitHub: open the repository's **Security** tab and click
**Report a vulnerability** ([direct link](https://github.com/JoaoASimoes/termdex/security/advisories/new)).

Include what you found, how to reproduce it and the version (`termdex --version`). You
should get an answer within a week. Once a fix is released, the report can be made public
and you will be credited if you wish.

## Scope

termdex is a local terminal program. It downloads data from `pokeapi.co` and sprites from
`raw.githubusercontent.com` over HTTPS, stores them in your user cache directory, and reads a
TOML config file. Problems in any of these areas (for example a crafted response or config
file causing a crash or writing outside the cache directory) are in scope.
