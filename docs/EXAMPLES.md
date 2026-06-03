# Examples

End-to-end recipes that show pipekit replacing real pipeline bash. Each example is copy-pasteable into the relevant CI system.

For per-command flag reference see **[COMMANDS.md](COMMANDS.md)**.

## Contents

- [Quick wins (one-line bash → one-line pipekit)](#quick-wins)
- [GitHub Actions](#github-actions)
  - [Inject config + mask secrets + deploy](#deploy-with-config-injection)
  - [Monorepo: only build changed services](#monorepo-changed-services)
  - [Dynamic build matrix from a directory](#dynamic-build-matrix)
  - [Parse a GitHub issue and act on it](#parse-issue-body)
  - [Preview deployments per branch](#preview-deployments)
  - [Step summaries that look good](#step-summaries)
- [GitLab CI](#gitlab-ci)
- [Jenkins](#jenkins)
- [Combining commands](#combining-commands)

---

## Quick wins

Direct bash → pipekit swaps. Each one removes a pile of `jq` / `sed` / `awk` / `curl` boilerplate.

<details>
<summary><strong>JSON config → env vars</strong></summary>

```sh
# BEFORE
for key in $(jq -r 'keys[]' config.json); do
  value=$(jq -r ".[\"$key\"]" config.json)
  echo "${key^^}=$value" >> "$GITHUB_ENV"
done

# AFTER
pipekit env from-json config.json --uppercase-keys --to-github
```

</details>

<details>
<summary><strong>Wait for service to be ready</strong></summary>

```sh
# BEFORE
for i in {1..30}; do
  curl -sf http://localhost:8080/healthz && break
  sleep 5
done

# AFTER
pipekit wait url http://localhost:8080/healthz --timeout 150s
```

</details>

<details>
<summary><strong>Retry a flaky command</strong></summary>

```sh
# BEFORE
n=0; until [ $n -ge 5 ]; do
  helm upgrade --install myapp ./chart && break
  n=$((n+1)); sleep $((5 * 2 ** n))
done

# AFTER
pipekit retry run --attempts 5 --delay 5s --backoff -- helm upgrade --install myapp ./chart
```

</details>

<details>
<summary><strong>Map branch → environment</strong></summary>

```sh
# BEFORE
case "$GITHUB_REF" in
  refs/heads/main) echo "TARGET_ENV=production" >> "$GITHUB_ENV" ;;
  refs/heads/develop) echo "TARGET_ENV=dev" >> "$GITHUB_ENV" ;;
  refs/heads/release/*) echo "TARGET_ENV=staging" >> "$GITHUB_ENV" ;;
  *) echo "TARGET_ENV=preview" >> "$GITHUB_ENV" ;;
esac

# AFTER
pipekit config branch-env --to-github \
  --mapping '{"main":"production","develop":"dev","release/*":"staging"}'
```

</details>

<details>
<summary><strong>Generate a deterministic cache key</strong></summary>

```sh
# BEFORE
KEY="go-mod-$(uname -s)-$(uname -m)-$(sha256sum go.sum | cut -d' ' -f1)"
echo "cache_key=$KEY" >> "$GITHUB_OUTPUT"

# AFTER
pipekit cache-key composite "$(uname -s)" "$(uname -m)" \
  "$(pipekit transform hash --file go.sum)" \
  --prefix "go-mod-" --to-github-output cache_key
```

</details>

<details>
<summary><strong>Mask a value in logs</strong></summary>

```sh
# BEFORE
echo "::add-mask::$SECRET_VALUE"

# AFTER
pipekit mask github "$SECRET_VALUE"
# or, mask many at once
pipekit mask env --env-match "*_TOKEN,*_SECRET,*_KEY" --github
```

</details>

---

## GitHub Actions

### Deploy with config injection

Reads an env-specific config (resolves `prod` → `production` automatically), masks the resulting secrets, asserts the deploy token is present, then deploys.

```yaml
name: Deploy

on:
  workflow_dispatch:
    inputs:
      env:
        type: choice
        options: [dev, staging, prod]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Install pipekit
        run: |
          curl -L https://github.com/AxeForging/pipekit/releases/latest/download/pipekit-linux-amd64.tar.gz | tar xz
          sudo mv pipekit-linux-amd64 /usr/local/bin/pipekit

      - name: Resolve env config
        run: pipekit config resolve envs.json --env "${{ inputs.env }}" --uppercase-keys --to-github

      - name: Mask sensitive values in logs
        run: pipekit mask env --env-match "*_TOKEN,*_SECRET,*_KEY" --github

      - name: Guard required vars
        run: pipekit assert env-exists DEPLOY_TOKEN CLUSTER_NAME IMAGE_TAG

      - name: Deploy
        run: pipekit retry run --attempts 3 --delay 30s --backoff -- ./scripts/deploy.sh

      - name: Notify Slack
        if: always()
        env:
          SLACK_WEBHOOK_URL: ${{ secrets.SLACK_WEBHOOK_URL }}
        run: |
          pipekit notify slack --status "${{ job.status }}" \
            --title "Deploy to ${{ inputs.env }}" \
            --field "image=$IMAGE_TAG" \
            --field "actor=${{ github.actor }}"
```

### Monorepo: changed services

Build only the services whose paths changed in this PR.

```yaml
name: CI

on: [pull_request]

jobs:
  detect:
    runs-on: ubuntu-latest
    outputs:
      services: ${{ steps.diff.outputs.services }}
    steps:
      - uses: actions/checkout@v4
        with: { fetch-depth: 0 }
      - run: |
          curl -L https://github.com/AxeForging/pipekit/releases/latest/download/pipekit-linux-amd64.tar.gz | tar xz
          sudo mv pipekit-linux-amd64 /usr/local/bin/pipekit
      - id: diff
        run: |
          pipekit diff affected --config .pipekit-diff.yaml \
            --base origin/${{ github.base_ref }} \
            --output json --to-github-output services

  build:
    needs: detect
    if: needs.detect.outputs.services != '[]'
    runs-on: ubuntu-latest
    strategy:
      matrix:
        service: ${{ fromJSON(needs.detect.outputs.services) }}
    steps:
      - uses: actions/checkout@v4
      - run: ./build.sh ${{ matrix.service }}
```

`.pipekit-diff.yaml`:

```yaml
services:
  api:    [api/, shared/]
  web:    [web/, shared/]
  worker: [worker/]
```

### Dynamic build matrix

Generate the matrix from the contents of `services/`:

```yaml
jobs:
  generate:
    runs-on: ubuntu-latest
    outputs:
      matrix: ${{ steps.gen.outputs.matrix }}
    steps:
      - uses: actions/checkout@v4
      - id: gen
        run: pipekit matrix from-dirs services/ --key service --to-github-output matrix

  build:
    needs: generate
    runs-on: ubuntu-latest
    strategy:
      matrix: ${{ fromJSON(needs.generate.outputs.matrix) }}
    steps:
      - run: echo "Building ${{ matrix.service }}"
```

### Parse issue body

A `/deploy` comment on an issue includes a YAML block. The workflow parses it, exposes the values as env vars, and acts.

```yaml
on:
  issue_comment:
    types: [created]

jobs:
  deploy:
    if: contains(github.event.comment.body, '/deploy')
    runs-on: ubuntu-latest
    steps:
      - run: |
          echo "${{ github.event.comment.body }}" \
            | pipekit parse extract-yaml --to-github -u
      - run: pipekit assert env-exists ENV REPLICAS IMAGE
      - run: ./deploy.sh
```

A comment like:

````
/deploy

```yaml
env: production
replicas: 3
image: ghcr.io/me/app:v1.2.3
```
````

…sets `ENV=production`, `REPLICAS=3`, `IMAGE=ghcr.io/me/app:v1.2.3`.

### Updatable PR comments

Render a readable PR comment with a hidden anchor, then let `gh` create or update the matching comment.

```yaml
- name: Build preview body
  run: |
    {
      echo "## Preview deployment"
      echo
      echo "URL: https://preview.example.com"
      echo
      pipekit comment fence --language yaml preview.yaml
    } > preview-body.md
    pipekit comment render --anchor preview-deploy --body-file preview-body.md > preview-comment.md

- name: Upsert PR comment
  env:
    GH_TOKEN: ${{ github.token }}
  run: |
    gh api repos/${{ github.repository }}/issues/${{ github.event.pull_request.number }}/comments \
      > comments.json

    if pipekit comment select comments.json --anchor preview-deploy --format id > comment-id.txt; then
      gh api \
        --method PATCH \
        repos/${{ github.repository }}/issues/comments/$(cat comment-id.txt) \
        --field body="$(cat preview-comment.md)"
    else
      gh pr comment ${{ github.event.pull_request.number }} --body-file preview-comment.md
    fi
```

### Preview deployments

Generate a clean, k8s-friendly slug for the preview environment per branch.

```yaml
- id: slug
  run: |
    pipekit transform slug \
      --prefix "preview-" --max-length 40 \
      "${{ github.head_ref }}" >> slug.txt
    echo "slug=$(cat slug.txt)" >> "$GITHUB_OUTPUT"

- run: kubectl apply -f overlays/${{ steps.slug.outputs.slug }}/
```

### Step summaries

Make the run-summary actually useful.

```sh
pipekit summary badge --label "Build" --status success --to-github-summary

pipekit summary table --title "Deployed services" deployed.json --to-github-summary

cat build.log | pipekit summary section --title "Build logs" --to-github-summary
```

---

## GitLab CI

```yaml
variables:
  PIPEKIT_VERSION: latest

before_script:
  - curl -L https://github.com/AxeForging/pipekit/releases/$PIPEKIT_VERSION/download/pipekit-linux-amd64.tar.gz | tar xz
  - mv pipekit-linux-amd64 /usr/local/bin/pipekit

deploy:
  script:
    # Resolve config and source it into the current shell
    - pipekit config resolve envs.yaml --env "$CI_ENVIRONMENT_NAME" --format yaml --to-gitlab > env.sh
    - source env.sh
    - pipekit assert env-exists DEPLOY_TOKEN PROJECT_ID
    - pipekit wait url "$HEALTH_URL" --timeout 60s
    - pipekit retry run --attempts 3 --delay 10s -- ./deploy.sh
```

---

## Jenkins

```groovy
pipeline {
  agent any
  stages {
    stage('Setup') {
      steps {
        sh '''
          curl -L https://github.com/AxeForging/pipekit/releases/latest/download/pipekit-linux-amd64.tar.gz | tar xz
          sudo mv pipekit-linux-amd64 /usr/local/bin/pipekit
        '''
      }
    }
    stage('Deploy') {
      steps {
        sh 'pipekit assert env-exists DEPLOY_TOKEN'
        sh 'pipekit wait url http://localhost:8080/healthz --timeout 120s'
        sh 'pipekit retry run --attempts 3 --delay 15s -- ./deploy.sh'
      }
    }
    stage('Notify') {
      steps {
        sh '''
          pipekit notify slack \
            --status "${currentBuild.currentResult}" \
            --title "Build #${BUILD_NUMBER} on ${BRANCH_NAME}"
        '''
      }
    }
  }
}
```

---

## v0.2 patterns: json, render, exec

Recipes built on the new commands.

<details>
<summary><strong>Helm-style values overlay → render → kubectl apply</strong></summary>

```sh
# Stack values: base + per-env + inline overrides
pipekit render manifests/deployment.yaml.tpl \
  --values values/base.yaml \
  --values values/prod.yaml \
  --set image.tag=$(pipekit time now --format tag) \
  --output /tmp/deployment.yaml

kubectl apply -f /tmp/deployment.yaml
```

</details>

<details>
<summary><strong>Pull config from helm chart, mutate, save</strong></summary>

```sh
# Bump only the .image.tag in values.yaml without touching anything else
pipekit yaml set chart/values.yaml --path '.image.tag' --value 'v1.2.3' --in-place

# Or: deep-merge a per-env overlay
pipekit yaml merge chart/values.yaml chart/values.prod.yaml --output /tmp/merged.yaml
```

</details>

<details>
<summary><strong>Render a step-summary table from a JSON manifest</strong></summary>

```sh
pipekit json table services.json --columns name,version,status \
  | pipekit summary section --title "Deployed services" --to-github-summary
```

</details>

<details>
<summary><strong>Wrap a flaky deploy in retry + mask + tee in one verb</strong></summary>

```sh
pipekit exec \
  --attempts 5 --backoff --jitter \
  --timeout 5m --max-elapsed 20m \
  --mask-preset github,aws \
  --tee deploy.log \
  --retry-on-stderr "(rate limit|connection reset|EOF)" \
  -- ./scripts/deploy.sh
```

</details>

<details>
<summary><strong>Database URL → connection env vars</strong></summary>

```sh
pipekit url parse "$DATABASE_URL" --prefix DB_ --to-github
# Now DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_PATH (etc) are set
pipekit assert env-exists DB_HOST DB_USER DB_PASSWORD
```

</details>

<details>
<summary><strong>Test sharding with matrix shard</strong></summary>

```yaml
# .github/workflows/test.yml
jobs:
  test:
    strategy:
      matrix:
        shard: [0, 1, 2, 3]
    steps:
      - run: |
          go test -list . ./... | tail -n +2 > tests.txt
          pipekit matrix shard --total 4 --index ${{ matrix.shard }} \
            --from-stdin-lines < tests.txt > my-shard.txt
          xargs -a my-shard.txt go test -run
```

</details>

---

## Combining commands

The real value comes from chaining a few commands together. A few patterns:

<details>
<summary><strong>Resolve config → mask secrets → assert → deploy → notify</strong></summary>

```sh
pipekit config resolve envs.json --env prod --uppercase-keys --to-github
pipekit mask env --env-match "*_TOKEN,*_SECRET" --github
pipekit assert env-exists DEPLOY_TOKEN CLUSTER_NAME
pipekit retry run --attempts 3 --delay 30s -- ./deploy.sh
pipekit notify slack --status success --title "Deploy succeeded"
```

</details>

<details>
<summary><strong>Compute version → tag → build cache key → publish</strong></summary>

```sh
NEXT=$(pipekit version next --source package.json)
pipekit assert semver "$NEXT"
KEY=$(pipekit cache-key composite "$(uname -s)" "$NEXT" "$(pipekit transform hash --file package-lock.json)")
echo "Building $NEXT with cache key $KEY"
```

</details>

<details>
<summary><strong>Build a matrix from changed dirs only</strong></summary>

```sh
pipekit diff dirs --base origin/main \
  | jq -R -s 'split("\n") | map(select(length > 0))' \
  | pipekit matrix from-json --key service \
  | tee matrix.json
```

</details>

<details>
<summary><strong>Release artifacts → checksums → manifest → notes</strong></summary>

```sh
make build-all

pipekit artifact assert "dist/pipekit-*"
pipekit checksum files dist/pipekit-* --output dist/checksums.txt
pipekit checksum verify dist/checksums.txt
pipekit artifact manifest "dist/pipekit-*" "dist/checksums.txt" \
  --pretty --output dist/artifacts.json

pipekit git sha --short --to-github-output git_sha
pipekit git ref --slug --to-github-output ref_slug
pipekit changelog since-tag --conventional --output RELEASE_NOTES.md
```

</details>

---

**See also:** [Commands](COMMANDS.md) · [Requirements](REQUIREMENTS.md) · [Install](INSTALL.md) · [← README](../README.md)
