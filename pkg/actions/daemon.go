package actions

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/urfave/cli/v2"
)

// Daemon blocks and triggers sync on a fixed interval.
func Daemon(c *cli.Context) error {
	interval := c.Duration("interval")
	if interval <= 0 {
		return errors.New("interval must be greater than zero")
	}

	nonInteractive := true
	if c.IsSet("non-interactive") || c.IsSet("quiet") {
		nonInteractive = isNonInteractive(c)
	}

	opts := syncOptions{
		UploadPath:     c.String("path"),
		SafeMode:       c.Bool("safe-mode"),
		NonInteractive: nonInteractive,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	fmt.Printf("Starting ssh-sync daemon with interval %s\n", interval)

	for {
		if err := runSync(opts); err != nil {
			fmt.Fprintf(os.Stderr, "sync run failed: %v\n", err)
		}
		select {
		case <-ctx.Done():
			fmt.Fprintln(os.Stderr, "Shutting down daemon.")
			return nil
		case <-ticker.C:
		}
	}
}
