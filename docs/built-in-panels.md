# Built-in panels

This page describes the panels included with Mykrobe, and the command `mykrobe2 panels`.

For the origin of each built-in panel, and links to the publications,
please see the [citing Mykrobe](citing-mykrobe.md) page.

## Quick start

After installing Mykrobe, run:

```bash
mykrobe2 panels update-metadata
mykrobe2 panels update-species all
```

This will install built-in panels, which can be used with `mykrobe2 predict`.

Keep reading if you want more details.

## Overview

At the time of writing, Mykrobe supports panels for _Mycobacterium tuberculosis_, _Staphylococcus aureus_, _Shigella sonnei_, _Shigella flexneri_, _Salmonella enterica_ serotype Paratyphi B, and _Salmonella_ Typhi.

The panel data are _not_ included with the source code. They must be downloaded after installation. Panel data can be updated without reinstalling Mykrobe.

## Download or update the panels

Run these two commands to get or update all supported panels:

```bash
mykrobe2 panels update-metadata
mykrobe2 panels update-species all
```

By default, panel files are stored under your home directory, with the exact location depending on your operating system. This location can be changed using the option `--panels-dir`, which is available for all `mykrobe2 panels` commands and for `mykrobe2 predict`. We recommend using the default location unless you have a reason to change it.

For example, to store the panel data in a directory called `panel_data`:

```bash
mykrobe2 panels --panels-dir panel_data update-metadata
mykrobe2 panels --panels-dir panel_data update-species all
```

## Panel information

Obtain information on the currently installed panels by running:

```bash
mykrobe2 panels describe
```

If no panels are installed, the output will simply be:

```text
No data
```

If panels are available but not installed (i.e., you have only run `mykrobe2 panels update-metadata`), the output will look like this:

```text
Gathering data from /your/panel/folder/data

Species summary:

Species	Update_available	Installed_version	Installed_url	Latest_version	Latest_url
flexneri	yes	None	NA	20250902	https://ndownloader.figshare.com/files/63190870
paratyphiB	yes	None	NA	20230627	https://figshare.com/ndownloader/files/43870968
sonnei	yes	None	NA	20210201	https://ndownloader.figshare.com/files/26274424
staph	yes	None	NA	20201001	https://ndownloader.figshare.com/files/24914930
tb	yes	None	NA	20230928	https://figshare.com/ndownloader/files/42494211
typhi	yes	None	NA	20240407	https://figshare.com/ndownloader/files/49527246
```

Running `mykrobe2 panels update-species all` will install all available updates. The output of `mykrobe2 panels describe` will then look something like this:

```text
Gathering data from /your/panel/folder/data

Species summary:

Species	Update_available	Installed_version	Installed_url	Latest_version	Latest_url
flexneri	no	20250902	https://ndownloader.figshare.com/files/63190870	20250902	https://ndownloader.figshare.com/files/63190870
paratyphiB	no	20230627	https://figshare.com/ndownloader/files/43870968	20230627	https://figshare.com/ndownloader/files/43870968
sonnei	no	20210201	https://ndownloader.figshare.com/files/26274424	20210201	https://ndownloader.figshare.com/files/26274424
staph	no	20201001	https://ndownloader.figshare.com/files/24914930	20201001	https://ndownloader.figshare.com/files/24914930
tb	no	20230928	https://figshare.com/ndownloader/files/42494211	20230928	https://figshare.com/ndownloader/files/42494211
typhi	no	20240407	https://figshare.com/ndownloader/files/49527246	20240407	https://figshare.com/ndownloader/files/49527246

flexneri default panel: 20250902
flexneri panels:
Panel	Reference	Description
20250902	AE005674	Genotyping panel for Shigella flexneri, including lineage and panel for variants in the quinolone resistance determining region. Described in Hawkey et al 2026.

... and then descriptions of the other species ...
```

## Panel choice and running Mykrobe predict

We recommend running `mykrobe2 predict` using the default panel. For `sonnei` and `staph`, there is currently only one panel. For `tb`, several panels are available.

To use a panel other than the default, specify it with the `--panel` option. For example, to use the older `walker-2015` panel:

```bash
mykrobe2 predict --sample sample_name --species tb --panel walker-2015 --seq reads.fastq
```
