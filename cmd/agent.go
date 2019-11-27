package cmd

import (
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/moolen/juno/pkg/agent/controller"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	//"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	//"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
)

var (
	nodeName *string
)

func init() {
	flags := agentCmd.PersistentFlags()
	nodeName = flags.String("node", "", "")
	viper.BindPFlags(flags)
	viper.BindEnv("node", "KUBERNETES_NODE")
	rootCmd.AddCommand(agentCmd)
}

var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Agent [...]",
	Run: func(cmd *cobra.Command, args []string) {
		log.Infof("starting agent")

		bpfController, err := controller.New()
		if err != nil {
			log.Fatal(err)
		}
		bpfController.Start()

		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		go func() {
			<-c
			log.Infof("received ctrl+c, cleaning up")
			bpfController.Stop()
			log.Infof("shutting down")
		}()

		/* ctx := context.Background()
		nodeName := "my-node"
		var pods v1.PodList
		k8sClient.List(ctx, types.NamespacedName{}, &pods)

			FieldSelector: "spec.nodeName=" + nodeName,
		*/

		go func() {

			cfg, err := ctrl.GetConfig()
			if err != nil {
				log.Error(err)
			}

			// use the current context in kubeconfig
			/* cfg, err := clientcmd.BuildConfigFromFlags("", "/.kube/config")
			if err != nil {
				log.Fatal(err.Error())
			} */

			clientset, err := kubernetes.NewForConfig(cfg)
			if err != nil {
				log.Error(err)
			}

			for {
				// get all pods
				pods, err := clientset.CoreV1().Pods("").List(metav1.ListOptions{})
				if err != nil {
					log.Error(err)
				}

				for _, po := range pods.Items {
					log.Infof("found pod: %s/%s", po.Namespace, po.Name)
				}

				log.Infof("sleeping...")
				time.Sleep(20 * time.Second)
			}
		}()

		http.Handle("/metrics", promhttp.Handler())
		http.ListenAndServe(":2112", nil)

	},
}
