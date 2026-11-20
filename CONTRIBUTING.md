# Contributing

Thanks for helping.

## Development setup

You need Go 1.25 or newer and [golangci-lint](https://golangci-lint.run) v2.

```sh
make test
make lint
make build
```

## Rules

- Write the test first, then the code.
- Use [Conventional Commits](https://www.conventionalcommits.org) for commit messages, for example `feat: add input switching`.
- Keep each pull request to one topic.
- JSON field names are camelCase.
- Add a comment only when the code cannot explain itself.

## Adding a monitor

Each supported monitor is one profile file. The profile format and the `monctl probe` command that collects the needed data are not built yet. This section will be filled in when they are.

Until then, open an issue with your monitor's maker and model.

## Safety

`monctl` writes to I2C buses. It must only ever open buses that belong to a graphics card. Never add code that opens SMBus or other system buses, because writing to them can damage hardware.
