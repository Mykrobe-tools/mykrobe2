extends RefCounted
class_name GUIHelpers

const TB_LEGACY_PANELS_LAST := {
	"bradley-2015": 0,
	"walker-2015": 1,
}

func load_texture(path: String) -> Texture2D:
	if path.to_lower().ends_with(".png"):
		var file := FileAccess.open(path, FileAccess.READ)
		if file == null:
			return null
		var bytes := file.get_buffer(file.get_length())
		file.close()
		var image := Image.new()
		if image.load_png_from_buffer(bytes) != OK:
			return null
		return ImageTexture.create_from_image(image)
	var loaded: Variant = load(path)
	if loaded is Texture2D:
		return loaded
	var image := Image.load_from_file(path)
	if image == null:
		return null
	return ImageTexture.create_from_image(image)

func temporary_output_path(sample: String) -> String:
	var base := sample.strip_edges()
	if base == "":
		base = "sample"
	return OS.get_user_data_dir().path_join("%s.predict.json" % base)

func default_panels_dir() -> String:
	var data_home := OS.get_environment("MYKROBE_DATA_HOME").strip_edges()
	if data_home != "":
		return data_home.path_join("mykrobe2").path_join("panels")

	match OS.get_name():
		"macOS":
			return OS.get_environment("HOME").path_join("Library").path_join("Application Support").path_join("mykrobe2").path_join("panels")
		"Windows":
			var appdata := OS.get_environment("APPDATA").strip_edges()
			if appdata != "":
				return appdata.path_join("mykrobe2").path_join("panels")
			return OS.get_environment("USERPROFILE").path_join("AppData").path_join("Roaming").path_join("mykrobe2").path_join("panels")
		_:
			return OS.get_environment("HOME").path_join(".local").path_join("share").path_join("mykrobe2").path_join("panels")

func species_installed_marker_exists(panels_dir: String, species: String) -> bool:
	if species == "":
		return false
	return FileAccess.file_exists(panels_dir.path_join(species).path_join("manifest.json"))

func load_species_entries(binary_path: String, panels_dir: String) -> Array:
	if panels_dir == "":
		return []
	var species_list: Array = []
	if binary_path != "":
		var output_lines: Array = []
		var exit_code := OS.execute(binary_path, PackedStringArray([
			"panels",
			"describe",
			"--panels_dir", panels_dir,
			"--format", "json",
		]), output_lines, true)
		if exit_code == 0:
			var parsed = JSON.parse_string("\n".join(output_lines))
			if typeof(parsed) == TYPE_DICTIONARY:
				var root: Dictionary = parsed
				var cli_species_list: Variant = root.get("species", [])
				if typeof(cli_species_list) == TYPE_ARRAY:
					species_list = cli_species_list
	if species_list.is_empty():
		species_list = _load_species_entries_from_panels_dir(panels_dir)
	var entries: Array = []
	for item in species_list:
		if typeof(item) != TYPE_DICTIONARY:
			continue
		var entry: Dictionary = item
		var species_name := str(entry.get("species", "")).strip_edges()
		if species_name == "":
			continue
		entries.append(entry)
	return entries

func _load_species_entries_from_panels_dir(panels_dir: String) -> Array:
	var manifest_path := panels_dir.path_join("manifest.json")
	if not FileAccess.file_exists(manifest_path):
		return []
	var text := FileAccess.get_file_as_string(manifest_path)
	var parsed = JSON.parse_string(text)
	if typeof(parsed) != TYPE_DICTIONARY:
		return []
	var manifest: Dictionary = parsed
	var species_names := manifest.keys()
	species_names.sort()
	var entries: Array = []
	for species_name_variant in species_names:
		var species_name := str(species_name_variant)
		var entry_variant: Variant = manifest.get(species_name, {})
		if typeof(entry_variant) != TYPE_DICTIONARY:
			continue
		var entry: Dictionary = entry_variant
		var installed := false
		var installed_info: Dictionary = {}
		var installed_variant: Variant = entry.get("installed", null)
		if typeof(installed_variant) == TYPE_DICTIONARY:
			installed = true
			installed_info = installed_variant
		var latest_info: Dictionary = {}
		var latest_variant: Variant = entry.get("latest", null)
		if typeof(latest_variant) == TYPE_DICTIONARY:
			latest_info = latest_variant
		var species_entry: Dictionary = {
			"species": species_name,
			"installed": installed,
			"installed_version": str(installed_info.get("version", "None")),
			"installed_url": str(installed_info.get("url", "NA")),
			"latest_version": str(latest_info.get("version", "")),
			"latest_url": str(latest_info.get("url", "")),
			"default_panel": "",
			"panels": [],
		}
		if installed:
			var species_manifest_path := panels_dir.path_join(species_name).path_join("manifest.json")
			if FileAccess.file_exists(species_manifest_path):
				var species_manifest_text := FileAccess.get_file_as_string(species_manifest_path)
				var species_manifest_variant = JSON.parse_string(species_manifest_text)
				if typeof(species_manifest_variant) == TYPE_DICTIONARY:
					var species_manifest: Dictionary = species_manifest_variant
					species_entry["default_panel"] = str(species_manifest.get("default_panel", ""))
					var panels_variant: Variant = species_manifest.get("panels", {})
					if typeof(panels_variant) == TYPE_DICTIONARY:
						var panels_dict: Dictionary = panels_variant
						var panel_names := _sort_panel_names(species_name, panels_dict.keys())
						var panels: Array = []
						for panel_name_variant in panel_names:
							var panel_name := str(panel_name_variant)
							var panel_info_variant: Variant = panels_dict.get(panel_name, {})
							if typeof(panel_info_variant) != TYPE_DICTIONARY:
								continue
							var panel_info: Dictionary = panel_info_variant
							panels.append({
								"name": panel_name,
								"reference": str(panel_info.get("reference_genome", "")),
								"description": str(panel_info.get("description", "")),
							})
						species_entry["panels"] = panels
		entries.append(species_entry)
	return entries

func _sort_panel_names(species_name: String, panel_names: Array) -> Array:
	var names: Array = []
	for panel_name_variant in panel_names:
		names.append(str(panel_name_variant))
	names.sort_custom(func(a: String, b: String) -> bool:
		if species_name.to_lower() == "tb":
			var a_is_legacy := TB_LEGACY_PANELS_LAST.has(a)
			var b_is_legacy := TB_LEGACY_PANELS_LAST.has(b)
			if a_is_legacy != b_is_legacy:
				return not a_is_legacy
			if a_is_legacy and b_is_legacy:
				return int(TB_LEGACY_PANELS_LAST[a]) < int(TB_LEGACY_PANELS_LAST[b])
		return a > b
	)
	return names
