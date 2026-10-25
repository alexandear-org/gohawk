package examples

import (
	"context"
	"os/exec"
)

// #region process
func run(ctx context.Context, wait bool) error {
	command := exec.CommandContext(ctx, "worker")
	if err := command.Start(); err != nil {
		return err
	}
	if wait {
		return command.Wait()
	}
	return nil
}

// #endregion process

// #region processfixed
func runSafely(ctx context.Context) error {
	command := exec.CommandContext(ctx, "worker")
	return command.Run()
}

// #endregion processfixed
