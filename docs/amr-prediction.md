# AMR prediction

This page describes the use of `mykrobe2 predict` to make AMR predictions on the species supported by Mykrobe - at the time of writing these are
_Mycobacterium tuberculosis_, _Staphylococcus aureus_, _Shigella sonnei_, _Shigella flexneri_, _Salmonella enterica_ serotype Paratyphi B, and _Salmonella_ Typhi. You can see the available panels by running:
```
mykrobe2 panels describe
```
If no panels are installed, then run:
```
mykrobe2 panels update-metadata
mykrobe2 panels update-species all
```


When running on _Shigella sonnei_ samples, the output should be post-processed as described at [https://github.com/katholt/sonneityping](https://github.com/katholt/sonneityping).
Details of the _S. sonnei_ genotyping scheme are available in the paper [Hawkey et al, 2021, Nature Communications](https://www.nature.com/articles/s41467-021-22700-4).


(examples-of-basic-usage)=

## Examples of basic usage

Run on a _Mycobacterium tuberculosis_ sample with one FASTQ file as input, writing the results to a comma-delimited file:
```
mykrobe2 predict --species tb --sample sample_name -i reads.fq --output out.json
```
Replace `sample_name` with the name of your sample - whatever is used here will appear in the output.
Replace `tb` with `staph` or `sonnei` for _Staphylococcus aureus_ or _Shigella sonnei_ samples.

As above, but the input is two gzipped FASTQ files:
```
mykrobe2 predict -S tb -s sample_name -i reads_1.fq.gz -i reads_2.fq.gz -o out.json
```

As above, but the input is a BAM file:
```
mykrobe2 predict -S tb -s sample_name -i reads.bam -o out.json
```

The default output format is JSON, which is detailed. For essential information on the lineage of the sample (Mtb and sonnei only), and the AMR calls, there is the option to make a csv file:
```
mykrobe2 predict -S tb -s sample_name -i reads.fq --format csv -o out.csv
```

Make both a JSON and a CSV file, called `out.json` and `out.csv`:
```
mykrobe2 predict -S tb -s sample_name -i reads.fq --format json_and_csv -o out
```

## Important options

### Nanopore data

By default, the assumption is that the input reads are Illumina. If instead, you have nanopore data, then use the option `--ont`. Example:
```
mykrobe2 predict -S tb -s sample_name --ont -i nanopore_reads.fq -o out.json
```

### Minor resistance calls

By default, if a variant call is identified where a significant minority (enough to trigger a heterozygous call) of the reads have the variant, then it triggers a resistance call, reported as a lowercase "r" in the output. An uppercase "R" is used for a normal resistance call where the majority of reads have the variant (homozygous call). Use the option `--ignore-minor-calls` to ignore these minor calls when predicting resistance. Example:
```
mykrobe2 predict -S tb -s sample_name --ignore-minor-calls -i reads.fq -o out.json
```

### Getting all call information

The default behaviour is to only report detailed call information when it is a non-reference call (and therefore causes a resistance call). For debugging, or other in-depth analysis, it can be useful to see all calls with the `--report-all-calls` option. If the output is in JSON format, this will add information for all calls in the panel into the output. Example:
```
mykrobe2 predict -S tb -s sample_name --report-all-calls -i reads.fq -o out.json
```

### Other options

The other options are more advanced, and we do not recommend using them unless you know what you are doing.


## Output

See [AMR prediction output](amr-prediction-output.md) for a description of the
output.
