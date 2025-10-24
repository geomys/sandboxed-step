# gVisor Sandboxed Step

A GitHub Action that runs commands in a gVisor sandbox.

## Motivation

Surprisingly enough, GitHub Actions with read-only permissions still receive a
cache write token, so they are not safe to run untrusted code.

Moreover, there is no isolation between steps in a workflow, since they all run
on the same VM with root access. The only alternative is separating workflows
with `workflow_run`, but that has its own limitations and overhead.

This can lead to a false sense of security when running untrusted code in a
workflow or step with read-only permissions.

This Action runs commands in an isolated gVisor sandbox, allowing e.g. running
CI against the latest versions of dependencies without risking being affected by
supply chain attacks.

## Usage

```yaml
- uses: geomys/sandboxed-step@v1.1.1
  with:
    run: |
      go get -u && go mod tidy
      go test ./...
```

The commands run in a gVisor sandbox with

  - a root filesystem similar to ubuntu-24.04 ([`ghcr.io/catthehacker/ubuntu:runner-24.04`](https://github.com/catthehacker/docker_images))
  - overlayed access to GITHUB_WORKSPACE (changes do not persist by default)
  - GITHUB_WORKSPACE working directory
  - host network access
  - allow-listed environment variables
  - same user as the GitHub Actions runner, with sudo access
  - read-only access to tools installed by setup-* actions (via RUNNER_TOOL_CACHE)

Changes to the workspace inside the sandbox can be made to persist on the host
by setting `persist-workspace-changes: 'true'`. This is unlikely to be safe, as
following steps will need to treat the workspace as untrusted.

> [!NOTE]
> This action will detect and fail if the `actions/checkout` Action has
> persisted authentication tokens in GITHUB_WORKSPACE (the default behavior).
>
> To use this action, you must disable credential persistence in your checkout
> step:
>
> ```yaml
> - uses: actions/checkout@v4
>   with:
>     persist-credentials: false
> ```
>
> Alternatively, you can set `allow-checkout-credentials: true` to bypass this
> check, but this is **NOT RECOMMENDED** as it will expose the GitHub token to
> the sandbox.

All tags use GitHub's Immutable Releases, so they can't be changed even if this
repository is compromised.

### Inputs

- `run` (required): Commands to run in the sandbox
- `env` (optional): Additional environment variables to set in the sandbox (one per line, KEY=VALUE format)
- `persist-workspace-changes` (optional, default: `false`): Allow changes to persist on the host
- `disable-network` (optional, default: `false`): Disable network access in the sandbox
- `allow-checkout-credentials` (optional, default: `false`): Allow persisted checkout credentials
- `rootfs-image` (optional): Docker image to use as the root filesystem

### Example for sandboxed Go tests with latest dependencies

```yaml
name: Go tests
on:
  push:
  pull_request:
  schedule: # daily at 09:42 UTC
    - cron: '42 9 * * *'
  workflow_dispatch:
permissions:
  contents: read
jobs:
  test:
    runs-on: ubuntu-latest
    strategy:
      fail-fast: false
      matrix:
        go:
          - { go-version: stable }
          - { go-version: oldstable }
          - { go-version-file: go.mod }
        deps:
          - locked
          - latest
    steps:
      - uses: actions/checkout@v5
        with:
          persist-credentials: false
      - uses: actions/setup-go@v6
        with:
          go-version: ${{ matrix.go.go-version }}
          go-version-file: ${{ matrix.go.go-version-file }}
      - uses: geomys/sandboxed-step@v1.1.1
        with:
          run: |
            if [ "${{ matrix.deps }}" = "latest" ]; then
              go get -u && go mod tidy
            fi
            go test ./...
  staticcheck:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v5
        with:
          persist-credentials: false
      - uses: actions/setup-go@v6
        with:
          go-version: stable
      - uses: geomys/sandboxed-step@v1.1.1
        with:
          run: go run honnef.co/go/tools/cmd/staticcheck@latest ./...
  govulncheck:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v5
        with:
          persist-credentials: false
      - uses: actions/setup-go@v6
        with:
          go-version: stable
      - uses: geomys/sandboxed-step@v1.1.1
        with:
          run: go run golang.org/x/vuln/cmd/govulncheck@latest ./...
```
