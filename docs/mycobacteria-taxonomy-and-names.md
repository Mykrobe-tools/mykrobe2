# Mycobacteria taxonomy and names

This is a brief explanation of the mycobacteria taxonomic
names reported by Mykrobe when using the species `tb`.

## Background

There is no consensus as to the naming of mycobacterial genera and species.

In 2018 the taxonomy was emended [1], splitting the
genus _Mycobacterium_ into five separate genera. This
is now the taxonomy used by the NCBI.
This change has been debated in the literature; see, for example, [2] and [3], which make the case for retaining the previous names.
Unfortunately, differences in species definitions and tree topologies mean that there is no one-to-one correspondence between the ‘old’ (pre-2018) and ‘new’ names.
This is a problem for Mykrobe when calling
species, since it cannot consistently call both taxonomies
at the same time.

We use GTDB [4] to generate the latest (panel 202309) species probes.
GTDB uses "old" names, and has one or more genomes per species.
Each of those genomes has a corresponding "new" NCBI taxon name.
Usually these are the same, but not always.
For example, GTDB had
(at the time of writing) four _Mycobacterium algericum_
genomes. Two of these have the "new" name
_Mycolicibacter algericus_, and the other two have the names
_Mycolicibacter sinensis_ and _Mycobacterium novum_.



## How does Mykrobe report species?

### Old Mykrobe panels

Mykrobe panels pre-2023 (walker-2015, bradley-2015, 202001, 202010, 202206) all used the "old" names and taxonomy.
Although some of these panels were released after 2018, they use species probes that were originally developed before the taxonomy was changed.

### New Mykrobe panels

In September 2023, the Mykrobe species probes were updated.
The updated panel is called 202309. It uses GTDB as the primary
source of metadata, genomes, taxonomic names and taxonomy.
The default names reported by Mykrobe are those used by
GTDB (and are essentially the "old" names).

You can also ask Mykrobe to report the "new"/NCBI names.
However, since there is no direct lookup between old and new names, the best we can do is report the NCBI names associated with a given species.
Returning to the _Mycobacterium algericum_ example above, if Mykrobe reports this species, using `--ncbi-names` will add the following to the output:
```
"ncbi_names": {
    "Mycolicibacter_sinensis": 1,
    "Mycolicibacter_algericus": 2,
    "Mycobacterium_novum": 1
}
```
The values are the numbers of GTDB genomes for that species associated with each NCBI name.




## References

[1] Gupta et al, Phylogenomics and comparative genomic studies robustly support division of the genus Mycobacterium into an emended genus Mycobacterium and four novel genera, Front Microbiol 2018, 9: 67, https://doi.org/10.3389/fmicb.2018.00067

[2] Tortoli et al, Same meat, different gravy: ignore the new names of mycobacteria, European Respiratory Journal 2019 54: 1900795; https://doi.org/10.1183/13993003.00795-2019

[3] Meehan et al, Reconstituting the genus Mycobacterium, Int J Syst Evol Microbiol., 2021; 71(9): 004922, https://doi.org/10.1099%2Fijsem.0.004922

[4] Parks et al, GTDB: an ongoing census of bacterial and archaeal diversity through a phylogenetically consistent, rank normalized and complete genome-based taxonomy, Nucleic Acids Research, Volume 50, Issue D1, 7 January 2022, Pages D785–D794, https://doi.org/10.1093/nar/gkab776
