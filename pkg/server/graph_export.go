package server

// ExportGraph roughly maps to statusgraph/graph
type ExportGraph struct {
	Nodes []ExportNode `json:"nodes"`
	Edges []ExportEdge `json:"edges"`
}

type ExportNode struct {
	ID        string `json:"id"`
	ServiceID string `json:"service_id"`
	Type      string `json:"type"`
}

type ExportEdge struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Type   string `json:"type"`
}
