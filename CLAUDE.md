# Claude Session Notes - gVisor Sandboxed Step

## Project Overview
Creating a GitHub Action that runs commands in a gVisor sandbox, similar to the built-in `run:` but with security isolation.

This is a small project, you can read all of action.yml, README.md, generate-config.go, and .github/workflows/test.yml before doing any work.

## User Preferences & Design Decisions

### Architecture Choices
- **No Docker in the Action**: gVisor provides sufficient isolation, Docker would be unnecessary layering
- **Composite Action**: Not a Docker-based action, runs directly on the runner
- **Pre-built binaries**: Include runsc and generate-config binaries (for Linux amd64) rather than downloading at runtime
- **Pre-included rootfs**: Ship Ubuntu 24.04 rootfs (~28MB) with the action, don't create minimal fallbacks
- **Single step execution**: Everything runs in one composite step, no separate setup step needed

### Security & Isolation Properties
1. **Overlay filesystem with read-only semantics**:
   - Use `--overlay2=all:self` to apply overlay to all mounts
   - Use `--file-access-mounts=exclusive` (gVisor assumes no external changes)
   - Workspace modifications stay in overlay, don't affect host
   - Tests verify: files created/modified/deleted in sandbox don't affect host

2. **Network configuration**:
   - Network is ENABLED (`--network=host`)
   - User explicitly doesn't care about network sandboxing
   - Allows apt-get and package installation

3. **User environment**:
   - Run as same user as host (typically `runner` with UID 1001 on GitHub Actions)
   - NOT as root inside the sandbox
   - Create matching user in rootfs with passwordless sudo
   - **Important**: Use `sudo` to run runsc (rootless mode doesn't work well with networking)
   - Container processes still run as the runner user, not root (configured via OCI config)

4. **Workspace mounting**:
   - Mount at SAME path as host (e.g., `/home/runner/work/repo/repo`)
   - Set as working directory
   - Read access to existing files, writes go to overlay

### Environment Variables
- **Allowlist approach**: Only pass through explicitly listed variables
- **HOME and PATH**: Always set based on sandbox, NOT inherited from host
  - PATH: `/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin`
  - HOME: `/home/{username}`
- **No fallbacks**: Fail explicitly if required variables missing (USER, GITHUB_WORKSPACE)
- **Hostname**: Match the host's hostname

### Implementation Details
1. **Go for config generation**: Created generate-config.go instead of fragile bash JSON manipulation
2. **No GITHUB_ENV pollution**: Everything runs in single step, uses local variables
3. **Error handling philosophy**: Fail fast with clear errors, no silent fallbacks
4. **Testing**: Comprehensive tests that verify overlay isolation works correctly
5. **CRITICAL Error Propagation**:
   - Use `set -euo pipefail` in both main script and entrypoint
   - Explicitly capture and check runsc exit code
   - Clean up even on failure, then exit with same code
   - Test that failures propagate (missing commands, exit codes)

### File Structure
```
action.yml              # Composite action definition
generate-config         # Pre-built Linux amd64 binary for OCI config
generate-config.go      # Source for config generator
runsc                   # Pre-built gVisor runtime (64MB)
ubuntu-24.04-rootfs.tar.gz  # Ubuntu rootfs (28MB)
```

### Ubuntu Rootfs Contents
The minimal Ubuntu 24.04 rootfs includes:
- **Package management**: apt, apt-get, dpkg
- **Shell & basics**: bash, cat, ls, cp, mv, rm, etc.
- **Text processing**: sed, awk, grep, sort, etc.
- **System tools**: ps, mount, service, systemctl stubs
- **NOT included by default**: curl, wget, ping, nc (can be installed with apt-get)

### Maintenance Scripts
- `update-runsc.sh`: Downloads latest runsc from Google
- `download-ubuntu-rootfs.sh`: Uses Docker to export Ubuntu 24.04 rootfs
- `build-generate-config.sh`: Cross-compiles generate-config for Linux amd64

### Key Technical Details
- OCI runtime specification for container configuration
- Uses gVisor's `runsc run` (not `runsc do`)
- Proper OCI bundle with config.json and rootfs directory
- DNS configuration copied from host for network access
- User/group creation in rootfs before container start
- **Platform**: Default (systrap) - faster than ptrace
- **Binary location**: Run runsc directly from action path, don't copy to /usr/local/bin
- **Execution**: Use `sudo runsc` (rootless doesn't work with networking), but processes inside run as runner user

### Testing Philosophy
- Tests should verify security properties (overlay isolation)
- Tests should check that modifications don't leak to host
- Include tests for user environment matching
- Test both success and failure cases
- **CRITICAL**: Test that failures propagate (script failures MUST fail the action)
- Test missing commands fail properly
- Network connectivity tested via `apt-get update` (apt-get is available in minimal Ubuntu rootfs)

### What NOT to Do
- Don't use Docker inside the action
- Don't download binaries at runtime (include them)
- Don't create minimal/busybox fallbacks
- Don't inherit HOME/PATH from host environment
- Don't use bash for JSON generation
- Don't pollute GITHUB_ENV
- Don't add fallback values for "reliability" - fail explicitly

### Future Considerations
- Currently Linux x86_64 only
- Could potentially support ARM64 runners
- Rootfs size could be optimized if needed
- Network isolation could be added as an option
