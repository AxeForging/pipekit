# pipekit

<div align="center">
  <p>
    <img src="https://img.shields.io/badge/Go-1.24%2B-00ADD8?style=flat-square&logo=go" alt="Go Version">
    <img src="https://img.shields.io/badge/OS-Linux%20%7C%20macOS%20%7C%20Windows-darkblue?style=flat-square&logo=windows" alt="OS Support">
    <img src="https://img.shields.io/badge/License-MIT-green?style=flat-square" alt="License">
  </p>
</div>

**CI/CD pipeline Swiss Army knife** - replace ugly bash one-liners with a single, portable Go binary.

## TL;DR

```sh
# Install
curl -L https://github.com/AxeForging/pipekit/releases/latest/download/pipekit-linux-amd64.tar.gz | tar xz
sudo mv pipekit-linux-amd64 /usr/local/bin/pipekit

# Extract JSON config into GITHUB_ENV as UPPER_SNAKE_CASE vars
pipekit env from-json config.json --uppercase-keys --to-github

# Assert required env vars exist before deploying
pipekit assert env-exists DEPLOY_TOKEN CLUSTER_NAME IMAGE_TAG

# Wait for a service to be ready
pipekit wait url http://localhost:8080/healthz --timeout 150s --interval 5s

# Retry a flaky command
pipekit retry run --attempts 3 --delay 30s -- helm upgrade --install myapp ./chart
```

## What it replaces

```bash
# BEFORE (fragile bash)
for key in $(jq -r 'keys[]' config.json); do
  value=$(jq -r ".[\"$key\"]" config.json)
  echo "${key^^}=$value" >> "$GITHUB_ENV"
done

# AFTER
pipekit env from-json config.json --uppercase-keys --to-github
```

## Documentation

| Audience | Link |
|----------|------|
| **Users** | You're looking at it |
| **AI Assistants** | [docs/AI/README.md](docs/AI/README.md) - Architecture, services, common tasks |

---

## Install

<details>
<summary><strong>Linux/macOS (AMD64)</strong></summary>

```sh
curl -L https://github.com/AxeForging/pipekit/releases/latest/download/pipekit-linux-amd64.tar.gz | tar xz
chmod +x pipekit-linux-amd64
sudo mv pipekit-linux-amd64 /usr/local/bin/pipekit
```

</details>

<details>
<summary><strong>Linux/macOS (ARM64 / Apple Silicon)</strong></summary>

```sh
# Linux ARM64
curl -L https://github.com/AxeForging/pipekit/releases/latest/download/pipekit-linux-arm64.tar.gz | tar xz
chmod +x pipekit-linux-arm64
sudo mv pipekit-linux-arm64 /usr/local/bin/pipekit

# macOS Apple Silicon
curl -L https://github.com/AxeForging/pipekit/releases/latest/download/pipekit-darwin-arm64.tar.gz | tar xz
chmod +x pipekit-darwin-arm64
sudo mv pipekit-darwin-arm64 /usr/local/bin/pipekit
```

</details>

<details>
<summary><strong>Windows (PowerShell)</strong></summary>

```powershell
Invoke-WebRequest -Uri https://github.com/AxeForging/pipekit/releases/latest/download/pipekit-windows-amd64.zip -OutFile pipekit.zip
Expand-Archive -Path pipekit.zip -DestinationPath .
Move-Item -Path pipekit-windows-amd64.exe -Destination pipekit.exe
```

</details>

<details>
<summary><strong>From Source (Go 1.24+)</strong></summary>

```sh
go install github.com/AxeForging/pipekit@latest
```

Or build locally:

```sh
git clone https://github.com/AxeForging/pipekit.git
cd pipekit
make build
sudo mv dist/pipekit /usr/local/bin/
```

</details>

---

## Commands

### `env` - Environment Variable Injection

Extract data from structured files and inject as env vars.

<details>
<summary><strong>from-json</strong> - Parse JSON into env vars</summary>

```sh
# Flat JSON to shell exports
pipekit env from-json config.json

# Nested JSON, flatten and uppercase, write to GITHUB_ENV
pipekit env from-json config.json --flatten --uppercase-keys --to-github

# With prefix
pipekit env from-json config.json --uppercase-keys --prefix "APP_" --to-github

# From stdin
cat config.json | pipekit env from-json --to-github-output

# With jq-style filter
pipekit env from-json config.json --filter "{name, version}" --to-github
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--uppercase-keys, -u` | Convert keys to UPPER_SNAKE_CASE |
| `--prefix` | Add prefix to all keys |
| `--flatten, -f` | Flatten nested objects with `_` separator |
| `--depth` | Max flattening depth |
| `--filter` | jq-style filter expression |
| `--strip-quotes` | Remove surrounding quotes from values |
| `--to-github` | Write to `$GITHUB_ENV` |
| `--to-github-output` | Write to `$GITHUB_OUTPUT` |
| `--to-gitlab` | Write export statements for GitLab CI |

</details>

