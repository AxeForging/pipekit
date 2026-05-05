# Contributing

Working on pipekit itself. For an architectural deep-dive aimed at AI assistants see **[AI/README.md](AI/README.md)** — it has detailed file/service maps that won't be repeated here.

## Contents

- [Dev setup](#dev-setup)
- [Repo layout](#repo-layout)
- [Common workflows](#common-workflows)
- [Adding a new command](#adding-a-new-command)
- [Adding a flag](#adding-a-flag)
- [Testing](#testing)
- [Linting](#linting)
- [Releasing](#releasing)

---

## Dev setup

| Tool | Version |
|---|---|
| Go | 1.24+ |
| `make` | any |
| `golangci-lint` | for `make lint` |
| `git` | for runtime tests of `diff` / `version next` |

```sh
git clone https://github.com/AxeForging/pipekit.git
cd pipekit
make build           # → dist/pipekit
make test            # run all unit tests
make ci              # tidy → lint → test → build
```

---

## Repo layout

```
pipekit/
├── main.go                # CLI app setup, command registration, version vars
├── actions/               # CLI handlers (one file per command group)
├── services/              # Business logic, no CLI dependency
│   └── *_service_test.go  # Unit tests (table-driven)
├── domain/types.go        # Shared types (KeyValue, NotifyMessage, DiffConfig)
├── helpers/               # logger, sentinel errors
├── docs/                  # User-facing docs (this file lives here)
│   └── AI/                # Reference for AI assistants
├── integration/           # End-to-end tests (currently empty — contributions welcome)
├── Makefile               # build, test, lint, clean, ci, build-all
├── .goreleaser.yml        # Multi-platform release config
└── .github/workflows/     # release.yaml on tag push
```

**Architectural rule:** `actions/*.go` handles CLI concerns (flag parsing, stdin, exit codes). `services/*.go` is pure logic that takes `io.Reader` / `io.Writer` and returns errors. Services must be testable without invoking the CLI.

The full file→function map lives in **[AI/README.md](AI/README.md)**.

---

## Common workflows

```sh
# Run a single test
go test ./services -run TestParseJSON_Flatten -v

# Build and quickly try the binary
make build && dist/pipekit assert semver 1.2.3

# Coverage
go test ./... -cover -coverprofile=coverage.out
go tool cover -func=coverage.out
go tool cover -html=coverage.out      # opens browser

# Build all platforms
make build-all                        # → dist/pipekit-{os}-{arch}

# Update deps
go get -u ./... && go mod tidy && go test ./...
```

---

## Adding a new command

1. **Service** — `services/newcmd_service.go`: pure functions accepting `io.Reader` / `io.Writer` / strings, returning `(result, error)`.
2. **Test** — `services/newcmd_service_test.go`: table-driven tests for the functions above.
3. **Action** — `actions/newcmd.go`: a `func NewCmdCommand() cli.Command` that wires flags, reads input, calls the service, formats output, and returns `cli.NewExitError` on failure.
4. **Register** — add `actions.NewCmdCommand()` to `app.Commands` in `main.go`.
5. **Docs** — add a row to the commands table in `README.md`, an entry under [`COMMANDS.md`](COMMANDS.md), and a recipe in [`EXAMPLES.md`](EXAMPLES.md) if it replaces a common bash idiom.

For a subcommand on an existing group, skip steps 1–4 minus the relevant additions: extend the existing service file and add a `cli.Command{}` to the group's `Subcommands` slice.

---

## Adding a flag

In `actions/<group>.go`:

```go
Flags: []cli.Flag{
    cli.StringFlag{Name: "my-flag", Usage: "what it does"},
    cli.BoolFlag{Name: "my-toggle"},
}
```

Read it in the action:

```go
val := c.String("my-flag")
on  := c.Bool("my-toggle")
```

Pass it to the service. Don't reach into `os.Args` from the service layer.

---

## Testing

- Unit tests live next to the service they cover (`services/*_service_test.go`).
- Prefer table-driven tests; existing files are good templates.
- Tests should not depend on network, the user's git config, or the host's `$GITHUB_ENV`. When testing GitHub-output writers, set the env var to a temp file.
- Integration tests (driving the built binary) belong in `integration/`.

---

## Linting

```sh
make lint    # runs golangci-lint with a 5m timeout
```

Install: https://golangci-lint.run/usage/install/.

---

## Releasing

Releases are tag-driven. Pushing a tag matching `v*` triggers `.github/workflows/release.yaml`, which runs `go test ./...` then GoReleaser to build and publish multi-platform binaries to GitHub Releases.

```sh
# Tag (Makefile target sanity-checks the VERSION)
VERSION=v1.2.3 make tag
git push origin v1.2.3
```

`make release-check` runs the test suite locally and reports the version that would be released — useful before tagging.

---

**See also:** [Commands](COMMANDS.md) · [Examples](EXAMPLES.md) · [AI architecture reference](AI/README.md) · [← README](../README.md)
