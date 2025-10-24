# Claude Session Notes - gVisor Sandboxed Step

## Project Overview
Creating a GitHub Action that runs commands in a gVisor sandbox, similar to the built-in `run:` but with security isolation.

This is a small project, you can read all of action.yml, README.md, generate-config.go, and .github/workflows/test.yml before doing any work.

## User Preferences & Design Decisions

### Architecture Choices
- **No Docker in the Action**: gVisor provides sufficient isolation, Docker would be unnecessary layering
- **Composite Action**: Not a Docker-based action, runs directly on the runner
- **Pre-built binaries**: Include runsc and generate-config binaries (for Linux amd64) rather than downloading at runtime
- **Dynamic rootfs**: Download Docker images at runtime to use as rootfs (default: ghcr.io/catthehacker/ubuntu:runner-24.04)
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
   - The catthehacker image includes runner user with passwordless sudo pre-configured
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
```

### Docker Image Rootfs
- **Default image**: ghcr.io/catthehacker/ubuntu:runner-24.04
- **Features**: Full GitHub Actions runner environment with all tools pre-installed
- **User**: Pre-configured runner user (UID 1001) with passwordless sudo
- **Customizable**: Can specify any Docker image via `rootfs-image` input
- **Downloaded at runtime**: Uses Docker to pull and export the image filesystem

### Maintenance Scripts
- `update-runsc.sh`: Downloads latest runsc from Google
- `build-generate-config.sh`: Cross-compiles generate-config for Linux amd64

### Key Technical Details
- OCI runtime specification for container configuration
- Uses gVisor's `runsc run` (not `runsc do`)
- Proper OCI bundle with config.json and rootfs directory
- DNS configuration copied from host for network access
- **Platform**: Default (systrap) - faster than ptrace
- **Binary location**: Run runsc directly from action path, don't copy to /usr/local/bin
- **Execution**: Use `sudo runsc` (rootless doesn't work with networking), but processes inside run as runner user
- **Image extraction**: Uses Docker to pull and export the container filesystem

### Testing Philosophy
- Tests should verify security properties (overlay isolation)
- Tests should check that modifications don't leak to host
- Include tests for user environment matching
- Test both success and failure cases
- **CRITICAL**: Test that failures propagate (script failures MUST fail the action)
- Test missing commands fail properly
- Network connectivity tested via `apt-get update` (available in catthehacker image)

### What NOT to Do
- Don't use Docker inside the action
- Don't download binaries at runtime (include them)
- Don't create minimal/busybox fallbacks
- Don't inherit HOME/PATH from host environment
- Don't use bash for JSON generation
- Don't pollute GITHUB_ENV
- Don't add fallback values for "reliability" - fail explicitly
- **NEVER add fallbacks in tests** - tests should fail clearly when something is wrong, not try alternate approaches

### Important Maintenance Tasks
- **Always rebuild generate-config**: When modifying generate-config.go, run `./build-generate-config.sh` to rebuild the binary (actually run it yourself)

### Lessons Learned / Important Notes
- **Simplicity over complexity**: When given a choice, prefer simple solutions (e.g., just check if file is empty with `-s`, don't overcomplicate with whitespace checks)
- **Pass files as arguments, not environment variables**: When passing data to programs, especially with special characters, write to a file and pass the file path as an argument rather than using environment variables. Don't create unnecessary environment variables like SANDBOXED_ADDITIONAL_ENV_FILE - just pass the file path directly as a command argument
- **Delete TODO.md when done**: Remove the TODO file completely when all tasks are complete, don't just mark items as done
- **apt-get update exit codes are unreliable**: apt-get update returns 0 even when network fails (uses cached data), check for error messages in output instead
- **GitHub Actions tool cache**: Setup-* actions install tools in RUNNER_TOOL_CACHE (typically /opt/hostedtoolcache), mount this read-only to expose tools in sandbox. Setup-go installs to paths like `/opt/hostedtoolcache/go/1.25.3/x64/bin/go` and adds them to PATH
- **Docker image benefits**: Using catthehacker images provides a full GitHub Actions environment with all standard tools and ca-certificates pre-installed

### Future Considerations
- Currently Linux x86_64 only
- Could potentially support ARM64 runners
- Network isolation could be added as an option (already implemented with disable-network input)
- Different base images could be used for different language environments
