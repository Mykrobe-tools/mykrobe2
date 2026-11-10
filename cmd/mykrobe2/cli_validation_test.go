package main

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestCommandArgumentCounts(t *testing.T) {
	commands := []struct {
		path []string
		n    int
	}{
		{nil, 0},
		{[]string{"predict"}, 0},
		{[]string{"index"}, 0},
		{[]string{"panels"}, 0},
		{[]string{"panels", "describe"}, 0},
		{[]string{"panels", "update-metadata"}, 0},
		{[]string{"panels", "update_metadata"}, 0},
		{[]string{"panels", "update-species"}, 1},
		{[]string{"panels", "update_species"}, 1},
		{[]string{"make-probes"}, 1},
		{[]string{"download-test-reads"}, 1},
		{[]string{"compare-output"}, 2},
		{[]string{"compare_output"}, 2},
	}
	for _, tt := range commands {
		t.Run(strings.Join(tt.path, "/"), func(t *testing.T) {
			for n := 0; n <= tt.n+1; n++ {
				root := newRootCmd()
				cmd, _, err := root.Find(tt.path)
				if err != nil {
					t.Fatal(err)
				}
				called := false
				cmd.RunE = func(*cobra.Command, []string) error {
					called = true
					return nil
				}
				var output bytes.Buffer
				root.SetOut(&output)
				root.SetErr(&output)
				args := append([]string(nil), tt.path...)
				for i := 0; i < n; i++ {
					args = append(args, "argument")
				}
				root.SetArgs(args)
				err = root.Execute()
				if n == tt.n {
					if err != nil || !called {
						t.Errorf("%v: valid arguments rejected: %v", args, err)
					}
				} else if err == nil || called {
					t.Errorf("%v: invalid arguments reached handler=%t, error=%v", args, called, err)
				}
			}
		})
	}
}

func TestCommandsDeclareArgumentValidation(t *testing.T) {
	var check func(*cobra.Command)
	check = func(cmd *cobra.Command) {
		if cmd.Args == nil {
			t.Errorf("%s has no argument validator", cmd.CommandPath())
		}
		for _, child := range cmd.Commands() {
			check(child)
		}
	}
	check(newRootCmd())
}

func TestFileFlagsPreserveLiteralPaths(t *testing.T) {
	for _, tt := range []struct {
		newCmd func() *cobra.Command
		flag   string
	}{
		{newPredictCmd, "seq"},
		{newPredictCmd, "i"},
		{newIndexCmd, "fasta"},
		{newMakeProbesCmd, "background-vcf"},
	} {
		t.Run(tt.flag, func(t *testing.T) {
			cmd := tt.newCmd()
			option := "--" + tt.flag
			name := tt.flag
			if name == "i" {
				option = "-i"
				name = "seq"
			}
			want := []string{"sample,one file.fa", `sample"two.fa`, "three.fa"}
			args := []string{option, want[0], option, want[1], option + "=" + want[2]}
			if err := cmd.ParseFlags(args); err != nil {
				t.Fatal(err)
			}
			got, err := cmd.Flags().GetStringArray(name)
			if err != nil || !reflect.DeepEqual(got, want) {
				t.Fatalf("parsed paths = %q, want %q, error = %v", got, want, err)
			}
		})
	}
}

func TestVariantFlagAcceptsRepeatedAndCommaSeparatedValues(t *testing.T) {
	cmd := newMakeProbesCmd()
	if err := cmd.ParseFlags([]string{"--variants", "A100T,C200G", "--variants", "G300A"}); err != nil {
		t.Fatal(err)
	}
	got, err := cmd.Flags().GetStringSlice("variants")
	if err != nil || !reflect.DeepEqual(got, []string{"A100T", "C200G", "G300A"}) {
		t.Fatalf("parsed variants = %v, error = %v", got, err)
	}
}

func TestPredictRejectsInvalidChoicesBeforeProcessing(t *testing.T) {
	for _, flag := range []string{"model", "ploidy", "format"} {
		t.Run(flag, func(t *testing.T) {
			dir := t.TempDir()
			out := filepath.Join(dir, "existing.json")
			progress := filepath.Join(dir, "progress.jsonl")
			const original = "existing output"
			if err := os.WriteFile(out, []byte(original), 0o644); err != nil {
				t.Fatal(err)
			}
			err := run([]string{"predict", "--" + flag, "typo", "--output", out, "--gui-progress-file", progress})
			if err == nil || !strings.Contains(err.Error(), "invalid --"+flag) {
				t.Fatalf("expected invalid --%s error before loading inputs, got %v", flag, err)
			}
			data, err := os.ReadFile(out)
			if err != nil || string(data) != original {
				t.Fatalf("existing output changed: %q, %v", data, err)
			}
			if _, err := os.Stat(progress); !os.IsNotExist(err) {
				t.Fatalf("progress file should not be created: %v", err)
			}
		})
	}
}
