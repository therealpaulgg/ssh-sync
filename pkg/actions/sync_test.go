package actions

import (
	"flag"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v2"
)

func TestSyncRunsUploadThenDownloadByDefault(t *testing.T) {
	originalUpload := uploadAction
	originalDownload := downloadAction
	defer func() {
		uploadAction = originalUpload
		downloadAction = originalDownload
	}()

	steps := []string{}
	uploadAction = func(c *cli.Context) error {
		steps = append(steps, "upload")
		return nil
	}
	downloadAction = func(c *cli.Context) error {
		steps = append(steps, "download")
		return nil
	}

	ctx := newSyncTestContext(nil)

	err := Sync(ctx)
	assert.NoError(t, err)
	assert.Equal(t, []string{"upload", "download"}, steps)
}

func TestSyncSkipsUploadWhenRequested(t *testing.T) {
	originalUpload := uploadAction
	originalDownload := downloadAction
	defer func() {
		uploadAction = originalUpload
		downloadAction = originalDownload
	}()

	uploadCalled := false
	downloadCalled := false
	uploadAction = func(c *cli.Context) error {
		uploadCalled = true
		return nil
	}
	downloadAction = func(c *cli.Context) error {
		downloadCalled = true
		return nil
	}

	ctx := newSyncTestContext([]string{"--no-upload"})

	err := Sync(ctx)
	assert.NoError(t, err)
	assert.False(t, uploadCalled)
	assert.True(t, downloadCalled)
}

func TestSyncSkipsDownloadWhenRequested(t *testing.T) {
	originalUpload := uploadAction
	originalDownload := downloadAction
	defer func() {
		uploadAction = originalUpload
		downloadAction = originalDownload
	}()

	uploadCalled := false
	downloadCalled := false
	uploadAction = func(c *cli.Context) error {
		uploadCalled = true
		return nil
	}
	downloadAction = func(c *cli.Context) error {
		downloadCalled = true
		return nil
	}

	ctx := newSyncTestContext([]string{"--no-download"})

	err := Sync(ctx)
	assert.NoError(t, err)
	assert.True(t, uploadCalled)
	assert.False(t, downloadCalled)
}

func TestSyncSkipsBothWhenRequested(t *testing.T) {
	originalUpload := uploadAction
	originalDownload := downloadAction
	defer func() {
		uploadAction = originalUpload
		downloadAction = originalDownload
	}()

	uploadCalled := false
	downloadCalled := false
	uploadAction = func(c *cli.Context) error {
		uploadCalled = true
		return nil
	}
	downloadAction = func(c *cli.Context) error {
		downloadCalled = true
		return nil
	}

	ctx := newSyncTestContext([]string{"--no-upload", "--no-download"})

	err := Sync(ctx)
	assert.NoError(t, err)
	assert.False(t, uploadCalled)
	assert.False(t, downloadCalled)
}

func newSyncTestContext(args []string) *cli.Context {
	set := flag.NewFlagSet("test", 0)
	set.Bool("safe-mode", false, "")
	set.String("path", "", "")
	set.Bool("no-upload", false, "")
	set.Bool("no-download", false, "")
	_ = set.Parse(args)
	return cli.NewContext(&cli.App{}, set, nil)
}
