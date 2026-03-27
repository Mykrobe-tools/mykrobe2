extends RefCounted
class_name ResultFormatter

func format_overview(sample: String, parsed: Variant) -> String:
	var sample_data := _extract_sample(sample, parsed)
	if sample_data.is_empty():
		return "No sample output found."

	var lines: PackedStringArray = []
	lines.append("Sample: %s" % sample)

	var phylo: Dictionary = sample_data.get("phylogenetics", {})
	lines.append("")
	lines.append("Phylogenetics")
	lines.append("Species: %s" % _best_phylo_name(phylo.get("species", {})))
	lines.append("Lineage: %s" % _best_phylo_name(phylo.get("lineage", {})))
	lines.append("Phylo group: %s" % _best_phylo_name(phylo.get("phylo_group", {})))

	var susceptibility: Dictionary = sample_data.get("susceptibility", {})
	var counts := {"R": 0, "S": 0, "N": 0}
	for value in susceptibility.values():
		var drug_data: Dictionary = value
		var predict := str(drug_data.get("predict", "?"))
		if counts.has(predict):
			counts[predict] += 1

	lines.append("")
	lines.append("Susceptibility totals")
	lines.append("Resistant: %d" % counts["R"])
	lines.append("Susceptible: %d" % counts["S"])
	lines.append("No call: %d" % counts["N"])
	return "\n".join(lines)

func format_drugs(sample: String, parsed: Variant) -> String:
	var sample_data := _extract_sample(sample, parsed)
	if sample_data.is_empty():
		return ""
	var susceptibility: Dictionary = sample_data.get("susceptibility", {})
	var drugs := susceptibility.keys()
	drugs.sort()
	var lines: PackedStringArray = []
	for drug in drugs:
		var drug_data: Dictionary = susceptibility.get(drug, {})
		lines.append("%s: %s" % [str(drug), str(drug_data.get("predict", "?"))])
	return "\n".join(lines)

func format_species(sample: String, parsed: Variant) -> String:
	var sample_data := _extract_sample(sample, parsed)
	if sample_data.is_empty():
		return ""
	var phylo: Dictionary = sample_data.get("phylogenetics", {})
	var lines: PackedStringArray = []
	lines.append("Phylo group")
	lines.append_array(_format_phylo_section(phylo.get("phylo_group", {})))
	lines.append("")
	lines.append("Sub-complex")
	lines.append_array(_format_phylo_section(phylo.get("sub_complex", {})))
	lines.append("")
	lines.append("Species")
	lines.append_array(_format_phylo_section(phylo.get("species", {})))
	lines.append("")
	lines.append("Lineage")
	lines.append_array(_format_phylo_section(phylo.get("lineage", {})))
	return "\n".join(lines)

func format_evidence(sample: String, parsed: Variant) -> String:
	var sample_data := _extract_sample(sample, parsed)
	if sample_data.is_empty():
		return ""
	var lines: PackedStringArray = []
	var variant_calls: Dictionary = sample_data.get("variant_calls", {})
	var sequence_calls: Dictionary = sample_data.get("sequence_calls", {})
	var lineage_calls: Dictionary = sample_data.get("lineage_calls", {})
	lines.append("Variant calls: %d" % variant_calls.size())
	lines.append("Sequence calls: %d" % sequence_calls.size())
	lines.append("Lineage calls: %d" % lineage_calls.size())
	lines.append("")
	lines.append("Top lineage calls")
	var lineage_keys: Array = lineage_calls.keys()
	lineage_keys.sort()
	var limit: int = min(10, lineage_keys.size())
	for i in range(limit):
		var key = lineage_keys[i]
		var call: Dictionary = lineage_calls.get(key, {})
		var info: Dictionary = call.get("info", {})
		lines.append("%s: %s (conf=%s)" % [str(key), str(call.get("genotype", "?")), str(info.get("conf", "?"))])
	return "\n".join(lines)

func _extract_sample(sample: String, parsed: Variant) -> Dictionary:
	if typeof(parsed) != TYPE_DICTIONARY:
		return {}
	var root: Dictionary = parsed
	if not root.has(sample):
		if root.keys().is_empty():
			return {}
		sample = str(root.keys()[0])
	return root.get(sample, {})

func _format_phylo_section(section: Variant) -> PackedStringArray:
	var lines: PackedStringArray = []
	if typeof(section) != TYPE_DICTIONARY:
		lines.append("Unknown")
		return lines
	var d: Dictionary = section
	if d.is_empty():
		lines.append("Unknown")
		return lines
	var keys := d.keys()
	keys.sort()
	for key in keys:
		var item: Variant = d.get(key, null)
		if typeof(item) == TYPE_DICTIONARY:
			var item_dict: Dictionary = item
			lines.append("%s: coverage=%s depth=%s" % [
				str(key),
				str(item_dict.get("percent_coverage", "?")),
				str(item_dict.get("median_depth", "?")),
			])
		elif typeof(item) == TYPE_ARRAY:
			var values: Array = item
			var rendered: PackedStringArray = []
			for value in values:
				rendered.append(str(value))
			lines.append("%s: %s" % [str(key), ", ".join(rendered)])
		else:
			lines.append("%s: %s" % [str(key), str(item)])
	return lines

func _best_phylo_name(section: Variant) -> String:
	if typeof(section) != TYPE_DICTIONARY:
		return "Unknown"
	var d: Dictionary = section
	for key in d.keys():
		if str(key) != "Unknown" and typeof(d.get(key)) == TYPE_DICTIONARY:
			return str(key)
	return "Unknown"
