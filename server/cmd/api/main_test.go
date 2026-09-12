package main

import "testing"

func TestParseArgsSupportsConfigurationCheck(t *testing.T) {
	check, err := parseArgs([]string{"-check-config"})
	if err != nil || !check {
		t.Fatalf("check=%v err=%v", check, err)
	}
	if _, err := parseArgs([]string{"-unknown"}); err == nil {
		t.Fatal("unknown flag accepted")
	}
}
