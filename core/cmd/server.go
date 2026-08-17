package cmd

import (
	"github.com/goastian/astiango-hub/core/apps"
	"github.com/goastian/astiango-hub/core/utils"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(serverCmd)
}

var serverCmd = &cobra.Command{
	Use:     "server",
	Aliases: []string{"s"},
	Short:   "Start AstianGO Hub server",
	Long:    `Start AstianGO Hub node server that can serve as API, task scheduler, task runner, etc.`,
	Run: func(cmd *cobra.Command, args []string) {
		// print logo if not pro
		if !utils.IsPro() {
			utils.PrintLogoWithWelcomeInfo()
		}

		// app
		svr := apps.GetServer()

		// start
		apps.Start(svr)
	},
}
