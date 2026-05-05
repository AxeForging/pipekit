# Commands

Full reference for every pipekit command and flag. For end-to-end pipeline recipes see **[EXAMPLES.md](EXAMPLES.md)**.

## Contents

- [`env`](#env) — JSON/YAML/dotenv → env vars
- [`mask`](#mask) — secret masking
- [`transform`](#transform) — value transformations
- [`summary`](#summary) — CI/CD step summaries
- [`assert`](#assert) — pipeline guards
- [`matrix`](#matrix) — dynamic matrix generation
- [`notify`](#notify) — webhook notifications
- [`wait`](#wait) — readiness polling
- [`diff`](#diff) — changed-file detection
- [`version`](#version) — version management
- [`retry`](#retry) — command retry
- [`cache-key`](#cache-key) — deterministic cache keys
- [`config`](#config) — environment configuration
- [`parse`](#parse) — structured data extraction
- [Exit codes](#exit-codes)

---

## env

Extract data from structured files and inject as env vars.

<details>
<summary><strong><code>env from-json</code></strong> — parse JSON</summary>

```sh
# Flat JSON to shell exports (stdout)
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

| Flag | Description |
|---|---|
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
<summary><strong><code>env from-yaml</code></strong> — parse YAML</summary>

```sh
pipekit env from-yaml config.yaml --flatten --uppercase-keys --to-github
```

Same flag set as `from-json`.

</details>

<details>
<summary><strong><code>env from-dotenv</code></strong> — parse <code>.env</code> files</summary>

```sh
pipekit env from-dotenv .env --to-github
pipekit env from-dotenv .env.production --prefix "PROD_" --to-github
```

Lines starting with `#` are skipped, surrounding quotes are stripped, leading `export ` is removed.

</details>

<details>
<summary><strong>Output targets</strong></summary>

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

## mask

Prevent secrets from leaking in logs.

<details>
<summary><strong>Examples</strong></summary>

```sh
# Mask patterns in a stream
some-command | pipekit mask values --pattern "sk-.*" --pattern "password=.*"

# Partial masking (show first/last 3 chars)
echo "sk-1234567890xf" | pipekit mask values --pattern "sk-.*" --partial 3
# sk-***0xf

# Mask a file before outputting
pipekit mask file output.log --pattern "token=\S+"

# GitHub Actions log masking (::add-mask::)
pipekit mask github "$SECRET_VALUE"

# Mask all env vars matching glob patterns
pipekit mask env --env-match "*_SECRET,*_TOKEN,*_KEY" --github
```

</details>

---

## transform

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
<summary><strong>case</strong> — convert between case formats</summary>

```sh
pipekit transform case --to upper-snake "myServiceName"
# MY_SERVICE_NAME

pipekit transform case --to kebab "MyServiceName"
# my-service-name

pipekit transform case --to pascal "my_service_name"
# MyServiceName
```

Supported: `camel`, `pascal`, `snake`, `upper-snake`, `kebab`, `upper`, `lower`.

</details>

<details>
<summary><strong>regex / template / hash</strong></summary>

```sh
# Regex find/replace
echo "foo-123-bar" | pipekit transform regex --find "\d+" --replace "***"

# Go template with env vars
echo "Deploy {{.Env.APP_NAME}} v{{.Env.VERSION}}" | pipekit transform template

# Hash a literal value
pipekit transform hash "hello"
# 2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824

# Hash a file (default sha256; sha1, md5 also supported)
pipekit transform hash --file go.sum --algorithm sha256
```

</details>

<details>
<summary><strong>slug</strong> — URL-safe slugs from branches / strings</summary>

```sh
echo "feature/my-cool-feature" | pipekit transform slug
# feature-my-cool-feature

# Strips refs/heads/ automatically
echo "refs/heads/release/v1.2.3" | pipekit transform slug
# release-v123

# Default max length is 63 (matches k8s label limits)
echo "feature/very-long-branch-name" | pipekit transform slug --max-length 20

# Add a prefix
echo "$GITHUB_HEAD_REF" | pipekit transform slug --prefix "preview-"
```

Useful for preview-deployment names, Cloudflare Worker names, k8s-friendly identifiers.

</details>

---

## summary

Generate formatted summaries for pipeline UIs (GitHub `$GITHUB_STEP_SUMMARY` and friends).

<details>
<summary><strong>Examples</strong></summary>

```sh
# Append markdown to GITHUB_STEP_SUMMARY
pipekit summary github "## Deploy complete"

# Markdown table from JSON
pipekit summary table --title "Deploy Summary" deploy-info.json --to-github-summary

# Status badge
pipekit summary badge --label "Build" --status success --to-github-summary

# Collapsible section (great for logs)
cat build.log | pipekit summary section --title "Build Logs" --to-github-summary
```

</details>

---

## assert

Fail the pipeline early with clear messages.

<details>
<summary><strong>Examples</strong></summary>

```sh
# Required env vars
pipekit assert env-exists DEPLOY_TOKEN CLUSTER_NAME IMAGE_TAG

# Required files
pipekit assert file-exists Dockerfile docker-compose.yml

# Value at a JSON path
pipekit assert json-path --file package.json --path ".version" --expected "1.0.0"

# Valid semver
pipekit assert semver "1.2.3"

# Compare two versions
pipekit assert compare 2.0.0 gt 1.5.0

# URL returns one of the expected statuses
pipekit assert url https://api.example.com/health --expected-status 200,204
```

Comparison operators: `gt`, `lt`, `eq`, `gte`, `lte` (and `>`, `<`, `==`, `>=`, `<=`).

</details>

---

## matrix

Generate matrix JSON for GitHub Actions `fromJSON()` (or any matrix-aware runner).

<details>
<summary><strong>Examples</strong></summary>

```sh
# From directory names
pipekit matrix from-dirs services/ --key service --to-github-output matrix
# {"service":["api","web","worker"]}

# From files matching a glob
pipekit matrix from-files "configs/*.yaml" --key config

# Cartesian product
pipekit matrix combine --set "os=linux,darwin" --set "arch=amd64,arm64"
# {"include":[{"arch":"amd64","os":"linux"},{"arch":"amd64","os":"darwin"},...]}

# Filter a JSON array of objects
cat services.json | pipekit matrix from-json --key service \
  --filter-field deploy --filter-value true
```

</details>

---

## notify

Send structured notifications without crafting curl + JSON payloads.

<details>
<summary><strong>Examples</strong></summary>

```sh
# Slack
pipekit notify slack --status success --title "Deploy v1.2.3 to prod" \
  --field "env=production" --field "duration=45s"

# Discord
pipekit notify discord --status failure --title "Build failed" \
  --message "See logs for details"

# Microsoft Teams (Adaptive Card)
pipekit notify teams --status warning --title "Disk usage at 85%"

# Generic webhook with a payload file
pipekit notify webhook --url https://hooks.example.com/deploy --from-json payload.json
```

Status values: `success`, `failure`, `warning`, anything else maps to "info". Webhook URLs come from `--url` or `SLACK_WEBHOOK_URL` / `DISCORD_WEBHOOK_URL` / `TEAMS_WEBHOOK_URL`.

</details>

---

## wait

Wait for services to become ready.

<details>
<summary><strong>Examples</strong></summary>

```sh
# HTTP endpoint
pipekit wait url http://localhost:8080/healthz --timeout 150s --interval 5s

# HTTP with body content match
pipekit wait url http://localhost:8080/healthz --expected-body "healthy"

# TCP port
pipekit wait tcp localhost:5432 --timeout 60s

# Arbitrary command (exit 0 = ready)
pipekit wait command "pg_isready -h localhost" --timeout 30s --backoff

# Quiet mode (only the exit code matters)
pipekit wait url http://localhost:8080/healthz --quiet
```

| Flag | Description | Default |
|---|---|---|
| `--timeout` | Total time before giving up | `120s` |
| `--interval` | Time between retries | `5s` |
| `--backoff` | Exponential backoff | `false` |
| `--quiet` | Suppress per-attempt output | `false` |
| `--expected-status` | Acceptable HTTP codes (csv) | `200` |
| `--expected-body` | Substring to look for in response body | — |

</details>

---

## diff

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

# Map file changes to service names via config
pipekit diff affected --config .pipekit-diff.yaml --base origin/main --output json
```

`.pipekit-diff.yaml`:

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

Output formats: `json`, `csv`, `list` (newline-separated, default).

</details>

---

## version

Extract, bump, and compare versions across file formats.

<details>
<summary><strong>Examples</strong></summary>

```sh
# Auto-detect and read version
pipekit version get
# 1.2.3

# From a specific file
pipekit version get --source package.json --format v-prefixed
# v1.2.3

# Bump patch
pipekit version bump patch --source package.json
# 1.2.4

# Bump with pre-release
pipekit version bump minor --source Cargo.toml --pre-release alpha.1
# 1.3.0-alpha.1

# Compare (exit codes: 0 = eq, 1 = gt, 2 = lt)
pipekit version compare 2.0.0 1.5.0

# Next version from conventional commits since the last tag
pipekit version next --to-github-output version
```

**Auto-detected files:** `package.json`, `Cargo.toml`, `pyproject.toml`, `Chart.yaml`, `VERSION`, `version.txt`, `setup.py`, `build.gradle`, `pom.xml`.

</details>

---

## retry

Run any command with configurable retry logic.

<details>
<summary><strong>Examples</strong></summary>

```sh
# Basic retry
pipekit retry run --attempts 3 --delay 10s -- npm publish

# Exponential backoff
pipekit retry run --attempts 5 --delay 5s --backoff -- helm upgrade --install myapp ./chart

# Only retry on specific exit codes
pipekit retry run --attempts 3 --delay 30s --on-exit-codes 1,137 -- ./deploy.sh
```

Use `--` to separate pipekit flags from the command being retried.

</details>

---

## cache-key

Generate deterministic cache keys from files or directories.

<details>
<summary><strong>Examples</strong></summary>

```sh
# Hash a single lockfile
pipekit cache-key from-files go.sum --prefix "go-mod-linux-" --to-github-output cache_key
# go-mod-linux-a1b2c3d4...

# Hash everything matching a glob
pipekit cache-key from-glob "**/*.lock" --prefix "deps-"

# Composite key from multiple parts
pipekit cache-key composite linux amd64 "$(pipekit transform hash --file go.sum)" --prefix "go-"
# go-linux-amd64-a1b2c3d4...
```

</details>

---

## config

Resolve environment-specific configuration from structured maps. Replaces ~80 lines of bash that typically maps `dev/staging/prod` to project IDs, region names, etc.

<details>
<summary><strong><code>config resolve</code></strong> — pull values for an env, with alias support</summary>

```sh
# Given a config file with dev/staging/production keys:
pipekit config resolve envs.json --env prod --to-github
# Normalizes "prod" → "production", exports values to $GITHUB_ENV

# From stdin
echo '{"dev": {"project_id": "my-dev"}, "production": {"project_id": "my-prod"}}' \
  | pipekit config resolve --env develop --uppercase-keys

# YAML config
pipekit config resolve envs.yaml --env staging --format yaml --to-github-output

# Custom aliases
pipekit config resolve envs.json --env preview \
  --aliases '{"preview": "staging", "canary": "production"}'

# Output as compact JSON
pipekit config resolve envs.json --env prod --json
# {"project_id":"my-prod","region":"eu-west1"}
```

**Built-in aliases:** `dev` / `develop` / `development` / `test` / `testing` → `dev` · `stage` / `staging` → `staging` · `prod` / `production` → `production`.

| Flag | Description |
|---|---|
| `--env, -e` | Environment name (required) |
| `--format, -f` | Config format: `json` (default), `yaml` |
| `--aliases` | Custom aliases as JSON |
| `--json` | Output as compact JSON instead of key-value pairs |
| `--uppercase-keys, -u` | Convert keys to UPPER_SNAKE_CASE |
| `--prefix, -p` | Add prefix to all keys |
| `--to-github` | Write to `$GITHUB_ENV` |
| `--to-github-output` | Write to `$GITHUB_OUTPUT` |

</details>

<details>
<summary><strong><code>config branch-env</code></strong> — map branches to environments</summary>

```sh
# Map a branch name to an environment
pipekit config branch-env main \
  --mapping '{"main":"production","develop":"dev","release/*":"staging"}'
# production

# refs/heads/ prefix is stripped automatically
pipekit config branch-env refs/heads/release/v1.2.0 \
  --mapping '{"main":"production","release/*":"staging"}'
# staging

# Auto-detect from $GITHUB_REF
pipekit config branch-env --mapping '{"main":"production","develop":"dev"}' --to-github
# Writes TARGET_ENV=production to $GITHUB_ENV

# Custom output key
pipekit config branch-env develop --mapping '{"develop":"dev"}' \
  --output-key DEPLOY_ENV --to-github-output
```

| Flag | Description |
|---|---|
| `--mapping, -m` | Branch-to-env JSON mapping (required) |
| `--output-key` | Output variable name (default: `TARGET_ENV`) |
| `--to-github` | Write to `$GITHUB_ENV` |
| `--to-github-output` | Write to `$GITHUB_OUTPUT` |

</details>

---

## parse

Extract code blocks and structured data from markdown text. Useful for parsing GitHub issue bodies, PR comments, and other markdown sources in CI.

<details>
<summary><strong><code>parse extract-block</code></strong> — fenced code blocks</summary>

```sh
# All code blocks as a JSON array
echo "$ISSUE_BODY" | pipekit parse extract-block
# [{"language":"yaml","content":"foo: bar","index":0},{"language":"json","content":"{...}","index":1}]

# Filter by language
echo "$COMMENT" | pipekit parse extract-block --language yaml

# Raw content of a specific block
echo "$COMMENT" | pipekit parse extract-block --language python --index 0 --content-only
# print("hello")

# First yaml block straight to GITHUB_OUTPUT
echo "$ISSUE_BODY" | pipekit parse extract-block --language yaml --to-github-output
```

Supports both ` ``` ` and `~~~` fences with any language tag.

| Flag | Description |
|---|---|
| `--language, -l` | Filter by language tag (case-insensitive) |
| `--index, -i` | Return only the Nth block (0-based) |
| `--content-only` | Output raw content without JSON wrapping |
| `--to-github-output` | Write content to `$GITHUB_OUTPUT` |
| `--output-key` | Variable name for `--to-github-output` (default: `PARSED_BLOCK`) |

</details>

<details>
<summary><strong><code>parse extract-yaml</code></strong> — extract and parse YAML blocks</summary>

```sh
# Parse YAML blocks from an issue body
echo "$ISSUE_BODY" | pipekit parse extract-yaml
# [{"env":"production","replicas":3}]

# Export parsed values as env vars
echo "$ISSUE_BODY" | pipekit parse extract-yaml --to-github -u

# A specific YAML block
echo "$COMMENT" | pipekit parse extract-yaml --index 0

# As JSON to GITHUB_OUTPUT
echo "$ISSUE_BODY" | pipekit parse extract-yaml --to-github-output
```

Matches blocks tagged `yaml`, `yml`, or untagged blocks that parse as valid YAML. Invalid YAML blocks are silently skipped.

| Flag | Description |
|---|---|
| `--index, -i` | Return only the Nth YAML block (0-based) |
| `--to-env` | Export top-level keys as shell export statements |
| `--to-github` | Write top-level keys to `$GITHUB_ENV` |
| `--to-github-output` | Write parsed JSON to `$GITHUB_OUTPUT` |
| `--uppercase-keys, -u` | Convert keys to UPPER_SNAKE_CASE |
| `--output-key` | Variable name for `--to-github-output` (default: `PARSED_YAML`) |

</details>

---

## Exit codes

| Code | Meaning |
|---|---|
| `0` | Success |
| `1` | Error / assertion failed / non-success in most commands |
| `2` | Used by `version compare` to signal `v1 < v2` |

`assert *` and `wait *` propagate non-zero on failure, so they slot directly into `set -e` pipelines.

---

**See also:** [Examples](EXAMPLES.md) · [Requirements](REQUIREMENTS.md) · [Install](INSTALL.md) · [← README](../README.md)
