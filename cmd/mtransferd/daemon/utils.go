package daemon

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/spf13/cobra"
)

// runEWithClientCtx runs cmd with client context and returns an error.
func runEWithClientCtx(
	fRunWithCtx func(ctx client.Context, cmd *cobra.Command, args []string) error,
) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		clientCtx, err := client.GetClientQueryContext(cmd)
		if err != nil {
			return err
		}

		return fRunWithCtx(clientCtx, cmd, args)
	}
}

// getSubCommand returns the command if it finds, otherwise it returns nil
func getSubCommand(cmd *cobra.Command, commandName string) *cobra.Command {
	for _, c := range cmd.Commands() {
		if !strings.EqualFold(c.Name(), commandName) {
			continue
		}

		return c
	}

	return nil
}

func makeDirectory(dir string) error {
	err := os.MkdirAll(dir, 0700)
	if err != nil {
		// Show a nicer error message if it's because a symlink
		// is linked to a directory that does not exist
		// (probably because it's not mounted).
		if e := new(os.PathError); errors.As(err, &e) && os.IsExist(err) {
			link, lerr := os.Readlink(e.Path)
			if lerr == nil {
				str := "is symlink %s -> %s mounted?"
				err = fmt.Errorf(str, e.Path, link)
			}
		}

		return fmt.Errorf("failed to create dir %s: %w", dir, err)
	}

	return nil
}


// fileExists reports whether the named file or directory exists.
// This function is taken from https://github.com/btcsuite/btcd
func fileExists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}

	return true
}