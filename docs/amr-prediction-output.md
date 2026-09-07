# AMR prediction output

This page describes the output of `mykrobe2 predict`.

## Output options

By default, `mykrobe2 predict` writes results in JSON format. An output filename must be provided using the `--output` option, for example:

```bash
mykrobe2 predict [options] --output results.json
```

JSON output contains detailed information about the AMR, species and lineage calls, and is suitable for further processing with scripts.

CSV output is also available, providing a simpler tabular summary that can be loaded into Excel or other spreadsheet software. The output format is controlled using `--format`. The available options are `json` (the default), `csv`, and `json_and_csv`.

For example, to write CSV output:

```bash
mykrobe2 predict [options] --format csv --output results.csv
```

See {ref}`AMR prediction examples <examples-of-basic-usage>` for complete
command-line examples.

## JSON file

The general format of the JSON is:

```json
{
  "sample_name": {
    "susceptibility": { ... AMR call information ... },
    "phylogenetics": { ... species/lineage information ... },
    "kmer": <integer>,
    "probe_sets": [ ... list of probe files ... ],
    "files": [ ... list of input reads files ... ],
    "version": { ... version information ... },
    "genotype_model": "name of genotype model"
  }
}
```

Susceptibility dictionary example:

```json
"Ofloxacin": {"predict": "S"},
"Streptomycin": {
  "predict": "R",
  "called_by": {
    "rpsL_K43R-AAG781686AGG": {
      "variant": null,
      "genotype": [1, 1],
      "genotype_likelihoods": [-3150.49, -99999999, -18.41],
      "info": {
        "coverage": {
          "reference": {
            "percent_coverage": 0.0,
            "median_depth": 0,
            "min_non_zero_depth": 0,
            "kmer_count": 0,
            "klen": 21
          },
          "alternate": {
            "percent_coverage": 100.0,
            "median_depth": 26,
            "min_non_zero_depth": 24,
            "kmer_count": 513,
            "klen": 20
          }
        },
        "expected_depths": [25],
        "contamination_depths": [],
        "filter": [],
        "conf": 3132
      },
      "_cls": "Call.VariantCall"
    }
  }
}
```

In this example, the sample is called as susceptible to ofloxacin and resistant to streptomycin. Details of the variant call that triggered the streptomycin resistance call are also provided. The reference allele had zero coverage, whereas the alternate allele had 100% coverage at a median depth of 26. The genotype call `[1, 1]` is homozygous for the alternate allele (`[0, 1]` would indicate a heterozygous call).

Phylogenetics dictionary example, where the species is `tuberculosis`:

```json
"phylo_group": {
  "Mycobacterium_tuberculosis_complex": {
    "percent_coverage": 99.544,
    "median_depth": 61.0
  }
},
"sub_complex": {
  "Unknown": {
    "percent_coverage": -1,
    "median_depth": -1
  }
},
"species": {
  "Mycobacterium_tuberculosis": {
    "percent_coverage": 98.328,
    "median_depth": 54.0
  }
},
"lineage": {
  "lineage": ["lineage2.2.9"],
  "calls_summary": {
    "lineage2.2.9": {
      "good_nodes": 3,
      "lineage_depth": 3,
      "genotypes": {
        "lineage2": 1,
        "lineage2.2": 1,
        "lineage2.2.9": 1
      }
    }
  },
  "calls": {
    "lineage2.2.9": {
      "lineage2": {
        "G497491A": { ... call info in same format as above call example ... }
      },
      "lineage2.2": {
        "G2505085A": { ... call info in same format as above call example ... }
      },
      "lineage2.2.9": {
        "G4086T": { ... call info in same format as above call example ... }
      }
    }
  }
}
```

In this example, the lineage was identified as `lineage2.2.9` (the entry `"lineage": ["lineage2.2.9"]`). If the sample were mixed, the list would have more than one entry, for example `["lineage2.2.9", "lineage4.10"]`.

The `calls_summary` section gives a high-level summary of the evidence for the lineage. `2.2.9` represents three levels in the lineage tree, as shown by `"lineage_depth": 3`. All three nodes in the tree agree with the lineage call, as shown by `"good_nodes": 3`. The genotype calls for all three nodes also agree with the lineage (all values in the `"genotypes"` dictionary are `1`).

Finally, the `calls` dictionary contains the full details of the genotype calls. These are omitted here because they have the same format as the resistance call example above.

MTB example where the species is not `tuberculosis`:

```json
"phylo_group": {
  "Mycobacterium_tuberculosis_complex": {
    "percent_coverage": 99.686,
    "median_depth": 97.0
  }
},
"sub_complex": {
  "Unknown": {
    "percent_coverage": -1,
    "median_depth": -1
  }
},
"species": {
  "Mycobacterium_africanum": {
    "percent_coverage": 59.351,
    "median_depth": 169
  }
},
"lineage": {
  "Unknown": {
    "percent_coverage": -1,
    "median_depth": -1
  }
}
```

## CSV file

The CSV output has the following columns:

| Column Name | Description |
|-------------|-------------|
| sample | Sample name, as given when running `mykrobe2 predict` |
| drug | Name of drug |
| susceptibility | Resistance call: "R"=resistant, "S"=susceptible, "r"=minority resistance detected |
| variants | Name of variant that caused the resistance call. See note 1 below for details |
| genes | Name of gene identified that caused the resistance call |
| mykrobe_version | Version of Mykrobe |
| files | Names of input read files |
| probe_sets | Names of probe sets used for AMR/lineage calls |
| genotype_model | Name of the genotyping model used to make ref/alt/heterozygous calls |
| kmer_size | k-mer length used in probes |
| phylo_group | High-level taxonomic identification, for example "Mycobacterium_tuberculosis_complex" |
| species | Species identified in the sample |
| lineage | Lineage identified in the sample. Mixed samples will have more than one entry |
| phylo_group_per_covg | Percentage of the `phylo_group` probe that has any coverage |
| species_per_covg | Percentage of the `species` probe that has any coverage |
| lineage_per_covg | Percentage of the `lineage` probe that has any coverage (see note 3 below) |
| phylo_group_depth | Average depth across the `phylo_group` probe |
| species_depth | Average depth across the `species` probe |
| lineage_depth | Average depth across the `lineage` probe (see note 3 below) |

Notes:

1. The `variants` column usually contains entries in the format `<gene>_<amino acid change>-<dna change>:<ref depth>:<alt depth>:<genotype confidence>`. For example, `rpsL_K43R-AAG781686AGG:0:513:3132` represents a K to R amino acid change at position 43 in the `rpsL` gene, with a nucleotide change from AAG to AGG at position 781686 in the genome. There is a reference depth of 0, an alternative allele depth of 513, and a genotype confidence of 3132.

2. Variants upstream of a gene have a negative position and are of the form `<gene>_<dna change>:<ref depth>:<alt depth>:<genotype confidence>` (an amino acid change is not applicable in this case). For example, `pncA_T-12C` represents a T to C nucleotide change 12 bp before the start of the `pncA` gene.

3. `lineage_per_covg` and `lineage_depth` are likely to be "NA", despite a lineage being identified. This happens when the lineage is called using a hierarchical scheme, such as lineage1, lineage1.1, etc., because there is no single probe for which to report coverage and depth. Instead, genotype calls across the lineage tree are used to determine the final lineage. The details can be found in the JSON output (see examples above), but are too complex for the CSV output.
