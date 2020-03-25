package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os/exec"
	"strings"
	"sync"

	"github.com/awalterschulze/gographviz"
)

type Graph struct {
	mu    sync.RWMutex
	nodes []*Node          `json:"nodes"`
	edges map[Node][]*Node `json:"edges"`
}

type Node struct {
	ServiceID string `json:"service_id"`
}

// NewGraph returns a new graph
func NewGraph() *Graph {
	return &Graph{
		mu:    sync.RWMutex{},
		nodes: make([]*Node, 0),
		edges: make(map[Node][]*Node),
	}
}

func (g *Graph) FindNode(id string) *Node {
	g.mu.RLock()
	defer g.mu.RUnlock()
	for _, node := range g.nodes {
		if node.ServiceID == id {
			return node
		}
	}
	return nil
}

// AddNode adds a node
func (g *Graph) AddNode(n *Node) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.nodes = append(g.nodes, n)
}

// AddEdge adds an edge
func (g *Graph) AddEdge(n1, n2 *Node) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.edges == nil {
		g.edges = make(map[Node][]*Node)
	}
	g.edges[*n1] = append(g.edges[*n1], n2)
}

func (g *Graph) EnsureEdge(n1, n2 *Node) {
	g.mu.Lock()
	found := false
	for _, v := range g.edges[*n1] {
		if v == n2 {
			found = true
		}
	}
	g.mu.Unlock()
	if !found {
		g.AddEdge(n1, n2)
	}
}

func (g *Graph) WriteDotGraph(target string) error {
	g.mu.RLock()
	graphAst, _ := gographviz.ParseString(`digraph G {}`)
	graph := gographviz.NewGraph()
	if err := gographviz.Analyse(graphAst, graph); err != nil {
		return err
	}

	for i := 0; i < len(g.nodes); i++ {
		err := graph.AddNode("G", sanitize(g.nodes[i].ServiceID), nil)
		if err != nil {
			return err
		}
	}

	for i := 0; i < len(g.nodes); i++ {
		near := g.edges[*g.nodes[i]]
		for j := 0; j < len(near); j++ {
			graph.AddEdge(sanitize(g.nodes[i].ServiceID), sanitize(near[j].ServiceID), true, nil)
		}
	}

	g.mu.RUnlock()

	output := graph.String()

	cmd := exec.Command("dot", "-Tsvg")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdin = bytes.NewBufferString(output)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("error running dot: %s \nstdout: %s\n stderr: %s", err, stdout.String(), stderr.String())
	}
	return ioutil.WriteFile(target, stdout.Bytes(), 0777)
}

func (g *Graph) JSONGraph() ([]byte, error) {
	g.mu.RLock()
	var export ExportGraph
	for i := 0; i < len(g.nodes); i++ {
		export.Nodes = append(export.Nodes, ExportNode{
			ID:        g.nodes[i].ServiceID,
			ServiceID: g.nodes[i].ServiceID,
			Type:      "rect",
		})
	}
	for i := 0; i < len(g.nodes); i++ {
		near := g.edges[*g.nodes[i]]
		for j := 0; j < len(near); j++ {
			export.Edges = append(export.Edges, ExportEdge{
				Source: g.nodes[i].ServiceID,
				Target: near[j].ServiceID,
				Type:   "regular",
			})
		}
	}
	g.mu.RUnlock()
	return json.Marshal(export)
}

var replacer = strings.NewReplacer("/", "", "_", "", "-", "")

func sanitize(in string) string {
	return replacer.Replace(in)
}
