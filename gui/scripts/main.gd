extends Control

@onready var sample_edit: LineEdit = $RootMargin/RootVBox/BodySplit/FormPanel/FormMargin/FormVBox/SampleRow/SampleEdit
@onready var reads_edit: LineEdit = $RootMargin/RootVBox/BodySplit/FormPanel/FormMargin/FormVBox/ReadsRow/ReadsPicker/ReadsEdit
@onready var panels_dir_edit: LineEdit = $RootMargin/RootVBox/BodySplit/FormPanel/FormMargin/FormVBox/PanelsRow/PanelsPicker/PanelsDirEdit
@onready var species_edit: LineEdit = $RootMargin/RootVBox/BodySplit/FormPanel/FormMargin/FormVBox/SpeciesRow/SpeciesEdit
@onready var panel_edit: LineEdit = $RootMargin/RootVBox/BodySplit/FormPanel/FormMargin/FormVBox/PanelRow/PanelEdit
@onready var output_edit: LineEdit = $RootMargin/RootVBox/BodySplit/FormPanel/FormMargin/FormVBox/OutputRow/OutputPicker/OutputEdit
@onready var report_all_calls_check: CheckBox = $RootMargin/RootVBox/BodySplit/FormPanel/FormMargin/FormVBox/FlagsGrid/ReportAllCallsCheck
@onready var ncbi_names_check: CheckBox = $RootMargin/RootVBox/BodySplit/FormPanel/FormMargin/FormVBox/FlagsGrid/NCBINamesCheck
@onready var ont_check: CheckBox = $RootMargin/RootVBox/BodySplit/FormPanel/FormMargin/FormVBox/FlagsGrid/ONTCheck
@onready var guess_method_check: CheckBox = $RootMargin/RootVBox/BodySplit/FormPanel/FormMargin/FormVBox/FlagsGrid/GuessMethodCheck
@onready var run_button: Button = $RootMargin/RootVBox/BodySplit/FormPanel/FormMargin/FormVBox/ButtonsRow/RunButton
@onready var status_label: Label = $RootMargin/RootVBox/BodySplit/FormPanel/FormMargin/FormVBox/StatusLabel
@onready var summary_text: RichTextLabel = $RootMargin/RootVBox/BodySplit/ResultsPanel/ResultsMargin/ResultsVBox/ResultsTabs/SummaryTab/SummaryText
@onready var raw_json_text: TextEdit = $RootMargin/RootVBox/BodySplit/ResultsPanel/ResultsMargin/ResultsVBox/ResultsTabs/RawJSONTab/RawJSONText
@onready var reads_dialog: FileDialog = $ReadsDialog
@onready var panels_dir_dialog: FileDialog = $PanelsDirDialog
@onready var output_dialog: FileDialog = $OutputDialog

func _ready() -> void:
	panels_dir_edit.text = _default_panels_dir()
	status_label.text = "Ready."
	summary_text.text = "Run a sample to see summary output here."
	raw_json_text.text = ""

func _on_reads_browse_pressed() -> void:
	reads_dialog.popup_centered_ratio(0.7)

func _on_panels_browse_pressed() -> void:
	panels_dir_dialog.popup_centered_ratio(0.7)

func _on_output_browse_pressed() -> void:
	output_dialog.popup_centered_ratio(0.7)

func _on_reads_dialog_file_selected(path: String) -> void:
	reads_edit.text = path
	if sample_edit.text.strip_edges() == "" or sample_edit.text == "sample":
		sample_edit.text = path.get_file().get_basename()

func _on_panels_dir_dialog_dir_selected(path: String) -> void:
	panels_dir_edit.text = path

func _on_output_dialog_file_selected(path: String) -> void:
	output_edit.text = path

func _on_clear_button_pressed() -> void:
	status_label.text = "Ready."
	summary_text.text = "Run a sample to see summary output here."
	raw_json_text.text = ""

