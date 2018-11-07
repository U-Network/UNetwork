package testing

import (
	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:   "UNetwork",
		Short: "This is the UNetwork stress test application",
	}
)

// Execute executes the root command.
func Execute() {
	rootCmd.Execute()
}

func AddCmd(cmd *cobra.Command) {
	rootCmd.AddCommand(cmd)
}
