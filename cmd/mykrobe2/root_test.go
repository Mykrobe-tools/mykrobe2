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
