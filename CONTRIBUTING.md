# Contributing

Thanks for helping.

## Development setup

You need Go 1.25 or newer. On macOS, also install the Xcode command line tools for cgo. The linter is downloaded for you the first time you run `make lint`, at a pinned version and with its checksum checked.

```sh
make ci
```

`make ci` runs the same checks as the CI workflow. Run it before you open a pull request. The single steps are `make dep`, `make lint`, `make test`, and `make build`.

## Rules

- Write the test first, then the code.
- Use [Conventional Commits](https://www.conventionalcommits.org) for commit messages, for example `feat: add input switching`.
- Keep each pull request to one topic.
- JSON field names are camelCase.
- Add a comment only when the code cannot explain itself.

## Adding a monitor

Each supported monitor is one JSON file in `profiles/`, named after the model, for example `profiles/msi-mpg-274urdfw-e16m.json`.

1. Run `monctl probe` and `monctl list -json`. Note the manufacturer and the product code.
2. For each port, try values with `monctl input -raw <value>` until the monitor shows that port.
3. While the monitor shows that port, run `monctl probe` and note the input value.
4. Write the profile and run `make ci`.

Each input has two values. `write` is the value that switches to the port. `read` is the value the monitor reports while it shows the port. They can differ: on the MSI, DisplayPort is write 5 and read 7. Trust what you measure. The capabilities list a monitor reports can be wrong.

```json
{
  "name": "MSI MPG 274URDFW E16M",
  "match": {
    "manufacturer": "MSI",
    "productCode": "0x3cf0"
  },
  "inputs": [
    { "id": "dp", "label": "DisplayPort", "write": 5, "read": 7 },
    { "id": "usbc", "label": "USB-C", "write": 8, "read": 8 }
  ]
}
```

- `manufacturer` is the three letter maker id.
- `productCode` is `0x` and four lower case hex digits, as `monctl probe` prints it.
- Input `id` is lower case letters and digits, starting with a letter, and unique in the file.
- Every input needs a `label`.
- `write` and `read` are from 0 to 255.

`make ci` checks every shipped profile. `monctl probe` output has no serial number, so it is safe to post in an issue or pull request.

## Safety

On Linux, `monctl` writes to I2C buses. It must only ever open buses that belong to a graphics card. Never add code that opens SMBus or other system buses, because writing to them can damage hardware.

On macOS, `monctl` only opens display services marked external. Keep it that way.
