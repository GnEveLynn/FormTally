package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunPrintsBlockedReportAndRequiresExplicitAllowance(t *testing.T) {
	manifest := `{"version":"v1","model":"test-model","promptVersion":"meal_analysis_v1","evaluatedAt":"2026-09-13T01:00:00Z","samples":[{"id":"sample-1","imagePath":"images/not-committed.jpg","category":"mixed","hiddenOil":false,"weighedGrams":300,"reference":{"energyKcal":500,"proteinGrams":20,"carbGrams":60,"fatGrams":18},"evaluation":{"structured":true,"estimatedEnergyKcal":520,"durationMs":1500,"costUsd":0.01}}]}`
	path := filepath.Join(t.TempDir(), "manifest.json")
	if err := os.WriteFile(path, []byte(manifest), 0600); err != nil {
		t.Fatal(err)
	}

	var output bytes.Buffer
	if code := run([]string{"-manifest", path}, &output); code != 2 {
		t.Fatalf("exit = %d, output=%s", code, output.String())
	}
	if !strings.Contains(output.String(), `"status": "blocked"`) || !strings.Contains(output.String(), `"sampleCount": 1`) {
		t.Fatalf("output = %s", output.String())
	}
	output.Reset()
	if code := run([]string{"-manifest", path, "-allow-blocked"}, &output); code != 0 {
		t.Fatalf("allow-blocked exit = %d, output=%s", code, output.String())
	}
}

func TestRunRejectsMissingManifest(t *testing.T) {
	var output bytes.Buffer
	if code := run(nil, &output); code != 2 || !strings.Contains(output.String(), "manifest is required") {
		t.Fatalf("exit/output = %d %s", code, output.String())
	}
}
