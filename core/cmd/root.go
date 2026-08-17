package cmd

import (
	"github.com/spf13/cobra"
)

var (
	// Used for flags.
	cfgFile string

	rootCmd = &cobra.Command{
		Use:   "astiango-hub",
		Short: "CLI tool for AstianGO Hub",
		Long: `The CLI tool is for controlling against AstianGO Hub.
AstianGO Hub is a distributed web crawler and task admin platform
aimed at making web crawling and task management easier.
`,
	}
)

// Execute executes the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "c", "", "Use Custom Config File")
}
