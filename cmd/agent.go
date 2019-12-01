package cmd

import (
	"net/http"
	"os"
	"os/signal"

	"github.com/moolen/juno/pkg/agent/controller"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"k8s.io/client-go/kubernetes"

	ctrl "sigs.k8s.io/controller-runtime"
)

func init() {
	flags := agentCmd.PersistentFlags()
	//nodeName := flags.String("node", "", "kubernetes node name. used to fetch resources specific to this node")
	flags.String("iface", "veth", "target interfaces for bpf injection")
	viper.BindPFlags(flags)
	viper.BindEnv("iface", "TARGET_INTERFACES")
	rootCmd.AddCommand(agentCmd)
}

var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Agent [...]",
	Run: func(cmd *cobra.Command, args []string) {
		log.Infof("starting agent")
		cfg, err := ctrl.GetConfig()
		if err != nil {
			log.Fatal(err)
		}
		clientset, err := kubernetes.NewForConfig(cfg)
		if err != nil {
			log.Error(err)
		}
		// TODO: pass in nodename? (see below)
		bpfController, err := controller.New(clientset, viper.GetString("iface"))
		if err != nil {
			log.Fatal(err)
		}

		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		go func() {
			<-c
			log.Infof("received ctrl+c, cleaning up")
			bpfController.Stop()
			log.Infof("shutting down")
			os.Exit(0)
		}()

		bpfController.Start()

		/* ctx := context.Background()
		nodeName := "my-node"
		var pods v1.PodList
		k8sClient.List(ctx, types.NamespacedName{}, &pods)

			FieldSelector: "spec.nodeName=" + nodeName,
		*/
		http.Handle("/metrics", promhttp.Handler())
		http.ListenAndServe(":2112", nil)
	},
}
