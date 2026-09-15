package governor

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
)

// Policy is intentionally small. In production this decision would be made by
// an embedded OPA/Rego policy; keeping it JSON makes the demo runnable without
// an external policy service or dependency.
type Policy struct {
	AllowedTools   []string `json:"allowed_tools"`
	AllowedDomains []string `json:"allowed_domains"`
	MaxTokenCost   int      `json:"max_token_cost"`
}

func LoadPolicy(path string) (Policy, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return Policy{}, err
	}
	var policy Policy
	if err := json.Unmarshal(contents, &policy); err != nil {
		return Policy{}, err
	}
	if policy.MaxTokenCost < 1 {
		return Policy{}, fmt.Errorf("max_token_cost must be greater than zero")
	}
	return policy, nil
}

func (p Policy) Evaluate(call ToolCall) Decision {
	if !contains(p.AllowedTools, call.Tool) {
		return Deny("tool_not_allowed", "tool is not on the allow-list")
	}
	if call.TokenCost < 0 || call.TokenCost > p.MaxTokenCost {
		return Deny("budget_exceeded", "requested token cost exceeds the per-call budget")
	}
	if call.EgressURL != "" && !domainAllowed(call.EgressURL, p.AllowedDomains) {
		return Deny("egress_denied", "destination is not on the egress allow-list")
	}
	return Allow()
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func domainAllowed(rawURL string, allowed []string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return false
	}
	return contains(allowed, parsed.Hostname())
}
