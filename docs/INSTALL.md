# Install

pipekit ships as a single statically-linked binary. No runtime dependencies, no installer, no PATH magic.

## Contents

- [Pre-built binaries](#pre-built-binaries)
  - [Linux / macOS](#linux--macos)
  - [Windows](#windows)
- [From source](#from-source)
- [Verify the install](#verify-the-install)
- [Upgrade](#upgrade)
- [Uninstall](#uninstall)

---

## Pre-built binaries

Releases are at https://github.com/AxeForging/pipekit/releases.

### Linux / macOS

<details open>
<summary><strong>Linux x86_64</strong></summary>

```sh
curl -L https://github.com/AxeForging/pipekit/releases/latest/download/pipekit-linux-amd64.tar.gz | tar xz
chmod +x pipekit-linux-amd64
sudo mv pipekit-linux-amd64 /usr/local/bin/pipekit
```

</details>

<details>
<summary><strong>Linux ARM64</strong></summary>

```sh
curl -L https://github.com/AxeForging/pipekit/releases/latest/download/pipekit-linux-arm64.tar.gz | tar xz
chmod +x pipekit-linux-arm64
sudo mv pipekit-linux-arm64 /usr/local/bin/pipekit
```

</details>

<details>
<summary><strong>macOS (Intel)</strong></summary>

```sh
curl -L https://github.com/AxeForging/pipekit/releases/latest/download/pipekit-darwin-amd64.tar.gz | tar xz
chmod +x pipekit-darwin-amd64
sudo mv pipekit-darwin-amd64 /usr/local/bin/pipekit
```

</details>

<details>
<summary><strong>macOS (Apple Silicon)</strong></summary>

```sh
curl -L https://github.com/AxeForging/pipekit/releases/latest/download/pipekit-darwin-arm64.tar.gz | tar xz
chmod +x pipekit-darwin-arm64
sudo mv pipekit-darwin-arm64 /usr/local/bin/pipekit
```

</details>

### Windows

<details>
<summary><strong>Windows x86_64 (PowerShell)</strong></summary>

```powershell
Invoke-WebRequest -Uri https://github.com/AxeForging/pipekit/releases/latest/download/pipekit-windows-amd64.zip -OutFile pipekit.zip
Expand-Archive -Path pipekit.zip -DestinationPath .
Move-Item -Path pipekit-windows-amd64.exe -Destination pipekit.exe
```

Add the directory containing `pipekit.exe` to your `PATH`, or move it to a directory already on `PATH`.

</details>

---

## From source

Requires Go 1.25 or later.

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

`make build` produces a stripped binary at `dist/pipekit` with version metadata baked in via `-ldflags`.

For multi-platform builds, contributing, and release notes see **[CONTRIBUTING.md](CONTRIBUTING.md)**.

---

## Verify the install

```sh
pipekit --version
pipekit build-info
```

`build-info` shows the version, build time, and git commit baked into the binary.

---

## Upgrade

Re-run the install command for your platform — it always pulls the latest release. If you installed from source, `go install github.com/AxeForging/pipekit@latest` again.

---

## Uninstall

```sh
sudo rm /usr/local/bin/pipekit
```

(Or wherever you put the binary.)

---

**See also:** [Requirements](REQUIREMENTS.md) · [Commands](COMMANDS.md) · [Examples](EXAMPLES.md) · [← README](../README.md)
