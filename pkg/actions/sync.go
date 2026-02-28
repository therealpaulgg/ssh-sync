package actions

import "github.com/urfave/cli/v2"

var (
	uploadAction   = Upload
	downloadAction = Download
)

// Sync uploads local keys first, then downloads the server state back to disk.
// Flags:
//
//	--path/-p: forwarded to upload
//	--safe-mode/-s: forwarded to download
//	--no-upload: skip the upload step
//	--no-download: skip the download step
func Sync(c *cli.Context) error {
	if !c.Bool("no-upload") {
		if err := uploadAction(c); err != nil {
			return err
		}
	}

	if !c.Bool("no-download") {
		if err := downloadAction(c); err != nil {
			return err
		}
	}

	return nil
}
