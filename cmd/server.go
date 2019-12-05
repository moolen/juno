package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/moolen/juno/pkg/server"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	flags := serverCmd.PersistentFlags()
	flags.String("target", "dns:///localhost:3000", "specify the grpc server to ask for traces. you may specify a dns+srv based discovery")
	flags.Int("listen", 3001, "specify the port to listen on")
	viper.BindPFlags(flags)
	viper.BindEnv("target", "TARGET_ADDR")
	viper.BindEnv("listen", "LISTEN")
	rootCmd.AddCommand(serverCmd)
}

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Server [...]",
	Run: func(cmd *cobra.Command, args []string) {
		log.Infof("starting server")
		// start poller to scrape the data
		// from the daemonsets.
		// it offers a channel to consume events
		// aswell as a array of items

		// 1st: spike end-to-end and push events
		// 2nd: add array buffer

		// put in channel
		srv, err := server.New(viper.GetString("target"), viper.GetInt("listen"))
		if err != nil {
			log.Fatal(err)
		}
		ctx, cancel := context.WithCancel(context.Background())
		stopChan := make(chan os.Signal)
		signal.Notify(stopChan, os.Interrupt)
		signal.Notify(stopChan, syscall.SIGTERM)

		go func() {
			<-stopChan
			log.Infof("stopping server")
			srv.Stop()
			cancel()
		}()

		srv.Serve(ctx)
	},
}
