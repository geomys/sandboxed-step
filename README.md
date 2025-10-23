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
  - overlayed access to GITHUB_WORKSPACE (changes do not persist)
  - GITHUB_WORKSPACE working directory
  - host network access
  - allow-listed environment variables
  - same user as the GitHub Actions runner, with sudo access

All tags use GitHub's Immutable Releases, so they can't be changed even if this
repository is compromised.

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
