package cmd

import (
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:   "juno",
	Short: "",
	Long:  ``,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		lvl, err := log.ParseLevel(viper.GetString("loglevel"))
		if err != nil {
			log.Fatal(err)
		}
		log.SetLevel(lvl)
	},
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

var (
	storePath string
	logLevel  string
)

func init() {
	viper.AutomaticEnv()
	flags := rootCmd.PersistentFlags()
	flags.String("kubeconfig", "", "kubeconfig to use")
	flags.StringVar(&logLevel, "loglevel", "debug", "set the loglevel")
	viper.BindPFlags(flags)
	viper.BindEnv("loglevel", "LOGLEVEL")
	viper.BindEnv("kubeconfig", "KUBECONFIG")
}

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
