package langgraph

// Graph is a minimal representation of an execution plan over plugins.
type Graph struct {
	Nodes []Node `json:"nodes" yaml:"nodes"`
}

// Node represents a single action. Supported types: transport_dial, mesh_log.
type Node struct {
	Type string `json:"type" yaml:"type"`
	Addr string `json:"addr,omitempty" yaml:"addr,omitempty"`
	Msg  string `json:"msg,omitempty" yaml:"msg,omitempty"`
}

// ExecuteGraph walks the nodes in order and invokes the corresponding plugin calls.
func (e *Executor) ExecuteGraph(g Graph) error {
	for _, n := range g.Nodes {
		switch n.Type {
		case "transport_dial":
			if err := e.ExecuteTransportPing(n.Addr); err != nil {
				return err
			}
		case "mesh_log":
			if e.Mesh != nil {
				if err := e.Mesh.LogEvent(n.Msg); err != nil {
					return err
				}
			}
		default:
			// ignore unknown nodes for now
		}
	}
	return nil
}
