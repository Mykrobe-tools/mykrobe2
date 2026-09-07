# Custom lineage calling

Custom lineage calling uses a lineage-definition JSON file embedded in a custom
`.panelindex`. Build the probes and lineage JSON once, package them with
`mykrobe2 index`, then use that index for every sample.

## Lineage names

The lineage tree is inferred from dot-separated names: `x` is the parent of
`x.1`, and `x.1` is the parent of `x.1.1` and `x.1.2`. A separate display name
can be assigned where needed.

## Create the lineage probes and definitions

Use the genome-coordinate text-file format described in
[custom panels](custom-panels.md). Add a sixth column to mark a variant as
lineage-defining, and an optional seventh column to set its displayed name:

```text
ref	1000	A	T	DNA	lineage1
ref	2000	C	A	DNA	lineage1.1
ref	3000	G	C	DNA	lineage1.2	1.2_display_name
ref	4000	T	A	DNA	lineage2
```

The non-reference allele normally defines the lineage: an observed `T` at
position 1000 supports `lineage1`. Prefix the lineage name with `*` when the
reference allele defines it:

```text
ref	5000	G	C	DNA	*lineage3
```

Here, `G` supports `lineage3`; `C` does not. More than one variant can be
assigned to a lineage, and any supporting variant is evidence for that lineage.

Run `make-probes`, using `--lineage` to write the definitions JSON:

```bash
mykrobe2 make-probes reference.fa \
  --text-file vars.ref.tsv \
  --lineage lineage.json > probes.fa
```

If the panel also contains AMR variants or presence/absence probes, make or add
those FASTA files as described in [custom panels](custom-panels.md). Keep the
k-mer length consistent: `--kmer 21` here requires `--k 21` when indexing.

## Build the index and call lineages

Embed both the probes and the lineage definitions in the custom index:

```bash
mykrobe2 index \
  --fasta probes.fa \
  --lineage-json lineage.json \
  --output custom.panelindex
```

Then run prediction:

```bash
mykrobe2 predict --sample sample_id \
  --index custom.panelindex \
  --seq reads.fq.gz \
  --output out.json
```

The lineage result is in the `phylogenetics` section of the JSON output. Add
`--report-all-calls` when inspecting the calls underlying a lineage result.
