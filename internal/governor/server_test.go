package governor

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func testServer() *Server {
	return NewServer(Policy{AllowedTools: []string{"search", "fetch"}, AllowedDomains: []string{"api.example.com"}, MaxTokenCost: 100})
}

func TestServerAllowsSafeCall(t *testing.T) {
	server := testServer()
	request := httptest.NewRequest(http.MethodPost, "/rpc", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tool.execute","params":{"run_id":"r1","call_id":"root","tool":"search","egress_url":"https://api.example.com/v1","token_cost":20}}`))
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"allowed":true`) {
		t.Fatalf("expected allowed response, got %d: %s", response.Code, response.Body.String())
	}
}

func TestServerBlocksEgressAndBudget(t *testing.T) {
	for _, payload := range []string{
		`{"jsonrpc":"2.0","id":1,"method":"tool.execute","params":{"run_id":"r1","call_id":"a","tool":"search","egress_url":"https://evil.example","token_cost":20}}`,
		`{"jsonrpc":"2.0","id":1,"method":"tool.execute","params":{"run_id":"r1","call_id":"a","tool":"search","token_cost":101}}`,
	} {
		response := httptest.NewRecorder()
		testServer().ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/rpc", bytes.NewBufferString(payload)))
		if !strings.Contains(response.Body.String(), `"allowed":false`) {
			t.Fatalf("expected denied response: %s", response.Body.String())
		}
	}
}

func TestCallGraphDetectsCycle(t *testing.T) {
	graph := NewCallGraph()
	graph.Add("a", "b")
	graph.Add("b", "c")
	if !graph.WouldCreateCycle("c", "a") {
		t.Fatal("expected c -> a to create a cycle")
	}
}

func TestPolicyRejectsMalformedEgressURL(t *testing.T) {
	decision := testServer().policy.Evaluate(ToolCall{Tool: "search", EgressURL: "://bad", TokenCost: 1})
	if decision.Allowed || decision.Code != "egress_denied" {
		t.Fatalf("expected malformed URL denial, got %#v", decision)
	}
}
