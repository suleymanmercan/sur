package cmd

import (
	"context"
	"os/exec"
)

// execCommandContext is a tiny shim so tests can override exec.CommandContext
// if they ever need to. Today it just delegates.
var execCommandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, name, args...)
}
