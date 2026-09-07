# Mykrobe2

Antibiotic resistance prediction in minutes.


# Quick start

## Install

Download the desktop app or command-line tool from the [latest release](https://github.com/Mykrobe-tools/mykrobe2/releases/latest) that matches your operating system and architecture.


## Desktop app

* When you first run, it will download and index all the panel data. This will take several minutes.
* Choose the species and panel, drag and drop your sequence file(s), and press "Analyse sample"


## Command-line tool

Run these two commands first to install and index the panel data. It will take several minutes:

```bash
mykrobe2 panels update-metadata
mykrobe2 panels update-species all
```

## Run AMR prediction

Run on an _M. tuberculosis_ sample, making a JSON file of results:

```bash
mykrobe2 predict --sample my_sample_name \
  --species tb \
  --output out.json \
  --format json \
  --seq reads.fq.gz
```

For other species, change `tb` in the option `--species tb` to one of: `flexneri`, `paratyphiB`, `sonnei`, `staph`, `typhi`.

Moving from the previous command-line program? See
[Migrating from Mykrobe 1](migrating-from-mykrobe1.md) for the user-facing
workflow changes, particularly custom panels and background variants.


## Test on example data

You can test that `mykrobe2 predict` runs as expected by downloading and running
on a small set of simulated reads:

```sh
mykrobe2 download-test-reads test_reads.fq.gz
mykrobe2 predict -s SAMPLE -S tb -o out.json --format json -i test_reads.fq.gz
```

The test reads perfectly match the reference, except for
the isoniazid resistance-associated variant inhA I21T. You should see a section
like this in the output file `out.json`:

```json
"Isoniazid": {
    "predict": "R",
    "called_by": {
        "inhA_I21T-ATC1674262ACT": {
    ... etc
```

For more detail, see:

```{toctree}
:maxdepth: 2
:caption: Contents

install
desktop-app
amr-prediction
amr-prediction-output
built-in-panels
migrating-from-mykrobe1
custom-panels
custom-lineage-calling
background-variants
mycobacteria-taxonomy-and-names
citing-mykrobe
mykrobe1-vs-mykrobe2-benchmark
```
