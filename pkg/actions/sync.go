package actions

import (
	"fmt"

	"github.com/urfave/cli/v2"
)

type syncOptions struct {
	UploadPath     string
	SafeMode       bool
	NonInteractive bool
}

// Sync runs an upload followed by a download to reconcile local and remote
// state. When running non-interactively it skips any conflicting changes.
func Sync(c *cli.Context) error {
	opts := syncOptions{
		UploadPath:     c.String("path"),
		SafeMode:       c.Bool("safe-mode"),
		NonInteractive: isNonInteractive(c),
	}
	if err := runSync(opts); err != nil {
		return err
	}
	fmt.Println("Sync completed.")
	return nil
}

func runSync(opts syncOptions) error {
	if err := runUpload(uploadOptions{
		Path:           opts.UploadPath,
		NonInteractive: opts.NonInteractive,
	}); err != nil {
		return err
	}
	if err := runDownload(downloadOptions{
		SafeMode:       opts.SafeMode,
		NonInteractive: opts.NonInteractive,
	}); err != nil {
		return err
	}
	return nil
}
