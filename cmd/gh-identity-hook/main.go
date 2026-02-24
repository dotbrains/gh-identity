package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/dotbrains/gh-identity/internal/hook"
)

func main() {
	shellFlag := flag.String("shell", "", "Shell type: fish, bash, zsh")
	flag.Parse()

	shell := hook.ShellType(strings.ToLower(*shellFlag))
	if shell == "" {
		// Try to detect from SHELL env.
		shell = detectShell()
	}

	dir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "gh-identity-hook: %v\n", err)
		os.Exit(1)
	}

	output, err := hook.Resolve(dir, shell)
	if err != nil {
		// Silently fail — the hook should not break the user's shell.
		fmt.Fprintf(os.Stderr, "gh-identity-hook: %v\n", err)
		os.Exit(0)
	}

	fmt.Print(output)
}

func detectShell() hook.ShellType {
	shellPath := os.Getenv("SHELL")
	if strings.HasSuffix(shellPath, "/fish") {
		return hook.Fish
	}
	if strings.HasSuffix(shellPath, "/zsh") {
		return hook.Zsh
	}
	return hook.Bash
}
