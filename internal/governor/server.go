package governor

import (
	"encoding/json"
	"net/http"
	"sync"
)

type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  ToolCall        `json:"params"`
}

type ToolCall struct {
	RunID     string `json:"run_id"`
	ParentID  string `json:"parent_id"`
	CallID    string `json:"call_id"`
	Tool      string `json:"tool"`
	EgressURL string `json:"egress_url"`
	TokenCost int    `json:"token_cost"`
}

type Decision struct {
	Allowed bool   `json:"allowed"`
	Code    string `json:"code,omitempty"`
	Reason  string `json:"reason,omitempty"`
}

func Allow() Decision { return Decision{Allowed: true} }
func Deny(code, reason string) Decision { return Decision{Allowed: false, Code: code, Reason: reason} }

type JSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  *Decision       `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    string `json:"data,omitempty"`
}

type Server struct {
	policy Policy
	mu     sync.Mutex
	graphs map[string]*CallGraph
}

func NewServer(policy Policy) *Server {
	return &Server{policy: policy, graphs: make(map[string]*CallGraph)}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	defer r.Body.Close()

	var request JSONRPCRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		s.write(w, http.StatusBadRequest, JSONRPCResponse{JSONRPC: "2.0", Error: &RPCError{Code: -32700, Message: "invalid JSON"}})
		return
	}
	if request.JSONRPC != "2.0" || request.Method != "tool.execute" {
		s.write(w, http.StatusBadRequest, JSONRPCResponse{JSONRPC: "2.0", ID: request.ID, Error: &RPCError{Code: -32601, Message: "unsupported method"}})
		return
	}

	decision := s.policy.Evaluate(request.Params)
	if decision.Allowed {
		decision = s.checkAndRecordGraph(request.Params)
	}
	s.write(w, http.StatusOK, JSONRPCResponse{JSONRPC: "2.0", ID: request.ID, Result: &decision})
}

func (s *Server) checkAndRecordGraph(call ToolCall) Decision {
	if call.RunID == "" || call.CallID == "" {
		return Deny("invalid_call", "run_id and call_id are required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	graph := s.graphs[call.RunID]
	if graph == nil {
		graph = NewCallGraph()
		s.graphs[call.RunID] = graph
	}
	if call.ParentID != "" {
		if graph.WouldCreateCycle(call.ParentID, call.CallID) {
			return Deny("cycle_detected", "tool call would create a dependency cycle")
		}
		graph.Add(call.ParentID, call.CallID)
	}
	return Allow()
}

func (s *Server) write(w http.ResponseWriter, status int, response JSONRPCResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response)
}
