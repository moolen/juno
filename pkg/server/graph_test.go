package server

import "testing"

func TestGraph(t *testing.T) {
	g := NewGraph()

	n1 := &Node{
		ServiceID: "1",
	}
	n2 := &Node{
		ServiceID: "2",
	}
	n3 := &Node{
		ServiceID: "3",
	}

	g.AddNode(n1)
	g.AddNode(n2)
	g.AddNode(n3)

	g.EnsureEdge(n1, n2)
	g.EnsureEdge(n1, n3)

	g.WriteDotGraph("/tmp/graph.svg")
}
