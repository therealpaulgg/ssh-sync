package actions

import (
	"flag"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v2"
)

func TestSyncRunsUploadThenDownload(t *testing.T) {
	steps := []string{}
	deps := syncDeps{
		checkSetup: func() (bool, error) { return true, nil },
		upload: func(*cli.Context) error {
			steps = append(steps, "upload")
			return nil
		},
		download: func(*cli.Context) error {
			steps = append(steps, "download")
			return nil
		},
	}

	err := syncWithDeps(newSyncTestContext(nil), deps)
	assert.NoError(t, err)
	assert.Equal(t, []string{"upload", "download"}, steps)
}

func TestSyncStopsWhenNotSetup(t *testing.T) {
	deps := syncDeps{
		checkSetup: func() (bool, error) { return false, nil },
		upload:     func(*cli.Context) error { t.Fatalf("upload should not run"); return nil },
		download:   func(*cli.Context) error { t.Fatalf("download should not run"); return nil },
	}

	err := syncWithDeps(newSyncTestContext(nil), deps)
	assert.NoError(t, err)
}

func newSyncTestContext(args []string) *cli.Context {
	set := flag.NewFlagSet("test", 0)
	set.Bool("safe-mode", false, "")
	set.String("path", "", "")
	_ = set.Parse(args)
	return cli.NewContext(&cli.App{}, set, nil)
}