<details>
<summary><strong>from-yaml</strong> - Parse YAML into env vars</summary>

```sh
pipekit env from-yaml config.yaml --flatten --uppercase-keys --to-github
```

Same flags as `from-json`.

</details>

<details>
<summary><strong>from-dotenv</strong> - Parse .env files</summary>

```sh
pipekit env from-dotenv .env --to-github
pipekit env from-dotenv .env.production --prefix "PROD_" --to-github
```

</details>

<details>
<summary><strong>to-shell / to-github / to-gitlab</strong> - Output targets</summary>

```sh
# Source-able shell exports
pipekit env from-json config.json > env.sh && source env.sh

# Direct to GITHUB_ENV
pipekit env from-json config.json --to-github

# Direct to GITHUB_OUTPUT
pipekit env from-json config.json --to-github-output

# GitLab CI format
pipekit env from-json config.json --to-gitlab
```

</details>

---

### `mask` - Secret Masking

Prevent secrets from leaking in logs.

<details>
<summary><strong>Examples</strong></summary>

```sh
# Mask patterns in a stream
some-command | pipekit mask values --pattern "sk-.*" --pattern "password=.*"

# Partial masking (show first/last 3 chars)
echo "sk-1234567890xf" | pipekit mask values --pattern "sk-.*" --partial 3
# Output: sk-***0xf

# Mask a file before outputting
pipekit mask file output.log --pattern "token=\S+"

# GitHub Actions log masking
pipekit mask github "$SECRET_VALUE"

# Mask all env vars matching patterns
pipekit mask env --env-match "*_SECRET,*_TOKEN,*_KEY" --github
```

</details>

---

### `transform` - Data Transformation

Transform values without sed/awk gymnastics.

<details>
<summary><strong>base64</strong></summary>

```sh
echo -n "hello" | pipekit transform base64-encode
# aGVsbG8=

echo "aGVsbG8=" | pipekit transform base64-decode
# hello
```

</details>

<details>
<summary><strong>url-encode / url-decode</strong></summary>

```sh
pipekit transform url-encode "hello world&foo=bar"
# hello+world%26foo%3Dbar

pipekit transform url-decode "hello+world%26foo%3Dbar"
# hello world&foo=bar
```

</details>

<details>
<summary><strong>case</strong> - Convert between case formats</summary>

```sh
pipekit transform case --to upper-snake "myServiceName"
# MY_SERVICE_NAME

pipekit transform case --to kebab "MyServiceName"
# my-service-name

pipekit transform case --to pascal "my_service_name"
# MyServiceName
```

Supported: `camel`, `pascal`, `snake`, `upper-snake`, `kebab`, `upper`, `lower`

</details>

<details>
<summary><strong>regex, template, hash</strong></summary>

```sh
# Regex find/replace
echo "foo-123-bar" | pipekit transform regex --find "\d+" --replace "***"

# Go template with env vars
echo "Deploy {{.Env.APP_NAME}} v{{.Env.VERSION}}" | pipekit transform template

# Hash a value
pipekit transform hash "hello"
# 2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824

# Hash a file
pipekit transform hash --file go.sum --algorithm sha256
```

</details>

---

### `summary` - CI/CD Step Summaries

Generate formatted summaries for pipeline UIs.

<details>
<summary><strong>Examples</strong></summary>

```sh
# Append markdown to GITHUB_STEP_SUMMARY
pipekit summary github "## Deploy complete"

# Generate table from JSON
pipekit summary table --title "Deploy Summary" deploy-info.json --to-github-summary

# Status badge
pipekit summary badge --label "Build" --status success --to-github-summary

# Collapsible section (great for logs)
cat build.log | pipekit summary section --title "Build Logs" --to-github-summary
```

</details>

---

### `assert` - Pipeline Guards

Fail the pipeline early with clear messages.

<details>
<summary><strong>Examples</strong></summary>

```sh
# Assert env vars exist
pipekit assert env-exists DEPLOY_TOKEN CLUSTER_NAME IMAGE_TAG

# Assert files exist
pipekit assert file-exists Dockerfile docker-compose.yml

# Assert JSON value
pipekit assert json-path --file package.json --path ".version" --expected "1.0.0"

# Validate semver
pipekit assert semver "1.2.3"

# Compare versions
pipekit assert compare 2.0.0 gt 1.5.0

# Assert URL returns 200
pipekit assert url https://api.example.com/health --expected-status 200,204
```

</details>

---

### `matrix` - Dynamic Matrix Generation

Generate CI matrix JSON for GitHub Actions `fromJSON()`.

<details>
<summary><strong>Examples</strong></summary>

```sh
# Matrix from directory names
pipekit matrix from-dirs services/ --key service --to-github-output matrix
# {"service":["api","web","worker"]}

# Matrix from files
pipekit matrix from-files "configs/*.yaml" --key config

# Cartesian product
pipekit matrix combine --set "os=linux,darwin" --set "arch=amd64,arm64"
# {"include":[{"arch":"amd64","os":"linux"},{"arch":"amd64","os":"darwin"},...]}

# Filter JSON array
cat services.json | pipekit matrix from-json --key service --filter-field deploy --filter-value true
```

