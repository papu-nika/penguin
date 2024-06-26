/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"errors"
	"os"
	"time"

	penguin "github.com/papu-nika/penguin/action"
	"github.com/spf13/cobra"
)

var (
	// Used for flags.
	cfgFile  string
	interval time.Duration
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "penguin [host1] [host2] ...",
	Short: "Ping multiple hosts and display the results in a table",
	Long:  `Ping multiple hosts and display the results in a table.`,

	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return errors.New("no hosts")
		}

		m, err := penguin.InitialModel(args, interval)
		if err != nil {
			return err
		}
		return m.Run()
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().DurationVarP(&interval, "interval", "i", 1*time.Second, "interval between pings")
}
