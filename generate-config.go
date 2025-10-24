package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// OCIConfig represents the OCI runtime specification
type OCIConfig struct {
	OCIVersion string  `json:"ociVersion"`
	Process    Process `json:"process"`
	Root       Root    `json:"root"`
	Hostname   string  `json:"hostname"`
	Mounts     []Mount `json:"mounts"`
	Linux      *Linux  `json:"linux,omitempty"`
}

type Process struct {
	Terminal        bool          `json:"terminal"`
	User            User          `json:"user"`
	Args            []string      `json:"args"`
	Env             []string      `json:"env"`
	Cwd             string        `json:"cwd"`
	Capabilities    *Capabilities `json:"capabilities,omitempty"`
	NoNewPrivileges bool          `json:"noNewPrivileges"`
}

type User struct {
	UID int `json:"uid"`
	GID int `json:"gid"`
}

type Capabilities struct {
	Bounding  []string `json:"bounding"`
	Effective []string `json:"effective"`
	Permitted []string `json:"permitted"`
}

type Root struct {
	Path     string `json:"path"`
	Readonly bool   `json:"readonly"`
}

type Mount struct {
	Destination string   `json:"destination"`
	Type        string   `json:"type"`
	Source      string   `json:"source"`
	Options     []string `json:"options,omitempty"`
}

type Linux struct {
	Namespaces []Namespace `json:"namespaces"`
}

type Namespace struct {
	Type string `json:"type"`
}

var envAllowlist = []string{
	"USER",
	"TERM",
	"LANG",
	"LC_ALL",
	"LC_CTYPE",
	"TZ",
	"GITHUB_WORKSPACE",
	"GITHUB_ACTION",
	"GITHUB_ACTIONS",
	"GITHUB_ACTOR",
	"GITHUB_ACTOR_ID",
	"GITHUB_API_URL",
	"GITHUB_BASE_REF",
	"GITHUB_EVENT_NAME",
	"GITHUB_GRAPHQL_URL",
	"GITHUB_HEAD_REF",
	"GITHUB_JOB",
	"GITHUB_REF",
	"GITHUB_REF_NAME",
	"GITHUB_REF_PROTECTED",
	"GITHUB_REF_TYPE",
	"GITHUB_REPOSITORY",
	"GITHUB_REPOSITORY_ID",
	"GITHUB_REPOSITORY_OWNER",
	"GITHUB_REPOSITORY_OWNER_ID",
	"GITHUB_RETENTION_DAYS",
	"GITHUB_RUN_ATTEMPT",
	"GITHUB_RUN_ID",
	"GITHUB_RUN_NUMBER",
	"GITHUB_SERVER_URL",
	"GITHUB_SHA",
	"GITHUB_WORKFLOW",
	"GITHUB_WORKFLOW_REF",
	"GITHUB_WORKFLOW_SHA",
	"RUNNER_ARCH",
	"RUNNER_DEBUG",
	"RUNNER_NAME",
	"RUNNER_OS",
	"RUNNER_TOOL_CACHE",
	"CI",
	// Test-related (for our workflow tests)
	"HOSTNAME_FOR_TEST",
}

func main() {
	workspace := os.Getenv("GITHUB_WORKSPACE")
	if workspace == "" {
		fmt.Fprintf(os.Stderr, "Error: GITHUB_WORKSPACE environment variable is required\n")
		os.Exit(1)
	}

	// Optional: read additional env file from command line argument
	var additionalEnvFile string
	if len(os.Args) > 1 {
		additionalEnvFile = os.Args[1]
	}

	// Get hostname from host system
	hostname, err := os.Hostname()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to get hostname: %v\n", err)
		os.Exit(1)
	}

	// Get current user information
	uid := os.Getuid()
	gid := os.Getgid()
	username := os.Getenv("USER")
	if username == "" {
		fmt.Fprintf(os.Stderr, "Error: USER environment variable is not set\n")
		os.Exit(1)
	}

	// Build environment variables list
	env := buildEnvList(additionalEnvFile)

	// Define capabilities
	caps := []string{
		"CAP_CHOWN",
		"CAP_DAC_OVERRIDE",
		"CAP_FSETID",
		"CAP_FOWNER",
		"CAP_MKNOD",
		"CAP_NET_RAW",
		"CAP_SETGID",
		"CAP_SETUID",
		"CAP_SETFCAP",
		"CAP_SETPCAP",
		"CAP_SYS_CHROOT",
		"CAP_KILL",
	}

	// Build mounts list
	mounts := buildMountsList(workspace)

	// Create the OCI config
	config := OCIConfig{
		OCIVersion: "1.0.0",
		Process: Process{
			Terminal: false,
			User: User{
				UID: uid,
				GID: gid,
			},
			Args: []string{
				"/bin/bash",
				"/entrypoint.sh",
			},
			Env: env,
			Cwd: workspace,
			Capabilities: &Capabilities{
				Bounding:  caps,
				Effective: caps,
				Permitted: caps,
			},
			NoNewPrivileges: false,
		},
		Root: Root{
			Path:     "rootfs",
			Readonly: false,
		},
		Hostname: hostname,
		Mounts:   mounts,
		Linux: &Linux{
			Namespaces: []Namespace{
				{Type: "pid"},
				{Type: "mount"},
				{Type: "ipc"},
				{Type: "uts"},
			},
		},
	}

	// Output JSON
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(config); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
		os.Exit(1)
	}
}

