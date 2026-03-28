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
	var species_name := _display_phylo_name(_best_phylo_name(phylo.get("species", {}), "species"))
	var lineage_name := _best_phylo_name(phylo.get("lineage", {}), "lineage")
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
			blocks.append(_format_evidence_summary(evidence, str(key)))
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

func _format_evidence_summary(evidence: Dictionary, fallback_key: String) -> String:
	var variant_value := str(evidence.get("variant", "")).strip_edges()
	var fields := _variant_query_fields(variant_value)
	var gene := str(fields.get("gene", "")).strip_edges()
	var mut := str(fields.get("mut", "")).strip_edges()
	if mut == "" and fallback_key.contains("_"):
		var left_part := fallback_key.split("-", false, 1)[0]
		var bits := left_part.split("_", false, 1)
		if bits.size() == 2:
			gene = bits[0]
			mut = bits[1]
	if gene != "" and mut != "":
		return "Resistance mutation found: %s in gene %s" % [mut, gene]
	if mut != "":
		return "Resistance mutation found: %s" % mut
	if variant_value != "" and variant_value != "null":
		return "Resistance mutation found: %s" % variant_value
	return "Resistance mutation found: %s" % fallback_key

func _variant_query_fields(variant_value: String) -> Dictionary:
	var out := {}
	if variant_value == "":
		return out
	var qmark := variant_value.find("?")
	if qmark == -1 or qmark + 1 >= variant_value.length():
		return out
	var query := variant_value.substr(qmark + 1)
	for part in query.split("&", false):
		if not part.contains("="):
			continue
		var bits := part.split("=", false, 1)
		if bits.size() != 2:
			continue
		out[bits[0]] = bits[1].uri_decode()
	return out

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

func _best_phylo_name(section: Variant, list_key: String = "") -> String:
	if typeof(section) != TYPE_DICTIONARY:
		return "Unknown"
	var d: Dictionary = section
	if list_key != "":
		var names_variant: Variant = d.get(list_key, null)
		if typeof(names_variant) == TYPE_ARRAY:
			var names: Array = names_variant
			if not names.is_empty():
				return str(names[0])
		elif typeof(names_variant) == TYPE_STRING:
			var single_name := str(names_variant).strip_edges()
			if single_name != "":
				return single_name
	for key in d.keys():
		var key_name := str(key)
		if key_name in ["Unknown", "calls", "calls_summary", "ncbi_names", "lineage", "species"]:
			continue
		if typeof(d.get(key)) == TYPE_DICTIONARY:
			return str(key)
	return "Unknown"

func _display_phylo_name(name: String) -> String:
	if name == "" or name == "Unknown":
		return "Unknown"
	return name.replace("_", " ")
