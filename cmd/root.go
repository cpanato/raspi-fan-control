package cmd

import (
	"github.com/spf13/cobra"
)

var (
	pinPort int

	rootCmd = &cobra.Command{
		Use:   "control-fan",
		Short: "Service to control an external fan to cooldown your raspberry pies",
	}
)

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().IntVar(&pinPort, "pin-port", 17, "pin port to use to control the fan")
}
