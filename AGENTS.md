# AGENTS

## Purpose

This repository is a Go reimplementation of the parts of Mykrobe needed to produce a self-contained `mykrobe2`.

Two goals matter:

1. Match Mykrobe behavior where users depend on it.
2. Keep the Go code structurally cleaner than the Python codebase.

Do not copy Python structure unless it is the clearest option. Port behavior, tests, and externally visible semantics, not Python implementation style.

## Current Architecture

- `cmd/mykrobe2`
  - The only user-facing CLI.
  - Uses Cobra.
  - Should stay thin: parse flags, call library code, write output.

- `mccortex`
  - Internal library only.
  - Reimplements the subset of `mccortex` needed by Mykrobe.
  - Do not recreate a standalone `mccortex` CLI unless explicitly requested.

- `mykrobe`
  - Core typing, prediction, phylogenetics, lineage, and pipeline logic.

- `mykrobe/speciesdata`
  - Panel manifest and installed-panel management for Mykrobe-style panel bundles.

## Design Rules

- Prefer library APIs over subprocess-style orchestration.
- Keep file formats private unless compatibility is explicitly required.
- Do not implement Cortex binary `.ctx` compatibility unless requested.
- Do not introduce MongoDB or any database dependency for future `make-probes` work unless explicitly justified.
- Prefer typed structs and narrow interfaces over large unstructured maps, except where matching JSON-like Mykrobe outputs is simplest.
- Keep CLI concerns out of core packages.

## Behavioral Source of Truth

When behavior is unclear, use these in order:

1. Existing Mykrobe tests.
2. Existing `mccortex` tests for reused functionality.
3. Real Mykrobe panel data and fixture outputs.
4. Python implementation details, only when needed to explain behavior.

The target is behavioral parity, not source parity.

## Testing Policy

- High test coverage is expected.
- Reuse or port tests from `~/git/mykrobe` and `~/git/mccortex` whenever practical.
- Prefer fixture-driven tests over hand-wavy unit tests when user-visible outputs are involved.
- For new parity work, add a test that demonstrates the Python/Mykrobe behavior before or alongside the Go implementation.
- When real panel data is useful, use it.

## External Dependencies

- Sequence file reading should use `faqt` from `~/git/faqt`.
- Keep new dependencies minimal.
- Adding Cobra for CLI structure is acceptable.
- Avoid adding infrastructure-heavy dependencies.

## Code Placement

- Put graph / kmer / coverage mechanics in `mccortex`.
- Put domain logic such as typing, resistance prediction, lineage, and phylo in `mykrobe`.
- Put panel install / manifest logic in `mykrobe/speciesdata`.
- If `cmd/mykrobe2/main.go` grows, move orchestration into a reusable package and keep the Cobra layer thin.

## Near-Term Priorities

1. Finish parity for the Python `amr.py` / `predict` workflow.
2. Drive parity from real end-to-end fixtures and output diffs.
3. Improve output-schema compatibility only after core behavior is correct.
4. Reimplement `make-probes` later, without MongoDB.

## What Not To Do

- Do not reintroduce a separate `mccortex` executable.
- Do not optimize prematurely for binary compatibility with Cortex.
- Do not mirror Python package layout just for familiarity.
- Do not replace reused upstream test behavior with weaker bespoke tests.

## Practical Workflow

- Before major changes, inspect the relevant Python/Mykrobe tests.
- Make the smallest clean change that preserves or improves structure.
- Verify with `go test ./...` using local cache dirs if needed.
- If a real-data workflow is being changed, prefer validating it against a real Mykrobe panel bundle or fixture.