func buildEnvList(additionalEnvFile string) []string {
	env := []string{}

	for _, name := range envAllowlist {
		if value := os.Getenv(name); value != "" {
			env = append(env, fmt.Sprintf("%s=%s", name, value))
		}
	}

	// Build PATH: start with standard Ubuntu paths
	pathComponents := []string{
		"/usr/local/sbin",
		"/usr/local/bin",
		"/usr/sbin",
		"/usr/bin",
		"/sbin",
		"/bin",
	}

	// Add any PATH components from host that are rooted in RUNNER_TOOL_CACHE
	if toolCache := os.Getenv("RUNNER_TOOL_CACHE"); toolCache != "" {
		if hostPath := os.Getenv("PATH"); hostPath != "" {
			for _, p := range strings.Split(hostPath, ":") {
				if strings.HasPrefix(p, toolCache) {
					pathComponents = append(pathComponents, p)
				}
			}
		}
	}

	env = append(env, "PATH="+strings.Join(pathComponents, ":"))
	env = append(env, fmt.Sprintf("HOME=%s", "/home/"+os.Getenv("USER")))

	// Add any additional user-specified environment variables from file
	if additionalEnvFile != "" {
		content, err := os.ReadFile(additionalEnvFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: Failed to read environment file %s: %v\n", additionalEnvFile, err)
			os.Exit(1)
		}
		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			// Skip empty lines and validate format
			if line != "" && strings.Contains(line, "=") {
				env = append(env, line)
			}
		}
	}

	return env
}

func buildMountsList(workspace string) []Mount {
	mounts := []Mount{
		{
			Destination: workspace,
			Type:        "bind",
			Source:      workspace,
			Options:     []string{"rbind", "rw"},
		},
		{
			Destination: "/proc",
			Type:        "proc",
			Source:      "proc",
		},
		{
			Destination: "/dev",
			Type:        "tmpfs",
			Source:      "tmpfs",
			Options:     []string{"nosuid", "strictatime", "mode=755", "size=65536k"},
		},
		{
			Destination: "/dev/pts",
			Type:        "devpts",
			Source:      "devpts",
			Options:     []string{"nosuid", "noexec", "newinstance", "ptmxmode=0666", "mode=0620"},
		},
		{
			Destination: "/dev/shm",
			Type:        "tmpfs",
			Source:      "shm",
			Options:     []string{"nosuid", "noexec", "nodev", "mode=1777", "size=65536k"},
		},
		{
			Destination: "/sys",
			Type:        "sysfs",
			Source:      "sysfs",
			Options:     []string{"nosuid", "noexec", "nodev", "ro"},
		},
		{
			Destination: "/tmp",
			Type:        "tmpfs",
			Source:      "tmpfs",
			Options:     []string{"nosuid", "nodev", "mode=1777"},
		},
	}

	// Add RUNNER_TOOL_CACHE mount if it exists
	if toolCache := os.Getenv("RUNNER_TOOL_CACHE"); toolCache != "" {
		// Check if directory exists
		if info, err := os.Stat(toolCache); err == nil && info.IsDir() {
			mounts = append(mounts, Mount{
				Destination: toolCache,
				Type:        "bind",
				Source:      toolCache,
				Options:     []string{"rbind", "ro"}, // Read-only mount
			})
		}
	}

	return mounts
}
