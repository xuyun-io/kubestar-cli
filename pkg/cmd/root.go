package cmd

import (
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/xuyun-io/kubestar-cli/pkg/utils"
	"os"
	"strings"
)

var RootCmd = &cobra.Command{
	Use:   "ks",
	Short: "KubeStar CLI",
	Long:  `The KubeStar command line interface.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		printEnvVars()
	},
}

func init() {
	RootCmd.AddCommand(DeployCmd)
}

// Execute is the main function for the Cobra CLI.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		utils.WithError(err).Fatal("Error executing command")
	}
}

func printEnvVars() {
	envs := os.Environ()
	var ksEnvs []string
	for _, env := range envs {
		if strings.HasPrefix(env, "KS_") {
			ksEnvs = append(ksEnvs, env)
		}
	}
	if len(ksEnvs) == 0 {
		return
	}
	green := color.New(color.Bold, color.FgGreen)
	green.Fprintf(os.Stderr, "*******************************\n")
	green.Fprintf(os.Stderr, "* ENV VARS\n")
	for _, env := range ksEnvs {
		green.Fprintf(os.Stderr, "* \t %s\n", env)
	}
	green.Fprintf(os.Stderr, "*******************************\n")
}
