package controller

import (
	"bytes"
	"io/ioutil"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/awalterschulze/gographviz"
	"github.com/moolen/juno/pkg/tracer"
	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Controller ...
type Controller struct {
	Client *kubernetes.Clientset
	Tracer *tracer.Tracer

	// track k8s state
	k8sMutex *sync.RWMutex
	pods     map[string]v1.Pod
	services map[string]v1.Service

	// track grap data
	graphMutex *sync.RWMutex
	// key: service node
	// value: edges start at this node
	graphData map[string][]string

	// this bool signals shutdown
	stop bool
}

// New ...
func New(client *kubernetes.Clientset, ifacePrefix string) (*Controller, error) {
	t, err := tracer.NewTracer(ifacePrefix)
	if err != nil {
		return nil, err
	}
	return &Controller{
		k8sMutex:   &sync.RWMutex{},
		pods:       make(map[string]v1.Pod),
		services:   make(map[string]v1.Service),
		graphMutex: &sync.RWMutex{},
		graphData:  make(map[string][]string),
		Client:     client,
		Tracer:     t,
	}, nil
}

func (c *Controller) pollK8sResources() {
	var err error
	var pods *v1.PodList
	var services *v1.ServiceList

	log.Infof("start polling k8s resources")

	for {
		pods, err = c.Client.CoreV1().Pods("").List(metav1.ListOptions{
			// LabelSelector: "app",
		})
		if err != nil {
			log.Error(err)
			goto wait
		}
		services, err = c.Client.CoreV1().Services("").List(metav1.ListOptions{
			// LabelSelector: "app",
		})
		if err != nil {
			log.Error(err)
			goto wait
		}

		c.k8sMutex.Lock()
		log.Debugf("writing k8s resources to cache")
		for _, po := range pods.Items {
			c.pods[po.Status.PodIP] = po
			log.Infof("found pod: %s/%s", po.Namespace, po.Name)
		}
		for _, svc := range services.Items {
			log.Infof("found service: %s/%s", svc.Namespace, svc.Name)
			c.services[svc.Spec.ClusterIP] = svc
		}
		c.k8sMutex.Unlock()

	wait:
		if c.stop {
			log.Infof("stopping k8s poller...")
			return
		}
		log.Infof("sleeping...")
		<-time.After(20 * time.Second)
	}
}

func (c *Controller) pollEvents() {
	log.Infof("start polling tcp events")
	for {
		select {
		case e := <-c.Tracer.Read():
			log.Debugf("trace event: %s", e.String())
			sAddr := e.SourceAddr.String()
			dAddr := e.DestAddr.String()
			//sPort := strconv.Itoa(int(e.SourcePort))
			//dPort := strconv.Itoa(int(e.DestPort))

			c.k8sMutex.RLock()
			spo, spoOk := c.pods[sAddr]
			dpo, dpoOk := c.pods[dAddr]
			dsvc, dsvcOk := c.services[dAddr]
			if !spoOk {
				log.Debugf("could not find source pod for %s", sAddr)
				c.k8sMutex.RUnlock()
				continue
			}
			sourceName := spo.ObjectMeta.Labels["app"]
			dpoName := dpo.ObjectMeta.Labels["app"]
			dsvcName := dsvc.ObjectMeta.Labels["app"]

			if dpoOk && dpoName != "" {
				c.graphMutex.Lock()
				addEdge(c.graphData, sourceName, dpoName)
				c.graphMutex.Unlock()
				//httpEventCounter.WithLabelValues().Inc()
			} else if dsvcOk && dsvcName != "" {
				c.graphMutex.Lock()
				addEdge(c.graphData, sourceName, dsvcName)
				c.graphMutex.Unlock()
				// TODO: add metrics
			} else {

				log.Debugf("missing src/dst for %s/%s", e.SourceAddr, e.DestAddr)
			}
			c.k8sMutex.RUnlock()
		default:
			if c.stop {
				return
			}
		}
	}
}

func addEdge(data map[string][]string, source, dest string) {
	if data[source] == nil {
		data[source] = []string{}
	}
	for _, v := range data[source] {
		if v == dest {
			return
		}
	}
	data[source] = append(data[source], dest)
}

func sanitize(in string) string {
	return strings.Replace(in, "-", "", -1)
}

func (c *Controller) genGraph() {
	for {
		// generate graph from graphData
		graphAst, _ := gographviz.ParseString(`digraph G {}`)
		graph := gographviz.NewGraph()
		if err := gographviz.Analyse(graphAst, graph); err != nil {
			panic(err)
		}

		c.graphMutex.RLock()
		log.Debugf("raw graph data: %#v", c.graphData)
		// add source nodes and prepare destinations nodes
		destinations := make(map[string]struct{})
		for src, ds := range c.graphData {
			for _, dest := range ds {
				destinations[dest] = struct{}{}
			}
			graph.AddNode("G", sanitize(src), nil)
		}
		// add destinations nodes
		for dest := range destinations {
			graph.AddNode("G", sanitize(dest), nil)
		}
		// add edges
		for src, ds := range c.graphData {
			for _, dest := range ds {
				graph.AddEdge(sanitize(src), sanitize(dest), true, nil)
			}
		}

		output := graph.String()
		log.Infof("graph: %s", output)
		c.graphMutex.RUnlock()

		cmd := exec.Command("dot", "-Tsvg")
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		cmd.Stdin = bytes.NewBufferString(output)
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		err := cmd.Run()
		if err != nil {
			log.Error(err)
			goto wait
		}
		log.Debugf("dot stderr: %s", stderr.String())
		err = ioutil.WriteFile("/tmp/graph.svg", stdout.Bytes(), 0777)
		if err != nil {
			log.Error(err)
		}
	wait:
		<-time.After(time.Second * 10)
	}
}

// Start ..
func (c *Controller) Start() {
	go c.pollK8sResources()
	go c.pollEvents()
	go c.genGraph()
	c.Tracer.Start()
}

// Stop ..
func (c *Controller) Stop() {
	c.Tracer.Stop()
	c.stop = true
}
