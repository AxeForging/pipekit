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
- [`checksum`](#checksum) — release checksums
- [`artifact`](#artifact) — artifact manifests and assertions
- [`git`](#git) — CI-friendly git metadata
- [`changelog`](#changelog) — release notes from git history
- [`config`](#config) — environment configuration
- [`parse`](#parse) — structured data extraction
- [`json` / `yaml`](#json) — read · query · mutate · merge · convert · pretty · table
- [`render`](#render) — file templating with sprig-like funcs
- [`exec`](#exec) — unified retry + mask + tee + timeout runner
- [`url`](#url) — URL parsing
- [`image`](#image) — container image ref parsing
- [`time`](#time) — timestamps, formatting, arithmetic
- [`port` · `uuid` · `random`](#misc) — small generators
- [`doctor`](#doctor) — environment diagnostics
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
<summary><strong><code>env from-toml</code></strong> — parse TOML (Cargo, pyproject, etc.)</summary>

```sh
pipekit env from-toml Cargo.toml --flatten --uppercase-keys --to-github
pipekit env from-toml pyproject.toml --filter '.tool.poetry' --flatten --to-github
```

Same flag set as `from-json`. Closes the Cargo / pyproject gap.

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

# Use built-in presets for common secret formats
some-command | pipekit mask values --preset "aws,github,jwt"

# Cross-line patterns (PEM keys, multi-line JWTs)
cat output.log | pipekit mask values --preset "gcp,pem" --multiline

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

**Presets:** `aws`, `github`, `gcp`, `jwt`, `slack`, `stripe`, `pem`. Combine via comma: `--preset aws,github`. Use `--multiline` for patterns that need to match across newlines (the default line-by-line scanner won't see them).

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

# Path exists (file or directory)
pipekit assert path /etc/myapp/config.yaml /var/lib/myapp

# Directory exists and contains entries
pipekit assert dir-not-empty ./build/output
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

# Test sharding — run shard 1 of 4 in parallel jobs
pipekit matrix shard --total 4 --index 1 \
  $(go test -list . ./... | tail -n +2)
# Or read items from stdin
go test -list . ./... | tail -n +2 \
  | pipekit matrix shard --total 4 --index 1 --from-stdin-lines
```

`shard` outputs in `list` format by default (one per line). Use `--format csv` or `--format json` for other shapes.

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

# Set an explicit version (replaces only the version literal in the right
# field — never rewrites a dep pin that happens to share the value)
pipekit version set 1.5.0 --source package.json
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

# Hash + mix in env values (e.g. tool versions) — key changes when env changes
pipekit cache-key from-files go.sum \
  --with-env "GO_VERSION,RUNNER_OS" --length 16 --prefix "go-"
# go-a1b2c3d4e5f60718

# Truncate the hash output to N hex chars
pipekit cache-key from-files go.sum --length 8

# Hash everything matching a glob
pipekit cache-key from-glob "**/*.lock" --prefix "deps-"

# Composite key from multiple parts
pipekit cache-key composite linux amd64 "$(pipekit transform hash --file go.sum)" --prefix "go-"
# go-linux-amd64-a1b2c3d4...
```

| Flag | On | Description |
|---|---|---|
| `--with-env` | `from-files` | Comma-separated env var names to mix into the hash (sorted, deterministic) |
| `--length` | `from-files` | Truncate hex output to N chars (0 = full) |

</details>

---

## checksum

Generate and verify release checksums without shell loops.

<details>
<summary><strong>Examples</strong></summary>

```sh
# Write a standard sha256 checksum file
pipekit checksum files dist/* --output dist/checksums.txt

# JSON output for manifests or summaries
pipekit checksum files dist/* --format json

# Verify checksums before uploading artifacts
pipekit checksum verify dist/checksums.txt

# Alternate algorithms
pipekit checksum files dist/* --algorithm sha1
```

| Subcommand | Description |
|---|---|
| `checksum files FILE...` | Hash each file independently |
| `checksum verify CHECKSUM_FILE` | Verify `<checksum> <path>` lines |

Flags: `--algorithm sha256|sha1|md5`, `--format text|json`, `--output`.

</details>

---

## artifact

Validate and describe CI artifacts before upload or release.

<details>
<summary><strong>Examples</strong></summary>

```sh
# Fail early if expected build outputs are missing
pipekit artifact assert "dist/pipekit-linux-*" "dist/checksums.txt"

# Generate a JSON manifest with path, size, and sha256
pipekit artifact manifest "dist/pipekit-*" --pretty --output dist/artifacts.json
```

| Subcommand | Description |
|---|---|
| `artifact assert PATH_OR_GLOB...` | Fail unless each path/glob resolves to at least one file |
| `artifact manifest PATH_OR_GLOB...` | Emit JSON artifact metadata |

</details>

---

## git

Read git metadata in formats that are easy to pass between CI steps.

<details>
<summary><strong>Examples</strong></summary>

```sh
pipekit git sha --short --to-github-output git_sha
pipekit git ref --slug --max-length 40 --to-github-output ref_slug
pipekit git current-tag
pipekit git previous-tag
pipekit git is-dirty --print
```

| Subcommand | Description |
|---|---|
| `git sha` | Print current commit SHA (`--short` supported) |
| `git ref` | Print current branch/tag, honoring GitHub Actions env vars |
| `git current-tag` | Print tag pointing at `HEAD` |
| `git previous-tag` | Print latest reachable tag |
| `git is-dirty` | Detect uncommitted tracked/untracked changes |

</details>

---

## changelog

Generate markdown release notes from git commits.

<details>
<summary><strong>Examples</strong></summary>

```sh
# Commits since a tag
pipekit changelog generate --from v1.2.0 --to HEAD

# Group conventional commits into Features / Fixes / Maintenance
pipekit changelog generate --from v1.2.0 --conventional --output RELEASE_NOTES.md

# Use the latest reachable tag as the start point
pipekit changelog since-tag --conventional --to-github-output release_notes
```

| Subcommand | Description |
|---|---|
| `changelog generate` | Generate notes for `--from..--to` |
| `changelog since-tag` | Generate notes since the latest reachable tag |

Flags: `--from`, `--to`, `--conventional`, `--format markdown|json`, `--output`, `--to-github-output`.

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

</details>

<details>
<summary><strong><code>parse extract-frontmatter</code></strong> — pull leading YAML / TOML frontmatter</summary>

```sh
# Extract the raw YAML body
pipekit parse extract-frontmatter post.md
# title: My Post
# draft: true

# As JSON (parses YAML or TOML automatically)
pipekit parse extract-frontmatter post.md --json
# {"title":"My Post","draft":true}

# Top-level keys straight to $GITHUB_ENV (UPPER_SNAKE_CASE optional)
pipekit parse extract-frontmatter post.md --to-github -u
```

Supports both `--- ... ---` (YAML) and `+++ ... +++` (TOML) delimiters. Common in Hugo / Jekyll content, issue templates, and notebook headers.

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

## comment

Render and manage markdown comments with hidden anchors. This is useful for PR bot comments where the visible body should be readable, but automation needs a stable marker to find and update the right comment later.

Hidden anchors use HTML comments, so GitHub keeps them in the API body but does not render them:

```md
<!-- pipekit:preview -->
```

<details>
<summary><strong><code>comment render</code></strong> — create an anchored comment body</summary>

```sh
pipekit comment render --anchor preview --body-file preview.md > comment.md

printf '## Preview\n\nReady\n' \
  | pipekit comment render --anchor preview
```

| Flag | Description |
|---|---|
| `--anchor, -a` | Hidden anchor name |
| `--body-file` | Read visible markdown body from a file |
| `--output, -o` | Write output to a file |

</details>

<details>
<summary><strong><code>comment fence</code></strong> — render safe fenced code blocks</summary>

```sh
pipekit comment fence --language yaml values.yaml

cat script.js | pipekit comment fence --language js
```

The fence is automatically lengthened when the content itself contains triple backticks.

</details>

<details>
<summary><strong><code>comment inspect</code></strong> — read anchors and code blocks</summary>

```sh
pipekit comment inspect comment.md

gh api repos/OWNER/REPO/issues/123/comments \
  | pipekit comment inspect
```

Outputs JSON with comment metadata, hidden anchors, and fenced code blocks.

</details>

<details>
<summary><strong><code>comment payload</code></strong> — create a GitHub comment API payload</summary>

```sh
pipekit comment payload comment.md > payload.json

gh api \
  --method POST \
  repos/OWNER/REPO/issues/123/comments \
  --input payload.json
```

Outputs JSON in the shape expected by GitHub's issue comments API:

```json
{"body":"...markdown..."}
```

</details>

<details>
<summary><strong><code>comment select</code></strong> — select a comment by anchor</summary>

```sh
gh api repos/OWNER/REPO/issues/123/comments \
  | pipekit comment select --anchor preview --format id

gh api repos/OWNER/REPO/issues/123/comments \
  | pipekit comment select --anchor preview --format body > existing.md
```

| Flag | Description |
|---|---|
| `--anchor, -a` | Hidden anchor to search for |
| `--format, -f` | `json`, `id`, `body`, or `url` |

</details>

<details>
<summary><strong><code>comment amend</code></strong> — replace visible content after an anchor</summary>

```sh
pipekit comment amend existing.md --anchor preview --body-file preview.md > updated.md
```

If the input does not contain the anchor, a fresh anchored comment is created.

</details>

---

## json

Read, query, mutate, deep-merge, convert, and pretty-print JSON. The `yaml` command is identical except the default format for stdin is YAML — both share the same subcommands and per-file decoding still uses the file's extension (`.json`, `.yaml`, `.toml`, `.csv`).

<details>
<summary><strong><code>json get</code></strong> — extract a value at a jq-style path</summary>

```sh
pipekit json get values.yaml --path '.image.tag' --raw
# v1.2.3

# Pipe stdin
echo '{"a":{"b":42}}' | pipekit json get --path '.a.b'

# Write to GITHUB_OUTPUT
pipekit json get values.yaml --path '.image.tag' --raw --to-github-output IMAGE_TAG
```

</details>

<details>
<summary><strong><code>json set</code> / <code>json del</code></strong> — write paths, in place or to stdout</summary>

```sh
pipekit json set values.yaml --path '.image.tag' --value 'v2.0.0' --in-place
pipekit json set config.json --path '.flags' --json-value '["a","b"]' --pretty
pipekit json del values.yaml --path '.legacy' --in-place

# Surgical edit — change ONLY the target, keep comments/key-order/quoting/spacing
pipekit yaml set values.yaml --path '.image.tag' --value 'v2.0.0' --in-place --preserve
pipekit json set config.json --path '.newKey' --json-value '{"on":true}' --in-place --preserve  # insert
pipekit json del config.json --path '.legacy' --in-place --preserve
```

By default `set`/`del` parse the document and re-serialize it, which normalizes
formatting (comments dropped, keys reordered, re-indented). Add `--preserve`
(`-P`) for a surgical, byte-level edit that touches **only** the targeted node
and leaves every other byte identical — comments (including their column
alignment), key order, quoting style, indentation, and blank lines are all kept.
Ideal for hand-maintained files like Helm `values.yaml`.

- Supported with `--preserve`: **yaml**, **json** (toml/csv return a clear error).
- Editing an existing value, deleting a key, and **inserting a new key into an
  existing object** are all supported and formatting-matched to siblings.
- In-place writes are **atomic** (temp file + fsync + rename) and keep the
  original file's permission bits, so a crash mid-write can't truncate the file.
- Safety: every YAML splice is re-parsed and validated; if the result wouldn't
  hold the intended value (e.g. a type-ambiguous edit like setting a plain
  numeric field to a numeric string) it automatically falls back to the safe
  re-encode path rather than risk a wrong edit.

</details>

<details>
<summary><strong><code>json merge</code></strong> — deep-merge files (helm-style overrides)</summary>

```sh
pipekit json merge base.yaml prod.yaml --pretty --output merged.yaml
pipekit json merge a.json b.json c.json --format yaml
```

Maps are deep-merged; scalars and slices in later files override.

</details>

<details>
<summary><strong><code>json convert</code> / <code>json pretty</code> / <code>json table</code></strong></summary>

```sh
# Convert formats — extension auto-detected, or use --from
pipekit json convert config.toml --to yaml
pipekit json convert services.csv --to json --pretty

# Pretty-print
pipekit json pretty messy.json --indent 4

# Aligned text table from an array of objects
pipekit json table services.json --columns name,version,replicas
```

</details>

---

## render

Render Go templates with a focused sprig-like FuncMap. Supports stacked `--values` files (deep-merged), inline `--set key=value` overrides (dotted keys), `--output` to a file.

<details>
<summary><strong>Examples</strong></summary>

```sh
# Helm-style values rendering
pipekit render deployment.yaml.tpl \
  --values base.yaml --values prod.yaml \
  --set image.tag=v1.2.3 \
  --output deployment.yaml

# Inline template through stdin
echo 'Hello {{ .Values.name | default "world" }}' \
  | pipekit transform template
```

**Available functions:** `default`, `env`, `envOr`, `b64enc`, `b64dec`, `sha256sum`, `sha1sum`, `md5sum`, `regexReplace`, `regexMatch`, `replace`, `lower`, `upper`, `trim`, `trimPrefix`, `trimSuffix`, `contains`, `hasPrefix`, `hasSuffix`, `split`, `join`, `quote`, `squote`, `indent`, `nindent`, `ternary`, `toJson`, `toYaml`, `fromJson`, `fromYaml`, `now`, `date`, `list`, `dict`.

</details>

---

## exec

Unified runner that combines retry, masking, timeout, and tee in one verb. Replaces stacked `pipekit retry run -- bash -c "cmd | pipekit mask"`.

<details>
<summary><strong>Examples</strong></summary>

```sh
# Retry with backoff, mask any preset patterns, tee to a file
pipekit exec --attempts 3 --backoff --jitter \
  --mask-preset "github,aws" \
  --tee deploy.log \
  -- ./deploy.sh

# Per-attempt timeout + total deadline
pipekit exec --attempts 5 --delay 5s --timeout 30s --max-elapsed 5m \
  -- helm upgrade --install myapp ./chart

# Only retry when stderr matches a regex (e.g. rate limiting)
pipekit exec --attempts 4 --delay 10s --retry-on-stderr "rate limit|429" \
  -- npm publish
```

| Flag | Description | Default |
|---|---|---|
| `--attempts, -a` | Number of attempts | `1` |
| `--delay, -d` | Initial delay between retries | `5s` |
| `--backoff` | Double the delay after each fail | `false` |
| `--jitter` | Add up to 20% jitter to retry delays | `false` |
| `--timeout, -t` | Per-attempt timeout | — |
| `--max-elapsed` | Total deadline across all attempts | — |
| `--mask` | Regex to mask in stdout/stderr (repeatable) | — |
| `--mask-preset` | Comma-separated: aws,github,gcp,jwt,slack,stripe,pem | — |
| `--mask-repl` | Replacement string | `***` |
| `--tee` | Also write combined output to this file | — |
| `--retry-on-stderr` | Regex; only retry when stderr matches | — |

</details>

---

## url

```sh
# Split a URL into env vars
pipekit url parse "postgres://app:secret@db.internal:5432/prod" --prefix DB_ --to-github
# DB_SCHEME=postgres
# DB_HOST=db.internal
# DB_PORT=5432
# DB_USER=app
# DB_PASSWORD=secret
# DB_PATH=/prod
```

Empty components are dropped.

---

## image

```sh
# Parse a container image reference
pipekit image parse "ghcr.io/org/repo:v1.2.3@sha256:abc..." --prefix IMG_ --to-github
# IMG_REGISTRY=ghcr.io  IMG_REPOSITORY=org/repo  IMG_TAG=v1.2.3  IMG_DIGEST=sha256:abc...

# JSON output
pipekit image parse "redis:7" --json
# {"Registry":"docker.io","Repository":"library/redis","Tag":"7","Digest":""}
```

Defaults: `redis` → `docker.io/library/redis:latest`. Registry is detected by `.`, `:`, or `localhost` in the leading segment.

---

## time

<details>
<summary><strong>Examples</strong></summary>

```sh
# Named layouts
pipekit time now --format rfc3339   # 2026-05-05T12:30:45Z
pipekit time now --format unix      # 1778070645
pipekit time now --format compact   # 20260505-123045
pipekit time now --format tag       # 20260505-1230 (great for image tags)
pipekit time now --format date      # 2026-05-05

# Format conversion
pipekit time format "2026-05-05T12:30:45Z" --from rfc3339 --to "Jan 2, 2006"

# Arithmetic
pipekit time add 30m --format rfc3339              # 30m from now
pipekit time add 1h --from "2026-05-05T12:00:00Z"  # 1h from explicit base
```

Named layouts: `rfc3339`, `rfc1123`, `rfc822`, `unix`, `unix-ms`, `compact`, `date`, `datetime`, `tag`, `iso`. Anything else is treated as a raw Go time layout.

</details>

---

## misc

```sh
# Free TCP port (OS-picked, or scan a range)
pipekit port free
pipekit port free --low 9000 --high 9100

# UUID v4 (or 8-char short form)
pipekit uuid
pipekit uuid --short

# Cryptographically random string
pipekit random --length 32 --alphabet hex
pipekit random --length 12 --alphabet base32 --to-github-output BUILD_ID
```

Alphabets: `alnum` (default), `alpha`, `hex`, `base32`, `digits`, `lower`, `upper`.

---

## doctor

Diagnose pipekit's runtime environment — CI platform detection, expected env vars, tool availability, and webhook configuration.

```sh
pipekit doctor
# [platform]
# · ci platform            github-actions
# · go runtime             go1.24.6 linux/amd64
#
# [tools]
# ✓ git on PATH
#
# [ci-vars]
# ✓ GITHUB_ENV             set
# ✓ GITHUB_OUTPUT          set
# ⚠ GITHUB_STEP_SUMMARY    not set
# ...

pipekit doctor --json   # for piping into jq / pipekit json
```

Detects: GitHub Actions, GitLab CI, Buildkite, CircleCI, Jenkins, plus a generic `CI=true` fallback.

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
