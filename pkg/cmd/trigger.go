package cmd

import (
	"fmt"
	"os"

"github.com/bitnami-labs/kubewatch/config"
c "github.com/bitnami-labs/kubewatch/pkg/client"
"github.com/sirupsen/logrus"
"github.com/spf13/cobra"
"github.com/spf13/viper"
)

var cfgFile string

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "trigger",
	Short: "A kubernetes event watcher and trigger",
	Long: `
Kube-Trigger: A watcher and trigger for Kubernetes Object Events.

It watches for CRD and trigger KubeVela application updates.
`,

	Run: func(cmd *cobra.Command, args []string) {
		config := &config.Config{}
		if err := config.Load(); err != nil {
			logrus.Fatal(err)
		}
		config.CheckMissingResourceEnvvars()
		c.Run(config)
	},
}

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Disable Help subcommand
	RootCmd.SetHelpCommand(&cobra.Command{
		Use:    "no-help",
		Hidden: true,
	})
	//RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.kubewatch.yaml)")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" { // enable ability to specify config file via flag
		viper.SetConfigFile(cfgFile)
	}

	viper.SetConfigName(kubewatchConfigFile) // name of config file (without extension)
	viper.AddConfigPath("$HOME")             // adding home directory as first search path
	viper.AutomaticEnv()                     // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