</details>

---

### `notify` - Webhook Notifications

Send structured notifications without crafting curl + JSON payloads.

<details>
<summary><strong>Examples</strong></summary>

```sh
# Slack
pipekit notify slack --status success --title "Deploy v1.2.3 to prod" \
  --field "env=production" --field "duration=45s"

# Discord
pipekit notify discord --status failure --title "Build failed" --message "See logs for details"

# Teams
pipekit notify teams --status warning --title "Disk usage at 85%"

# Generic webhook
pipekit notify webhook --url https://hooks.example.com/deploy --from-json payload.json
```

Webhook URLs are read from `--url` or env vars: `SLACK_WEBHOOK_URL`, `DISCORD_WEBHOOK_URL`, `TEAMS_WEBHOOK_URL`.

</details>

---

### `wait` - Health Check & Readiness Polling

Wait for services to become ready.

<details>
<summary><strong>Examples</strong></summary>

```sh
# Wait for HTTP endpoint
pipekit wait url http://localhost:8080/healthz --timeout 150s --interval 5s

# Wait with expected body content
pipekit wait url http://localhost:8080/healthz --expected-body "healthy"

# Wait for TCP port
pipekit wait tcp localhost:5432 --timeout 60s

# Wait for a command to succeed
pipekit wait command "pg_isready -h localhost" --timeout 30s --backoff

# Quiet mode (just exit code)
pipekit wait url http://localhost:8080/healthz --quiet
```

**Flags:**

| Flag | Description | Default |
|------|-------------|---------|
| `--timeout` | Total time before giving up | `120s` |
| `--interval` | Time between retries | `5s` |
| `--backoff` | Exponential backoff | `false` |
| `--quiet` | Suppress output | `false` |
| `--expected-status` | Acceptable HTTP codes | `200` |
| `--expected-body` | Match substring in response | - |

</details>

---

### `diff` - Changed File Detection

Detect changes between git refs. Essential for monorepos.

<details>
<summary><strong>Examples</strong></summary>

```sh
# List changed files
pipekit diff files --base origin/main

# List changed top-level directories
pipekit diff dirs --base origin/main --to-github-output changed_services

# Check if specific paths changed (exit 0 if yes)
pipekit diff match "api/**" --base origin/main && echo "API changed"

# Map changes to service names via config
pipekit diff affected --config .pipekit-diff.yaml --base origin/main --output json
```

**Config file format** (`.pipekit-diff.yaml`):

```yaml
services:
  api:
    - api/
    - shared/
  web:
    - web/
    - shared/
  worker:
    - worker/
```

</details>

---

### `version` - Version Management

Extract, bump, and compare versions across file formats.

<details>
<summary><strong>Examples</strong></summary>

```sh
# Auto-detect and read version
pipekit version get
# 1.2.3

# From specific file
pipekit version get --source package.json --format v-prefixed
# v1.2.3

# Bump patch
pipekit version bump patch --source package.json
# 1.2.4

# Bump with pre-release
pipekit version bump minor --source Cargo.toml --pre-release alpha.1
# 1.3.0-alpha.1

# Compare versions (exit: 0=eq, 1=gt, 2=lt)
pipekit version compare 2.0.0 1.5.0
# greater

# Next version from conventional commits
pipekit version next --to-github-output version
```

**Auto-detected files:** `package.json`, `Cargo.toml`, `pyproject.toml`, `Chart.yaml`, `VERSION`, `version.txt`, `setup.py`, `build.gradle`, `pom.xml`

</details>

---

### `retry` - Command Retry

Run any command with configurable retry logic.

<details>
<summary><strong>Examples</strong></summary>

```sh
# Basic retry
pipekit retry run --attempts 3 --delay 10s -- npm publish

# With exponential backoff
pipekit retry run --attempts 5 --delay 5s --backoff -- helm upgrade --install myapp ./chart

# Only retry on specific exit codes
pipekit retry run --attempts 3 --delay 30s --on-exit-codes 1,137 -- ./deploy.sh
```

</details>

---

### `cache-key` - Cache Key Generation

Generate deterministic cache keys from files or directories.

<details>
<summary><strong>Examples</strong></summary>

```sh
# Hash lockfiles
pipekit cache-key from-files go.sum --prefix "go-mod-linux-" --to-github-output cache_key
# go-mod-linux-a1b2c3d4...

# Hash all matching files
pipekit cache-key from-glob "**/*.lock" --prefix "deps-"

# Composite key from multiple parts
pipekit cache-key composite linux amd64 "$(pipekit transform hash --file go.sum)" --prefix "go-"
# go-linux-amd64-a1b2c3d4...
```

</details>

---

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Success |
| `1` | Error / assertion failed |
| `2` | Used by `version compare` (v1 < v2) |

---

## License

MIT - see [LICENSE](LICENSE)
