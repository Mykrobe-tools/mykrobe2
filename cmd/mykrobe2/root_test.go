package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Mykrobe-tools/mykrobe2/internal/buildinfo"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
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

func TestPredictRetainsLegacyShortOptions(t *testing.T) {
	cmd := newPredictCmd()
	var stderr bytes.Buffer
	cmd.SetErr(&stderr)
	if err := cmd.ParseFlags([]string{
		"-s", "SAMPLE",
		"-k", "21",
		"-A",
		"-e", "0.1",
		"-D", "0.4",
		"-o", "result.json",
		"-S", "tb",
		"-O", "json",
	}); err != nil {
		t.Fatal(err)
	}

	wants := map[string]string{
		"sample":                        "SAMPLE",
		"kmer":                          "21",
		"report-all-calls":              "true",
		"expected-error-rate":           "0.1",
		"min-proportion-expected-depth": "0.4",
		"output":                        "result.json",
		"species":                       "tb",
		"format":                        "json",
	}
	for name, want := range wants {
		flag := cmd.Flags().Lookup(name)
		if flag == nil {
			t.Fatalf("flag --%s not found", name)
		}
		if got := flag.Value.String(); got != want {
			t.Errorf("flag --%s = %q, want %q", name, got, want)
		}
	}
}

func TestPredictAcceptsKmerLongOptions(t *testing.T) {
	for _, option := range []string{"--kmer", "--k"} {
		t.Run(option, func(t *testing.T) {
			cmd := newPredictCmd()
			if err := cmd.ParseFlags([]string{option, "21"}); err != nil {
				t.Fatal(err)
			}
			if got := cmd.Flags().Lookup("kmer").Value.String(); got != "21" {
				t.Fatalf("--kmer = %q after parsing %s, want 21", got, option)
			}
		})
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

func TestAllCommandsAndFlagsHaveHelpText(t *testing.T) {
	root := newRootCmd()
	var check func(*cobra.Command)
	check = func(cmd *cobra.Command) {
		t.Helper()
		if strings.TrimSpace(cmd.Short) == "" {
			t.Errorf("command %q has no help summary", cmd.CommandPath())
		}
		cmd.LocalNonPersistentFlags().VisitAll(func(flag *pflag.Flag) {
			if !flag.Hidden && strings.TrimSpace(flag.Usage) == "" {
				t.Errorf("flag --%s on command %q has no help text", flag.Name, cmd.CommandPath())
			}
		})
		for _, child := range cmd.Commands() {
			check(child)
		}
	}
	check(root)
}

func TestCommandHelpDescribesCommonWorkflows(t *testing.T) {
	tests := []struct {
		args []string
		want []string
	}{
		{args: nil, want: []string{"antimicrobial resistance", "panels", "predict"}},
		{args: []string{"predict"}, want: []string{"--species", "--index", "Sequence file(s) in FASTA, FASTQ, or BAM format"}},
		{args: []string{"panels"}, want: []string{"Inspect available panel data", "update-metadata", "update-species"}},
		{args: []string{"panels", "describe"}, want: []string{"whether updates are available", "Output format: text or json"}},
		{args: []string{"panels", "update-metadata"}, want: []string{"latest versions", "--manifest-file", "--manifest-url"}},
		{args: []string{"panels", "update-species"}, want: []string{"one species", "<species|all>"}},
		{args: []string{"make-probes"}, want: []string{"reference genome", "--background-vcf", "--variants"}},
		{args: []string{"index"}, want: []string{"self-contained custom panel index", "--fasta", "--output"}},
		{args: []string{"download-test-reads"}, want: []string{"TB test-read dataset", "<output-filename>"}},
		{args: []string{"compare-output"}, want: []string{"floating-point differences", "--float-tolerance"}},
	}

	for _, tt := range tests {
		name := "root"
		if len(tt.args) > 0 {
			name = strings.Join(tt.args, "_")
		}
		t.Run(name, func(t *testing.T) {
			cmd := newRootCmd()
			var output bytes.Buffer
			cmd.SetOut(&output)
			cmd.SetErr(&output)
			cmd.SetArgs(append(append([]string(nil), tt.args...), "--help"))
			if err := cmd.Execute(); err != nil {
				t.Fatal(err)
			}
			for _, want := range tt.want {
				if !strings.Contains(output.String(), want) {
					t.Errorf("help output does not contain %q:\n%s", want, output.String())
				}
			}
		})
	}
}

func TestRootHelpUsesWorkflowCommandOrder(t *testing.T) {
	cmd := newRootCmd()
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	help := output.String()
	previous := -1
	for _, command := range []string{"predict", "panels", "make-probes", "index", "download-test-reads", "help", "completion"} {
		position := strings.Index(help, "  "+command)
		if position < 0 {
			t.Errorf("command %q missing from help", command)
		} else if position < previous {
			t.Errorf("command %q is out of order:\n%s", command, help)
		}
		previous = position
	}
}
