package daemon

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/babylonlabs-io/mtransfer/types"
	"github.com/spf13/cobra"
)

func CommandInit() *cobra.Command {
	var cmd = &cobra.Command{
		Use:     "init",
		Short:   "Initialize an mtransfer home directory.",
		Long:    `Creates a new mtransfer home directory`,
		Example: `mtransferd init --home /home/user/.mtransfer`,
		Args:    cobra.NoArgs,
		RunE:    types.RunEWithCtx(runInitCmd),
	}

	return cmd
}

func runInitCmd(ctx types.Context, cmd *cobra.Command, _ []string) error {
	homePath, err := filepath.Abs(ctx.ClientCtx.HomeDir)
	if err != nil {
		return err
	}

	homePath = cleanAndExpandPath(homePath)

	if fileExists(homePath) {
		return fmt.Errorf("home path %s already exists", homePath)
	}

	if err := makeDirectory(homePath); err != nil {
		return err
	}

	return nil
}

// cleanAndExpandPath expands environment variables and leading ~ in the
// passed path, cleans the result, and returns it.
// This function is taken from https://github.com/btcsuite/btcd
func cleanAndExpandPath(path string) string {
	if path == "" {
		return ""
	}

	// Expand initial ~ to OS specific home directory.
	if strings.HasPrefix(path, "~") {
		var homeDir string
		u, err := user.Current()
		if err == nil {
			homeDir = u.HomeDir
		} else {
			homeDir = os.Getenv("HOME")
		}

		path = strings.Replace(path, "~", homeDir, 1)
	}

	// NOTE: The os.ExpandEnv doesn't work with Windows-style %VARIABLE%,
	// but the variables can still be expanded via POSIX-style $VARIABLE.
	return filepath.Clean(os.ExpandEnv(path))
}
