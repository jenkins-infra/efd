package cmd

import (
	"fmt"
	"os"

	"github.com/jenkins-infra/efd/pkg"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:   "efd",
		Short: "efd is a small utility to get the main email address of every members of a Discourse group",
		Long: `EFD stands for "Email From Discourse".
It's is a small utility to get the main email address of every members of a Discourse group".`,
		Run: func(cmd *cobra.Command, args []string) {
			pkg.Execute(Group, ApiUsername, ApiKey, ApiEndpoint)
		},
	}

	ApiEndpoint string
	ApiUsername string
	ApiKey      string
	Group       string
	verbose     bool
)

func init() {
	rootCmd.PersistentFlags().StringVarP(&ApiEndpoint, "apiEndpoint", "e", "community.jenkins.io", "Specify Api Endpoint")

	rootCmd.PersistentFlags().StringVarP(&ApiUsername, "apiUsername", "u", "", "Specify Api Username")
	err := rootCmd.MarkPersistentFlagRequired("apiUsername")
	if err != nil {
		logrus.Errorln(err)

	}

	rootCmd.PersistentFlags().StringVarP(&ApiKey, "apiKey", "k", "", "Specify Api Key")
	err = rootCmd.MarkPersistentFlagRequired("apiKey")
	if err != nil {
		logrus.Errorln(err)
	}

	rootCmd.Flags().StringVarP(&Group, "group", "g", "trust_level_0", "Specify group name to fetch users' email address from")

	rootCmd.PersistentFlags().BoolVarP(&verbose, "debug", "", false, "Debug Output")

	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		if verbose {
			logrus.SetLevel(logrus.DebugLevel)
		}
	}
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
