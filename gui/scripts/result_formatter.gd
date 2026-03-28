extends RefCounted
class_name ResultFormatter

const FIRST_LINE_DRUGS := [
	"Isoniazid",
	"Rifampicin",
	"Ethambutol",
	"Pyrazinamide",
]

const SECOND_LINE_DRUGS := [
	"Ofloxacin",
	"Moxifloxacin",
	"Ciprofloxacin",
	"Streptomycin",
	"Amikacin",
	"Capreomycin",
	"Kanamycin",
]

func format_all_tab(sample: String, parsed: Variant) -> Dictionary:
	var sample_data := _extract_sample(sample, parsed)
	if sample_data.is_empty():
		return {"susceptible": "", "resistant": ""}
	var susceptibility: Dictionary = sample_data.get("susceptibility", {})
	var susceptible: PackedStringArray = []
	var resistant: PackedStringArray = []
	for drug in _sorted_drugs(susceptibility):
		var predict := _predict_for_drug(susceptibility, drug)
		if predict == "R":
			resistant.append(str(drug))
		elif predict == "S":
			susceptible.append(str(drug))
	return {
		"susceptible": "\n".join(susceptible),
		"resistant": "\n".join(resistant),
	}

func format_drugs_tab(sample: String, parsed: Variant) -> Dictionary:
	var sample_data := _extract_sample(sample, parsed)
	if sample_data.is_empty():
		return {"first_line": "", "second_line": ""}
	var susceptibility: Dictionary = sample_data.get("susceptibility", {})
	return {
		"first_line": _format_drug_group(FIRST_LINE_DRUGS, susceptibility),
		"second_line": _format_drug_group(SECOND_LINE_DRUGS, susceptibility),
	}

func format_species_tab(sample: String, parsed: Variant) -> String:
	var sample_data := _extract_sample(sample, parsed)
	if sample_data.is_empty():
		return ""
	var phylo: Dictionary = sample_data.get("phylogenetics", {})
	var species_name := _best_phylo_name(phylo.get("species", {}))
	var lineage_name := _best_phylo_name(phylo.get("lineage", {}))
	if lineage_name != "Unknown":
		return "%s (%s)" % [species_name, lineage_name]
	return species_name

func format_evidence_tab(sample: String, parsed: Variant) -> String:
	var sample_data := _extract_sample(sample, parsed)
	if sample_data.is_empty():
		return ""
	var susceptibility: Dictionary = sample_data.get("susceptibility", {})
	var blocks: PackedStringArray = []
	for drug in _sorted_drugs(susceptibility):
		var drug_call: Dictionary = susceptibility.get(drug, {})
		if str(drug_call.get("predict", "")) != "R":
			continue
		blocks.append("[b][color=#3987b5]%s[/color][/b]" % str(drug).to_upper())
		blocks.append("")
		var called_by_variant: Variant = drug_call.get("called_by", {})
		if typeof(called_by_variant) != TYPE_DICTIONARY or Dictionary(called_by_variant).is_empty():
			blocks.append("No evidence details available.")
			blocks.append("")
			continue
		var called_by: Dictionary = called_by_variant
		var keys := called_by.keys()
		keys.sort()
		for key in keys:
			var evidence_variant: Variant = called_by.get(key, {})
			if typeof(evidence_variant) != TYPE_DICTIONARY:
				continue
			var evidence: Dictionary = evidence_variant
			var variant_name := str(evidence.get("variant", key))
			if variant_name == "null" or variant_name == "":
				variant_name = str(key)
			blocks.append("Resistance mutation found: %s" % variant_name)
			var info_variant: Variant = evidence.get("info", {})
			if typeof(info_variant) == TYPE_DICTIONARY:
				var info: Dictionary = info_variant
				var median_depth_variant: Variant = info.get("median_depth", null)
				var reference_depth_variant: Variant = info.get("reference_median_depth", null)
				if median_depth_variant != null:
					blocks.append("Depth %s on the resistant allele" % _display_number(median_depth_variant))
				if reference_depth_variant != null:
					blocks.append("Depth %s on the susceptible allele" % _display_number(reference_depth_variant))
			blocks.append("")
		blocks.append("")
	if blocks.is_empty():
		return "No resistant evidence found."
	return "\n".join(blocks).strip_edges()

func _format_drug_group(drugs: Array, susceptibility: Dictionary) -> String:
	var lines: PackedStringArray = []
	for drug in drugs:
		if not susceptibility.has(drug):
			continue
		var predict := _predict_for_drug(susceptibility, drug)
		match predict:
			"R":
				lines.append("%s [color=#f55a32]▲ RESISTANT[/color]" % drug)
			"S":
				lines.append("%s [color=#78b13f]● SUSCEPTIBLE[/color]" % drug)
			_:
				lines.append("%s [color=#7a7a7a]• NO CALL[/color]" % drug)
	return "\n".join(lines)

func _sorted_drugs(susceptibility: Dictionary) -> Array:
	var preferred_order: Array = FIRST_LINE_DRUGS.duplicate()
	preferred_order.append_array(SECOND_LINE_DRUGS)
	var seen := {}
	var ordered: Array = []
	for drug in preferred_order:
		if susceptibility.has(drug):
			ordered.append(drug)
			seen[drug] = true
	var remaining := susceptibility.keys()
	remaining.sort()
	for drug in remaining:
		if not seen.has(drug):
			ordered.append(drug)
	return ordered

func _predict_for_drug(susceptibility: Dictionary, drug: Variant) -> String:
	var drug_data_variant: Variant = susceptibility.get(drug, {})
	if typeof(drug_data_variant) != TYPE_DICTIONARY:
		return "?"
	var drug_data: Dictionary = drug_data_variant
	return str(drug_data.get("predict", "?"))

func _display_number(value: Variant) -> String:
	if typeof(value) == TYPE_FLOAT:
		var f := float(value)
		if is_equal_approx(f, round(f)):
			return "%d" % int(round(f))
		return "%0.2f" % f
	return str(value)

func _extract_sample(sample: String, parsed: Variant) -> Dictionary:
	if typeof(parsed) != TYPE_DICTIONARY:
		return {}
	var root: Dictionary = parsed
	if not root.has(sample):
		if root.keys().is_empty():
			return {}
		sample = str(root.keys()[0])
	return root.get(sample, {})

func _best_phylo_name(section: Variant) -> String:
	if typeof(section) != TYPE_DICTIONARY:
		return "Unknown"
	var d: Dictionary = section
	for key in d.keys():
		if str(key) != "Unknown" and typeof(d.get(key)) == TYPE_DICTIONARY:
			return str(key)
	return "Unknown"
