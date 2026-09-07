# Background variants

Background variants let Mykrobe2 make probe alternatives that remain matchable
when common nearby variation occurs. They are supplied as VCF files directly to
`mykrobe2 make-probes`.

If expected calls are missing after you build a custom panel, omitted nearby
background SNPs are often the cause. This page explains how to include them
when making probes.

## Why they matter

Mykrobe matches exact k-mers. If a variant of interest is `C100T` and a sample
also has a nearby `G95T`, the nearby change can prevent the `C100T` probe k-mers
from matching and lead to a missed call. Providing `G95T` as a background variant
causes `make-probes` to generate compatible probe contexts for the target
variant.

The relevant neighbourhood is determined by the selected k-mer length. A larger
k-mer can therefore require a broader background-variant catalogue.

## Prepare VCF files

Put background SNPs in one or more VCF files aligned to the same reference FASTA
used to make probes. Mykrobe2 uses alternate alleles from `PASS` (or `.`) VCF
records and ignores filtered and symbolic alleles. Symbolic alleles are ALT
values that name a variant class rather than supplying its DNA sequence, such as
`<DEL>`, `<INS>`, or `<DUP>`; they cannot provide the concrete base changes
needed for background probe contexts.

Unlike Mykrobe 1, Mykrobe2 does not require or inspect `GT` and `GT_CONF` VCF
fields for background variants.

For example, this minimal VCF record is used as the background variant `G42T`:

```text
ref	42	.	G	T	.	PASS	.
```

For TB background SNP datasets, use the VCF files provided at
<https://figshare.com/articles/dataset/Mykrobe_TB_panel_background_variants/19582597>.

## Make probes with background context

Pass each background VCF with `--background-vcf` while making the probes:

```bash
mykrobe2 make-probes reference.fa \
  --text-file variants.tsv \
  --background-vcf background1.vcf \
  --background-vcf background2.vcf \
  > probes.fa
```

For many VCF files, place one filename per line in a text file (blank lines and
lines beginning with `#` are ignored), then use:

```bash
mykrobe2 make-probes reference.fa \
  --text-file variants.tsv \
  --background-vcf-list background-vcfs.txt \
  > probes.fa
```

The same options work with `--variants` and `--vcf` input. Continue the normal
custom-panel workflow afterwards: package `probes.fa`, the optional AMR JSON, and
the optional lineage JSON with `mykrobe2 index`, then use the resulting
`.panelindex` with `mykrobe2 predict --index`.

Background variants are an input to `make-probes`, not to `index` or `predict`.
If the background catalogue changes, regenerate the probes and rebuild the index.
