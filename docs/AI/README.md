# pipekit - AI Assistant Reference

This document provides structured context for AI assistants working with the pipekit codebase.

## Quick Reference

| Item | Value |
|------|-------|
| **Purpose** | CLI tool replacing bash one-liners in CI/CD pipelines |
| **Language** | Go 1.25+ |
| **CLI Framework** | `urfave/cli` v1 |
| **Repository** | https://github.com/AxeForging/pipekit |
| **License** | MIT |
| **Commands** | 14 command groups, 45+ subcommands |

---

## Architecture

```
pipekit/
├── main.go                          # Entry point, CLI app setup, version vars
├── go.mod                           # github.com/AxeForging/pipekit
├── Makefile                         # build, test, lint, clean, ci
├── .goreleaser.yml                  # Multi-platform release config
├── .github/workflows/release.yaml   # GitHub Actions release on tag push
├── actions/                         # CLI command handlers (one file per command group)
│   ├── env.go                       # env from-json, from-yaml, from-dotenv, to-*
│   ├── mask.go                      # mask values, file, github, env
│   ├── transform.go                 # transform base64, url, case, regex, template, hash, slug
│   ├── summary.go                   # summary github, table, badge, section
│   ├── assert.go                    # assert env-exists, file-exists, json-path, semver, compare, url
│   ├── matrix.go                    # matrix from-dirs, from-files, from-json, combine
│   ├── notify.go                    # notify slack, discord, teams, webhook
│   ├── wait.go                      # wait url, tcp, command
│   ├── diff.go                      # diff files, dirs, match, affected
│   ├── version.go                   # version get, bump, compare, next
│   ├── retry.go                     # retry run
│   ├── cache_key.go                 # cache-key from-files, from-glob, composite
│   ├── config.go                    # config resolve, branch-env
│   └── parse.go                     # parse extract-block, extract-yaml
├── services/                        # Business logic (one file per domain)
│   ├── env_service.go               # JSON/YAML/dotenv parsing, key transforms, GitHub/shell output
│   ├── mask_service.go              # Pattern-based masking, GitHub ::add-mask::, env var matching
│   ├── transform_service.go         # Base64, URL encoding, case conversion, regex, templates, hashing, slug
│   ├── summary_service.go           # Markdown tables, badges, collapsible sections, GITHUB_STEP_SUMMARY
│   ├── assert_service.go            # Env/file existence, JSON path, semver validation, URL checks
│   ├── matrix_service.go            # Dir/file/JSON matrix generation, Cartesian product
│   ├── notify_service.go            # Slack/Discord/Teams webhook payloads, generic POST
│   ├── wait_service.go              # URL/TCP/command polling with backoff
│   ├── diff_service.go              # Git diff, path filtering, service mapping
│   ├── version_service.go           # Version extraction, bumping, comparison, conventional commits
│   ├── retry_service.go             # Command execution with retries and backoff
│   ├── cache_key_service.go         # SHA256 file hashing, glob matching, composite keys
│   ├── config_service.go            # Env name normalization, config map resolution, branch-to-env mapping
│   ├── parse_service.go             # Fenced code block extraction, YAML parsing from markdown
│   └── *_service_test.go            # Unit tests for each service
├── domain/
│   └── types.go                     # Shared types: KeyValue, NotifyMessage, DiffConfig, WaitResult
└── helpers/
    ├── logger.go                    # zerolog console logger setup
    └── errors.go                    # Sentinel errors
```

---

## Command Flow

```
User Command → main.go (urfave/cli dispatch)
                 ↓
              actions/*.go (parse flags, get input reader)
                 ↓
              services/*.go (business logic, no CLI dependency)
                 ↓
              Output: stdout, GITHUB_ENV, GITHUB_OUTPUT, GITHUB_STEP_SUMMARY, or webhook
```

Key design principle: **actions handle CLI concerns** (flags, stdin, exit codes), **services are pure logic** (accept io.Reader/io.Writer, return errors). This makes services independently testable.

