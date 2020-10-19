package main

import (
	"os"

	"github.com/spf13/cobra"
)

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sftpd",
		Short: "start a sftpd server",
	}

	out := cmd.OutOrStderr()
	cmd.AddCommand(
		newForegroundCmd(out),
	)
	return cmd
}

func main() {
	cmd := newRootCmd()
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
