package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/GnEveLynn/FormTally/server/internal/aieval"
)

func main() { os.Exit(run(os.Args[1:], os.Stdout)) }

func run(args []string, output io.Writer) int {
	flags := flag.NewFlagSet("ai-eval", flag.ContinueOnError)
	flags.SetOutput(output)
	manifestPath := flags.String("manifest", "", "path to an AI quality manifest")
	allowBlocked := flags.Bool("allow-blocked", false, "return success for a valid but externally blocked evaluation")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if *manifestPath == "" {
		fmt.Fprintln(output, "manifest is required")
		return 2
	}
	file, err := os.Open(*manifestPath)
	if err != nil {
		fmt.Fprintf(output, "open manifest: %v\n", err)
		return 2
	}
	defer file.Close()
	var manifest aieval.Manifest
	decoder := json.NewDecoder(io.LimitReader(file, 8<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		fmt.Fprintf(output, "decode manifest: %v\n", err)
		return 2
	}
	report, err := aieval.Evaluate(manifest)
	if err != nil {
		fmt.Fprintf(output, "evaluate manifest: %v\n", err)
		return 2
	}
	result := struct {
		aieval.Report
		Gate aieval.Gate `json:"gate"`
	}{Report: report, Gate: aieval.Assess(report)}
	encoder := json.NewEncoder(output)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		return 2
	}
	switch result.Gate.Status {
	case aieval.StatusPass:
		return 0
	case aieval.StatusBlocked:
		if *allowBlocked {
			return 0
		}
		return 2
	default:
		return 1
	}
}
