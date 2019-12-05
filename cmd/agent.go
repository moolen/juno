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

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func init() {
	flags := agentCmd.PersistentFlags()
	flags.String("iface", "veth", "target interfaces for bpf injection")
	flags.Duration("sync-interval", time.Second*60, "sync intervall")
	flags.Duration("perf-poll-interval", time.Millisecond, "poll interval on perf map")
	flags.Int("cache-buffer-size", 3000, "cache buffer size")
	flags.Int("listen-port", 3000, "server port")
	flags.String("k8s-node", "", "kubernetes node name")
	flags.String("apiserver-address", "10.96.0.1", "kubernetes apiserver address")

	viper.BindPFlags(flags)
	viper.BindEnv("iface", "TARGET_INTERFACES")
	viper.BindEnv("listen-port", "LISTEN_PORT")
	viper.BindEnv("k8s-node", "KUBERNETES_NODE")
	viper.BindEnv("apiserver-address", "APISERVER_ADDRESS")
	rootCmd.AddCommand(agentCmd)
}

var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Agent [...]",
	Run: func(cmd *cobra.Command, args []string) {
		log.Infof("starting agent")
		kubeClient, err := newClient()
		if err != nil {
			log.Fatal(err)
		}
		bpfController, err := controller.New(
			kubeClient,
			viper.GetString("iface"),
			viper.GetString("k8s-node"),
			viper.GetString("apiserver-address"),
			viper.GetDuration("sync-interval"),
			viper.GetDuration("perf-poll-interval"),
			viper.GetInt("listen-port"),
			viper.GetInt("cache-buffer-size"),
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

func newClient() (*kubernetes.Clientset, error) {
	var cfg *rest.Config
	var err error
	kubeConfig := viper.GetString("kubeconfig")
	if kubeConfig == "" {
		cfg, err = rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
	} else {
		cfg, err = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeConfig}, &clientcmd.ConfigOverrides{}).ClientConfig()
		if err != nil {
			return nil, err
		}
	}
	return kubernetes.NewForConfig(cfg)
}