func _on_run_button_pressed() -> void:
	var sample := sample_edit.text.strip_edges()
	var reads_path := reads_edit.text.strip_edges()
	var panels_dir := panels_dir_edit.text.strip_edges()
	var species := species_edit.text.strip_edges()

	if sample == "":
		_set_status("Sample name is required.")
		return
	if reads_path == "":
		_set_status("Reads file is required.")
		return
	if panels_dir == "":
		_set_status("Panels directory is required.")
		return
	if species == "":
		_set_status("Species is required.")
		return

	var binary_path := _resolve_binary_path()
	if binary_path == "":
		_set_status("Could not find mykrobe2 binary. Set MYKROBE2_BINARY or provide a bundled binary.")
		return

	var output_path := output_edit.text.strip_edges()
	if output_path == "":
		output_path = _temporary_output_path(sample)

	var args := PackedStringArray([
		"predict",
		"--sample", sample,
		"--seq", reads_path,
		"--species", species,
		"--panels_dir", panels_dir,
		"--output", output_path,
		"--format", "json",
	])
	if panel_edit.text.strip_edges() != "":
		args.append_array(["--panel", panel_edit.text.strip_edges()])
	if report_all_calls_check.button_pressed:
		args.append("--report_all_calls")
	if ncbi_names_check.button_pressed:
		args.append("--ncbi_names")
	if ont_check.button_pressed:
		args.append("--ont")
	if guess_method_check.button_pressed:
		args.append("--guess_sequence_method")

	run_button.disabled = true
	_set_status("Running mykrobe2 predict...")
	await get_tree().process_frame

	var output_lines: Array = []
	var exit_code := OS.execute(binary_path, args, output_lines, true)
	run_button.disabled = false

	if exit_code != 0:
		var joined := "\n".join(output_lines)
		_set_status("mykrobe2 failed with exit code %d.\n%s" % [exit_code, joined])
		return

	if not FileAccess.file_exists(output_path):
		_set_status("Predict completed but no JSON output was found at %s." % output_path)
		return

	var file := FileAccess.open(output_path, FileAccess.READ)
	if file == null:
		_set_status("Could not read output JSON at %s." % output_path)
		return
	var text := file.get_as_text()
	file.close()

	var parsed = JSON.parse_string(text)
	if parsed == null:
		raw_json_text.text = text
		_set_status("Predict completed, but JSON parsing failed.")
		return

	raw_json_text.text = JSON.stringify(parsed, "\t")
	summary_text.text = _format_summary(sample, parsed)
	_set_status("Completed successfully using %s." % binary_path)

func _set_status(message: String) -> void:
	status_label.text = message

func _format_summary(sample: String, parsed: Variant) -> String:
	if typeof(parsed) != TYPE_DICTIONARY:
		return "Unexpected JSON output."
	var root: Dictionary = parsed
	if not root.has(sample):
		if root.keys().is_empty():
			return "No sample output found."
		sample = str(root.keys()[0])
	var sample_data: Dictionary = root.get(sample, {})

	var lines: PackedStringArray = []
	lines.append("Sample: %s" % sample)

	var phylo: Dictionary = sample_data.get("phylogenetics", {})
	lines.append("")
	lines.append("Phylogenetics")
	lines.append("Species: %s" % _best_phylo_name(phylo.get("species", {})))
	lines.append("Lineage: %s" % _best_phylo_name(phylo.get("lineage", {})))

	var susceptibility: Dictionary = sample_data.get("susceptibility", {})
	var drugs := susceptibility.keys()
	drugs.sort()
	lines.append("")
	lines.append("Susceptibility")
	for drug in drugs:
		var drug_data: Dictionary = susceptibility.get(drug, {})
		lines.append("%s: %s" % [str(drug), str(drug_data.get("predict", "?"))])

	return "\n".join(lines)

func _best_phylo_name(section: Variant) -> String:
	if typeof(section) != TYPE_DICTIONARY:
		return "Unknown"
	var d: Dictionary = section
	for key in d.keys():
		if str(key) != "Unknown":
			return str(key)
	return "Unknown"

func _temporary_output_path(sample: String) -> String:
	var base := sample.strip_edges()
	if base == "":
		base = "sample"
	return OS.get_user_data_dir().path_join("%s.predict.json" % base)

func _resolve_binary_path() -> String:
	var from_env := OS.get_environment("MYKROBE2_BINARY").strip_edges()
	if from_env != "" and FileAccess.file_exists(from_env):
		return from_env

	var candidates: PackedStringArray = []
	if OS.has_feature("editor"):
		candidates.append(ProjectSettings.globalize_path("res://../build/%s" % _binary_name()))
		candidates.append(ProjectSettings.globalize_path("res://../build/mykrobe2"))

	var exec_dir := OS.get_executable_path().get_base_dir()
	candidates.append(exec_dir.path_join("bin").path_join(_platform_triplet()).path_join(_binary_name()))
	candidates.append(exec_dir.path_join(_binary_name()))

	for candidate in candidates:
		if FileAccess.file_exists(candidate):
			return candidate
	return ""

func _binary_name() -> String:
	if OS.get_name() == "Windows":
		return "mykrobe2.exe"
	return "mykrobe2"

func _platform_triplet() -> String:
	var arch := "amd64"
	if OS.has_feature("arm64"):
		arch = "arm64"
	elif OS.has_feature("x86_64"):
		arch = "amd64"

	match OS.get_name():
		"macOS":
			return "darwin-%s" % arch
		"Windows":
			return "windows-%s" % arch
		_:
			return "linux-%s" % arch

func _default_panels_dir() -> String:
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
