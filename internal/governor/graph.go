package governor

// CallGraph stores parent -> child relationships for one agent run. It is a
// stand-in for the Redis-backed graph used by a distributed deployment.
type CallGraph struct {
	edges map[string]map[string]bool
}

func NewCallGraph() *CallGraph {
	return &CallGraph{edges: make(map[string]map[string]bool)}
}

// WouldCreateCycle reports whether adding parent -> child creates a loop.
func (g *CallGraph) WouldCreateCycle(parent, child string) bool {
	if parent == child {
		return true
	}
	return g.reachable(child, parent, map[string]bool{})
}

func (g *CallGraph) Add(parent, child string) {
	if g.edges[parent] == nil {
		g.edges[parent] = make(map[string]bool)
	}
	g.edges[parent][child] = true
}

func (g *CallGraph) reachable(from, wanted string, visited map[string]bool) bool {
	if from == wanted {
		return true
	}
	if visited[from] {
		return false
	}
	visited[from] = true
	for next := range g.edges[from] {
		if g.reachable(next, wanted, visited) {
			return true
		}
	}
	return false
}
