package main

import (
	"bytes"
	"testing"

	"github.com/martinghunt/mykrobe2/internal/buildinfo"
)

func TestRootVersionFlag(t *testing.T) {
	previous := buildinfo.Version
	buildinfo.Version = "v1.2.3"
	t.Cleanup(func() {
		buildinfo.Version = previous
	})

	cmd := newRootCmd()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)
	cmd.SetArgs([]string{"--version"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if stdout.String() != "mykrobe2 1.2.3\n" {
		t.Fatalf("stdout = %q, want %q", stdout.String(), "mykrobe2 1.2.3\n")
	}
}

func TestDisplayVersion(t *testing.T) {
	tests := []struct {
		raw  string
		want string
	}{
		{raw: "v1.2.3", want: "1.2.3"},
		{raw: "V1.2.3", want: "1.2.3"},
		{raw: "1.2.3", want: "1.2.3"},
		{raw: "dev", want: "dev"},
		{raw: "version", want: "version"},
	}

	for _, tt := range tests {
		if got := displayVersion(tt.raw); got != tt.want {
			t.Fatalf("displayVersion(%q) = %q, want %q", tt.raw, got, tt.want)
		}
	}
}

func TestFlagNamesAcceptDashesAndUnderscores(t *testing.T) {
	cmd := newRootCmd()
	predict, _, err := cmd.Find([]string{"predict"})
	if err != nil {
		t.Fatal(err)
	}
	dashed := predict.Flags().Lookup("panels-dir")
	underscored := predict.Flags().Lookup("panels_dir")
	if dashed == nil || underscored == nil || dashed != underscored {
		t.Fatalf("flag aliases do not resolve to the same flag: dashed=%p underscored=%p", dashed, underscored)
	}
}

func TestCommandNamesAcceptDashesAndUnderscores(t *testing.T) {
	cmd := newRootCmd()
	tests := []struct {
		args []string
		want string
	}{
		{args: []string{"compare-output"}, want: "compare-output"},
		{args: []string{"compare_output"}, want: "compare-output"},
		{args: []string{"panels", "update-metadata"}, want: "update-metadata"},
		{args: []string{"panels", "update_metadata"}, want: "update-metadata"},
		{args: []string{"panels", "update-species"}, want: "update-species"},
		{args: []string{"panels", "update_species"}, want: "update-species"},
	}
	for _, tt := range tests {
		got, _, err := cmd.Find(tt.args)
		if err != nil {
			t.Fatalf("Find(%v) error = %v", tt.args, err)
		}
		if got.Name() != tt.want {
			t.Fatalf("Find(%v) name = %q, want %q", tt.args, got.Name(), tt.want)
		}
	}
}
