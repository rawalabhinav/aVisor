package main

import (
	"log"
	"net/http"
	"os"

	"github.com/rawalabhinav/aVisor/internal/governor"
)

func main() {
	policyPath := os.Getenv("AVISOR_POLICY")
	if policyPath == "" {
		policyPath = "policy.json"
	}

	policy, err := governor.LoadPolicy(policyPath)
	if err != nil {
		log.Fatalf("load policy: %v", err)
	}

	server := governor.NewServer(policy)
	log.Printf("aVisor listening on http://localhost:8080/rpc")
	log.Fatal(http.ListenAndServe(":8080", server))
}