---

## Key Services

| Service | Core Functions | Notes |
|---------|---------------|-------|
| `env_service.go` | `ParseJSON()`, `ParseYAML()`, `ParseDotenv()`, `TransformKeys()`, `WriteToShell()`, `WriteToGitHubEnv()` | Uses `gojq` for filter expressions, handles multiline GitHub env syntax |
| `mask_service.go` | `MaskValues()`, `MaskFile()`, `MaskGitHub()`, `MaskEnvVars()` | Compiled regex patterns, partial masking support |
| `transform_service.go` | `Base64Encode/Decode()`, `URLEncode/Decode()`, `ConvertCase()`, `RegexReplace()`, `RenderTemplate()`, `HashData()`, `Slugify()` | Word splitting handles camelCase boundaries, slugify strips refs/heads/ |
| `summary_service.go` | `AppendToGitHubSummary()`, `GenerateTable()`, `GenerateBadge()`, `GenerateSection()` | Reads `$GITHUB_STEP_SUMMARY` path from env |
| `assert_service.go` | `AssertEnvExists()`, `AssertFileExists()`, `AssertJSONPath()`, `AssertSemver()`, `AssertSemverCompare()`, `AssertURL()` | Uses `gojq` for JSON path, `Masterminds/semver` for version validation |
| `matrix_service.go` | `MatrixFromDirs()`, `MatrixFromFiles()`, `MatrixFromJSON()`, `MatrixCombine()` | Output is GitHub Actions `fromJSON()`-compatible |
| `notify_service.go` | `SendSlack()`, `SendDiscord()`, `SendTeams()`, `SendWebhook()` | Platform-specific payload formatting (Slack blocks, Discord embeds, Teams Adaptive Cards) |
| `wait_service.go` | `WaitForURL()`, `WaitForTCP()`, `WaitForCommand()` | Context-based timeout, exponential backoff |
| `diff_service.go` | `DiffFiles()`, `DiffDirs()`, `DiffMatch()`, `DiffAffected()` | Shells out to `git diff`, falls back from three-dot to two-dot |
| `version_service.go` | `VersionGet()`, `VersionBump()`, `VersionCompare()`, `VersionNext()` | Auto-detects package.json, Cargo.toml, pyproject.toml, Chart.yaml, VERSION, etc. |
| `retry_service.go` | `RetryRun()` | Configurable exit code filtering |
| `cache_key_service.go` | `CacheKeyFromFiles()`, `CacheKeyFromGlob()`, `CacheKeyComposite()` | SHA256-based, deterministic |
| `config_service.go` | `NormalizeEnvName()`, `ResolveConfig()`, `ResolveConfigJSON()`, `BranchToEnv()` | Built-in env aliases, custom alias support, glob branch matching |
| `parse_service.go` | `ExtractCodeBlocks()`, `ExtractAndParseYAML()`, `FormatCodeBlocksJSON()`, `FormatParsedYAMLJSON()` | Supports ``` and ~~~ fences, language filtering, case-insensitive |

---

## Shared Types (domain/types.go)

```go
type KeyValue struct {
    Key   string
    Value string
}

type NotifyMessage struct {
    Status  string            // success, failure, warning
    Title   string
    Message string
    Fields  map[string]string
    URL     string
}

type DiffConfig struct {
    Services map[string][]string `yaml:"services"`
}

type WaitResult struct {
    Success    bool
    Attempts   int
    StatusCode int
    Body       string
}
```

---

## Dependencies

| Package | Purpose |
|---------|---------|
| `github.com/urfave/cli` v1.22 | CLI framework |
| `github.com/itchyny/gojq` | jq-style JSON filtering (env, assert commands) |
| `github.com/Masterminds/semver/v3` | Semver parsing and comparison (assert, version commands) |
| `github.com/rs/zerolog` | Structured logging |
| `gopkg.in/yaml.v3` | YAML parsing |

---

## Testing

**Approach**: Unit tests on the services layer.

```sh
# Run all tests
go test ./... -v

