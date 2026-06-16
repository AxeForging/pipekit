# Requirements

What pipekit needs to run, and what it expects from the surrounding environment.

## Contents

- [Supported platforms](#supported-platforms)
- [Runtime dependencies](#runtime-dependencies)
- [Build-time requirements](#build-time-requirements)
- [CI environment contract](#ci-environment-contract)
  - [GitHub Actions](#github-actions)
  - [GitLab CI](#gitlab-ci)
  - [Webhook URLs](#webhook-urls)
- [File format support](#file-format-support)

---

## Supported platforms

| OS | Architectures |
|---|---|
| Linux | amd64, arm64, 386, arm |
| macOS | amd64, arm64 (Apple Silicon) |
| Windows | amd64, arm64, 386 |

The binary is statically linked (`CGO_ENABLED=0`). No glibc version constraints, no shared libraries.

---

## Runtime dependencies

Most commands have **zero** runtime dependencies. The exceptions:

| Command | Needs | Why |
|---|---|---|
| `diff *` | `git` on `PATH` | Shells out to `git diff --name-only` |
| `version next` | `git` on `PATH` | Reads commit history for conventional-commits analysis |
| `wait command` | A POSIX-ish shell (`sh`) | Runs the user-supplied command via `sh -c` |
| `retry run -- ...` | The command being retried | Self-evident |

Everything else (env, mask, transform, summary, assert, matrix, notify, cache-key, config, parse) runs on the binary alone.

---

## Build-time requirements

Only relevant if you build from source.

| Tool | Version |
|---|---|
| Go | 1.25 or later |
| `make` | Any (used by the Makefile; not required for `go install` / `go build`) |
| `golangci-lint` | For `make lint` only |

See **[CONTRIBUTING.md](CONTRIBUTING.md)** for the full dev setup.

---

## CI environment contract

pipekit reads/writes the standard CI variables — you don't pass them, the runner does.

### GitHub Actions

| Variable | Read by | Written by |
|---|---|---|
| `GITHUB_ENV` | — | `env from-* --to-github`, `config resolve --to-github`, `parse extract-yaml --to-github`, `config branch-env --to-github` |
| `GITHUB_OUTPUT` | — | `--to-github-output` flag on env, matrix, diff, version, cache-key, config, parse |
| `GITHUB_STEP_SUMMARY` | — | `summary github`, `summary table`, `summary badge`, `summary section` |
| `GITHUB_REF` | `config branch-env` (auto-detect) | — |
| `GITHUB_HEAD_REF` | (you typically pipe into `transform slug`) | — |

If a `--to-github*` flag is used outside a GitHub Actions runner, the command exits non-zero with a clear `"$GITHUB_ENV is not set"` error.

### GitLab CI

`env from-* --to-gitlab` writes `export KEY="value"` lines to stdout, suitable for sourcing in subsequent script blocks. Standard GitLab variables (`CI_COMMIT_REF_NAME`, etc.) work as plain env vars and need no special handling.

### Webhook URLs

`notify` reads webhook URLs from flags or environment, in this order:

| Subcommand | Flag | Env var |
|---|---|---|
| `notify slack` | `--url` | `SLACK_WEBHOOK_URL` |
| `notify discord` | `--url` | `DISCORD_WEBHOOK_URL` |
| `notify teams` | `--url` | `TEAMS_WEBHOOK_URL` |
| `notify webhook` | `--url` (required) | — |

---

## File format support

| Format | Parser | Used by |
|---|---|---|
| JSON | `encoding/json` | `env from-json`, `config resolve`, `assert json-path`, `summary table`, `matrix from-json`, `parse extract-yaml`, `json *`, `yaml *`, `render --values` |
| YAML | `gopkg.in/yaml.v3` | `env from-yaml`, `config resolve`, `parse extract-yaml`, `parse extract-frontmatter`, `yaml *`, `json *` (when input is `.yaml`/`.yml`), `render --values`, `diff` config file (`.pipekit-diff.yaml`) |
| TOML | `pelletier/go-toml/v2` | `env from-toml`, `json convert --to toml`, `yaml *` (when input is `.toml`), `render --values`, `parse extract-frontmatter` (`+++` delimiter) |
| dotenv | built-in line parser | `env from-dotenv` |
| CSV | `encoding/csv` | `summary table`, `json convert --to csv` / `--from csv` |
| Markdown | regex-based fenced-block extraction | `parse extract-block`, `parse extract-yaml`, `parse extract-frontmatter` |
| Version files | per-format regex | `version get/bump/set`: `package.json`, `Cargo.toml`, `pyproject.toml`, `Chart.yaml`, `setup.py`, `build.gradle`, `pom.xml`, `VERSION`, `version.txt` |

---

**See also:** [Install](INSTALL.md) · [Commands](COMMANDS.md) · [Examples](EXAMPLES.md) · [← README](../README.md)
