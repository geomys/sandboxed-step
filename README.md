# gVisor Sandboxed Step

A GitHub Action that runs commands in a gVisor sandbox.

## Usage

```yaml
- uses: geomys/sandboxed-step@v1.0.0
  with:
    run: |
      apt-get update && apt-get install -y golang-go
      go get -u && go mod tidy
      go test ./...
```

The commands run in a gVisor sandbox with

  - an Ubuntu 24.04 root filesystem
  - overlayed access to GITHUB_WORKSPACE (changes do not persist by default)
  - GITHUB_WORKSPACE working directory
  - host network access
  - allow-listed environment variables
  - same user as the GitHub Actions runner, with sudo access

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
- `allow-checkout-credentials` (optional, default: `false`): Allow persisted checkout credentials (NOT RECOMMENDED)

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
