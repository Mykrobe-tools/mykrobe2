extends RefCounted
class_name GUIHelpers

func load_png_texture(path: String) -> Texture2D:
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
	if binary_path == "" or panels_dir == "":
		return []
	var output_lines: Array = []
	var exit_code := OS.execute(binary_path, PackedStringArray([
		"panels",
		"describe",
		"--panels_dir", panels_dir,
		"--format", "json",
	]), output_lines, true)
	if exit_code != 0:
		return []
	var parsed = JSON.parse_string("\n".join(output_lines))
	if typeof(parsed) != TYPE_DICTIONARY:
		return []
	var root: Dictionary = parsed
	var species_list: Variant = root.get("species", [])
	if typeof(species_list) != TYPE_ARRAY:
		return []
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