# Run tests for a specific service
go test ./services -run TestParseJSON -v

# With coverage
go test ./... -cover -coverprofile=coverage.out
go tool cover -func=coverage.out
```

### Test Files

| File | What it tests |
|------|---------------|
| `services/env_service_test.go` | JSON/YAML/dotenv parsing, key transforms, shell output |
| `services/mask_service_test.go` | Pattern masking, partial masking, GitHub mask format |
| `services/transform_service_test.go` | Base64, URL encoding, case conversion, regex, templates, hashing |
| `services/summary_service_test.go` | Table generation (JSON/CSV), badges, collapsible sections |
| `services/assert_service_test.go` | Env/file existence, JSON path assertions, semver validation and comparison |
| `services/matrix_service_test.go` | Dir/file matrix, JSON filtering, Cartesian product |
| `services/version_service_test.go` | Version extraction from multiple file formats, bump, compare |
| `services/cache_key_service_test.go` | File hashing, glob matching, composite keys, determinism |
| `services/config_service_test.go` | Env name normalization, config resolution (JSON/YAML), branch-to-env mapping |
| `services/parse_service_test.go` | Code block extraction, language filtering, YAML parsing from markdown |

---

## Build System

```sh
# Build for current platform
make build              # Output: dist/pipekit

# Build all platforms
make build-all          # Output: dist/pipekit-{os}-{arch}

# Full CI pipeline
make ci                 # tidy → lint → test → build

# Clean
make clean
```

### Version Injection

```sh
go build -ldflags="-s -w \
  -X main.Version=$(git describe --tags) \
  -X main.BuildTime=$(date -u '+%Y-%m-%d_%H:%M:%S') \
  -X main.GitCommit=$(git rev-parse --short HEAD)" -o pipekit .
```

---

## Common Tasks for AI

### Adding a New Command Group

1. Create `services/newcmd_service.go` with business logic functions
2. Create `services/newcmd_service_test.go` with unit tests
3. Create `actions/newcmd.go` with `func NewCmdCommand() cli.Command`
4. Register in `main.go`: add `actions.NewCmdCommand()` to `app.Commands`

### Adding a Subcommand to Existing Group

1. Add business logic function in the corresponding `services/*_service.go`
2. Add test for the new function
3. Add `cli.Command{}` entry in the `Subcommands` slice in the corresponding `actions/*.go`

### Adding a New Flag to a Subcommand

1. Add `cli.StringFlag{}` / `cli.BoolFlag{}` / etc. to the command's `Flags` slice in `actions/*.go`
2. Access it via `c.String("flag-name")` / `c.Bool("flag-name")` in the `Action` function
3. Pass value to the service function

### Key Patterns

**Input reading** - most commands accept a file arg or stdin:
```go
// Defined in actions/env.go, reused across actions
func getInputReader(c *cli.Context) (*os.File, error)

// Defined in actions/transform.go
func readValueOrStdin(c *cli.Context) ([]byte, error)
```

**GitHub output** - shared helper in matrix_service.go:
```go
func WriteToGitHubOutputValue(key, value string) error
```

**Exit codes** - use `cli.NewExitError()` for non-zero exits:
```go
return cli.NewExitError(err.Error(), 1)
```

### Debugging

```sh
# Run single test
go test ./services -run TestParseJSON_Flatten -v

# Quick binary test
echo '{"foo": "bar"}' | go run . env from-json --uppercase-keys

# Build and test
make build && dist/pipekit assert semver 1.2.3
```

### Update Dependencies

```sh
go get -u ./...
go mod tidy
go test ./...
```

---

## Links

- **Repository**: https://github.com/AxeForging/pipekit
- **Releases**: https://github.com/AxeForging/pipekit/releases
- **Issues**: https://github.com/AxeForging/pipekit/issues
- **Organization**: https://github.com/AxeForging
