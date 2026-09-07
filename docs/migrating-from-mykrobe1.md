# Migrating from Mykrobe 1

This page highlights the command-line usage changes when moving from Mykrobe 1
to Mykrobe2.

## Installation and desktop app

The desktop app is available again for macOS, Windows, and Linux. Download the
appropriate self-contained app from the release page; no Python or other
dependencies need to be installed.

The command-line program is similarly distributed as a standalone binary. After
downloading it, run `mykrobe2` directly rather than installing a Python package
or setting up a Python environment.

## Command name and installed panels

Use `mykrobe2` instead of `mykrobe`. The installed-panel workflow is otherwise
similar:

```bash
mykrobe2 panels update-metadata
mykrobe2 panels update-species all
mykrobe2 predict --sample sample_name --species tb --seq reads.fq.gz --output out.json
```

Use `mykrobe2 panels describe` to see the species and panel versions available
in your local panel-data directory.

## No more skeleton files

Mykrobe 1 built and reused Cortex `.ctx` "skeleton" files for each panel, by
default under `mykrobe/data/skeletons/`. Mykrobe2 does not create or require
skeleton files when running `predict`; panel indexes are prepared when panels
are installed, and custom panels are supplied as `.panelindex` files.

The Mykrobe 1 `--skeleton_dir` and `--force` options therefore have no Mykrobe2
equivalents.

## Custom panels now use an index file

This is the main workflow change. In Mykrobe 1, `predict` received separate
paths to a custom probe FASTA and optional AMR and lineage JSON files:

```bash
mykrobe predict --species custom \
  --custom_probe_set_path probes.fa \
  --custom_variant_to_resistance_json var2res.json \
  --custom_lineage_json lineage.json \
  --seq reads.fq.gz
```

In Mykrobe2, first make the probe FASTA, then package it and its metadata into a
self-contained `.panelindex` file:

```bash
mykrobe2 make-probes reference.fa --text-file variants.tsv > probes.fa
mykrobe2 index \
  --fasta probes.fa \
  --variant-to-resistance-json var2res.json \
  --lineage-json lineage.json \
  --output custom.panelindex
```

Run the custom panel with `--index`; do not use `--species custom` or the old
`--custom_*` options:

```bash
mykrobe2 predict --sample sample_name \
  --index custom.panelindex \
  --seq reads.fq.gz \
  --output out.json
```

The index contains the probe data and supplied metadata. Keep the source files
if you want to rebuild it, but they do not have to accompany the index when it
is used or shared.

## Probe generation command

`make-probes` is now a top-level command:

```text
Mykrobe 1: mykrobe variants make-probes ...
Mykrobe2:  mykrobe2 make-probes ...
```

For a custom index, use the same k-mer length when making probes and building
the index: `--kmer 21` on `make-probes` means `--k 21` on `index`.

## Background variants no longer use MongoDB

Mykrobe 1 required loading background VCFs into MongoDB with
`mykrobe variants add`, then selecting that database with `--db_name` when
running `make-probes`.

Mykrobe2 takes the background VCF files directly when probes are made:

```bash
mykrobe2 make-probes reference.fa \
  --text-file variants.tsv \
  --background-vcf background1.vcf \
  --background-vcf background2.vcf \
  > probes.fa
```

For a large collection, use `--background-vcf-list`, containing one VCF path per
line. Rebuild the probes and `.panelindex` whenever the background catalogue
changes.

See [custom panels](custom-panels.md),
[custom lineage calling](custom-lineage-calling.md), and
[background variants](background-variants.md) for the full workflows.
