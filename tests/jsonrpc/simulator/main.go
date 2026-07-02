package main

import (
	"flag"
	"log"

	"github.com/shardeum/shardeum-evm/tests/jsonrpc/simulator/report"
	"github.com/shardeum/shardeum-evm/tests/jsonrpc/simulator/runner"

	_ "embed"
)

func main() {
	verbose := flag.Bool("v", false, "Enable verbose output")
	outputExcel := flag.Bool("xlsx", false, "Save output as xlsx")
	flag.Parse()

	rCtx, err := runner.Setup()
	if err != nil {
		log.Fatalf("Setup failed: %v", err)
	}

	// Execute all tests
	results := runner.ExecuteAllTests(rCtx)

	// Generate report
	report.Results(results, *verbose, *outputExcel, rCtx)
}
