# Custom panels

A custom Mykrobe2 panel is distributed and used as a single `.panelindex` file.
Build the probe FASTA files and any metadata first, then package them with
`mykrobe2 index`. The index contains the k-mer index, probe sequences, and the
optional resistance and lineage metadata, so the files used to create it are not
needed when running prediction.

The overall workflow is:

1. Make variant probes with `mykrobe2 make-probes`, and optionally prepare
   presence/absence probes in FASTA format.
2. Make a variant-to-resistance JSON file if the panel makes AMR predictions.
3. Build `custom.panelindex` with `mykrobe2 index`.
4. Run `mykrobe2 predict --index custom.panelindex`.

All coordinates in the text inputs below are 1-based.

## Make variant probes

For genome-coordinate variants, use a tab-separated file with five columns:
reference name, position, reference base, alternate base, and alphabet. The
reference-name field is normally `ref`.

```text
ref	1000	A	T	DNA
```

This represents `A1000T`. Make its probes against a reference FASTA:

```bash
mykrobe2 make-probes reference.fa --text-file vars.ref.tsv > probes.ref.fa
```

Alternatively, pass simple DNA variants directly:

```bash
mykrobe2 make-probes reference.fa --variants A1000T,C245G > probes.ref.fa
```

VCF input is also supported:

```bash
mykrobe2 make-probes reference.fa --vcf variants.vcf > probes.ref.fa
```

For variants described relative to genes, provide a GenBank annotation and a
tab-separated file with gene name, mutation, and alphabet (`DNA` or `PROT`):

```text
ileS	D2E	PROT
```

```bash
mykrobe2 make-probes reference.fa \
  --genbank reference.gbk \
  --text-file vars.gene.tsv > probes.gene.fa
```

Use the same k-mer length for `make-probes` and `index`; the default is 31.
For example, for 21-mers, add `--kmer 21` to `make-probes` and `--k 21` to
`index`.

## Add presence/absence probes

Sequences whose presence implies resistance are already probes, so do not pass
them through `make-probes`. Put them in a FASTA file. Each sequence header must
include its name and version, for example:

```fasta
>presAbs?name=presAbs&version=1
GTGGCAAGGCTTTTTACACAGCCTTTAGCTTCCCCGTTTTTTTATAGCAAGTTCGTAATT
TCGGAAATTGGGACGCTCAGACATTAATCTGCGGTGGGCGTTAACCTGACTGCACAAGTA
GTTCTAAGGAACATCTTTGG
```

More than one sequence version may be supplied by using `version=2`, `version=3`,
and so on.

## Define resistance predictions

Create a JSON object that maps each variant or presence/absence sequence name to
the drugs it confers resistance to. Variant names are the same names supplied to
`make-probes`; for gene mutations use `gene_mutation`.

```json
{
  "ileS_D2E": ["Drug1"],
  "A1000T": ["Drug2"],
  "presAbs": ["Drug3", "Drug4"]
}
```

Save this as, for example, `var2res.json`.

## Build and use the panel index

`index` accepts one or more FASTA files, so there is no need to concatenate the
variant and presence/absence probe files:

```bash
mykrobe2 index \
  --fasta probes.ref.fa,probes.gene.fa,probes.pres_abs.fa \
  --variant-to-resistance-json var2res.json \
  --output custom.panelindex
```

Run prediction with the resulting self-contained index:

```bash
mykrobe2 predict --sample sample_name \
  --index custom.panelindex \
  --seq test_reads.fq.gz \
  --output result.json
```

Do not use the Mykrobe 1 `--custom_probe_set_path` or
`--custom_variant_to_resistance_json` options: custom panels in Mykrobe2 are
provided through `--index`.

For lineage definitions, see [custom lineage calling](custom-lineage-calling.md).
For nearby variation that can affect probe matching, see
[background variants](background-variants.md).
