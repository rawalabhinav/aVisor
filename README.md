# aVisor — Agent Egress Governor

A small Go demonstration of a **zero-trust control point** for AI-agent tool calls. An agent sends a JSON-RPC `tool.execute` request to aVisor before it performs work. aVisor either allows it or returns a structured reason for blocking it.

It demonstrates the ideas behind the resume project without infrastructure dependencies:

| Resume concept | This demo |
| --- | --- |
| Inline JSON-RPC proxy | `POST /rpc` accepts `tool.execute` requests and returns an allow/deny decision. |
| OPA policy enforcement | A tiny JSON allow-list policy evaluator. Its decision boundary is deliberately shaped so it can be replaced by embedded OPA/Rego. |
| Redis graph cycle detection | An in-memory, per-agent-run dependency graph that prevents recursive tool-call loops. |
| gVisor sandboxing | The proxy returns an authorization decision before execution. A real executor would run allowed calls with `runsc`/gVisor after this check. |

## Run

Requires Go 1.22+.

```sh
go test ./...
go run ./cmd/avisor
```

In another terminal:

```sh
curl -s http://localhost:8080/rpc \
  -H 'content-type: application/json' \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "tool.execute",
    "params": {
      "run_id": "demo-run",
      "call_id": "search-1",
      "tool": "search",
      "egress_url": "https://api.example.com/search?q=weather",
      "token_cost": 120
    }
  }'
```

Expected response:

```json
{"jsonrpc":"2.0","id":1,"result":{"allowed":true}}
```

Change the domain to `evil.example`, use an unknown tool, or set `token_cost` above 500 to see a denial. Edit `policy.json` to adjust the demo policy.

## How to explain it in an interview

1. **Assume agent input is untrusted.** Prompt injection can persuade an agent to call unexpected tools or send data to an attacker-controlled URL. The governor sits between the agent and tools, so the agent cannot bypass its rules.
2. **Make each request a policy decision.** The demo checks the tool name, egress destination, and estimated token cost. The response contains a machine-readable reason such as `egress_denied` or `budget_exceeded`.
3. **Stop loops before they become expensive.** Tool calls form a graph: a tool can trigger another tool. If a new edge makes a path lead back to an earlier call, the governor blocks it. The demo keeps the graph in memory; Redis would share it across replicas.
4. **Separate authorization from isolation.** This service decides *whether* a call may run. A sandbox such as gVisor limits what an allowed call can do at the OS boundary. They solve different problems and work together.

## Deliberate limits

This is a learning project, not a production proxy. It does not forward calls to real tools, authenticate callers, persist graphs, stream responses, estimate token use automatically, or launch gVisor. Those are the natural next steps after the decision path is understood.
