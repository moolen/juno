package cmd

import (
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/moolen/juno/pkg/agent/controller"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	flags := agentCmd.PersistentFlags()
	flags.String("iface", "veth", "target interfaces for bpf injection")
	flags.Duration("sync-interval", time.Second*60, "poll interval to attach eBPF programs to interfaces")
	flags.Duration("perf-poll-interval", time.Millisecond, "poll interval on perf map")
	flags.String("k8s-node", "", "kubernetes node name")

	viper.BindPFlags(flags)
	viper.BindEnv("iface", "TARGET_INTERFACES")
	viper.BindEnv("sync-interval", "SYNC_INTERVAL")
	viper.BindEnv("perf-poll-interval", "PERF_POLL_INTERVAL")
	viper.BindEnv("k8s-node", "KUBERNETES_NODE")
	rootCmd.AddCommand(agentCmd)
}

var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "The agent captures network traffic on specific interfaces",
	Run: func(cmd *cobra.Command, args []string) {
		log.Infof("starting agent")
		bpfController, err := controller.New(
			viper.GetString("iface"),
			viper.GetString("k8s-node"),
			viper.GetDuration("sync-interval"),
			viper.GetDuration("perf-poll-interval"),
		)
		if err != nil {
			log.Fatal(err)
		}

		stopChan := make(chan os.Signal)
		signal.Notify(stopChan, os.Interrupt)
		signal.Notify(stopChan, syscall.SIGTERM)
		go func() {
			<-stopChan
			log.Infof("received ctrl+c, cleaning up")
			bpfController.Stop()
			log.Infof("shutting down")
			os.Exit(0)
		}()

		bpfController.Start()

		// start metrics server
		http.Handle("/metrics", promhttp.Handler())
		http.ListenAndServe(":2112", nil)
	},
}
