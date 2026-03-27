extends Control

const LocalMykrobe2ManagerScript = preload("res://scripts/local_mykrobe2_manager.gd")

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
@onready var setup_panel: PanelContainer = $RootMargin/RootVBox/BodySplit/FormPanel/FormMargin/FormVBox/SetupPanel
@onready var setup_status_label: Label = $RootMargin/RootVBox/BodySplit/FormPanel/FormMargin/FormVBox/SetupPanel/SetupMargin/SetupVBox/SetupStatus
@onready var update_metadata_button: Button = $RootMargin/RootVBox/BodySplit/FormPanel/FormMargin/FormVBox/SetupPanel/SetupMargin/SetupVBox/SetupButtons/UpdateMetadataButton
@onready var install_species_button: Button = $RootMargin/RootVBox/BodySplit/FormPanel/FormMargin/FormVBox/SetupPanel/SetupMargin/SetupVBox/SetupButtons/InstallSpeciesButton
@onready var refresh_setup_button: Button = $RootMargin/RootVBox/BodySplit/FormPanel/FormMargin/FormVBox/SetupPanel/SetupMargin/SetupVBox/SetupButtons/RefreshSetupButton
@onready var setup_log_text: TextEdit = $RootMargin/RootVBox/BodySplit/FormPanel/FormMargin/FormVBox/SetupPanel/SetupMargin/SetupVBox/SetupLog
@onready var run_button: Button = $RootMargin/RootVBox/BodySplit/FormPanel/FormMargin/FormVBox/ButtonsRow/RunButton
@onready var status_label: Label = $RootMargin/RootVBox/BodySplit/FormPanel/FormMargin/FormVBox/StatusLabel
@onready var overview_text: RichTextLabel = $RootMargin/RootVBox/BodySplit/ResultsPanel/ResultsMargin/ResultsVBox/ResultsTabs/OverviewTab/OverviewText
@onready var drugs_text: RichTextLabel = $RootMargin/RootVBox/BodySplit/ResultsPanel/ResultsMargin/ResultsVBox/ResultsTabs/DrugsTab/DrugsText
@onready var species_text: RichTextLabel = $RootMargin/RootVBox/BodySplit/ResultsPanel/ResultsMargin/ResultsVBox/ResultsTabs/SpeciesTab/SpeciesText
@onready var evidence_text: TextEdit = $RootMargin/RootVBox/BodySplit/ResultsPanel/ResultsMargin/ResultsVBox/ResultsTabs/EvidenceTab/EvidenceText
@onready var raw_json_text: TextEdit = $RootMargin/RootVBox/BodySplit/ResultsPanel/ResultsMargin/ResultsVBox/ResultsTabs/RawJSONTab/RawJSONText
@onready var reads_dialog: FileDialog = $ReadsDialog
@onready var panels_dir_dialog: FileDialog = $PanelsDirDialog
@onready var output_dialog: FileDialog = $OutputDialog

var _local_mykrobe2_manager: RefCounted
var _setup_task_running := false
var _setup_task_pid := -1
var _setup_log_path := ""
var _setup_result_path := ""
var _setup_last_log_text := ""

func _ready() -> void:
	panels_dir_edit.text = _default_panels_dir()
	status_label.text = "Ready."
	_clear_results()
	_local_mykrobe2_manager = LocalMykrobe2ManagerScript.new()
	_local_mykrobe2_manager.configure("bin")
	get_viewport().files_dropped.connect(_on_files_dropped)
	_refresh_setup_state()
	_maybe_start_initial_panels_bootstrap()

func _process(_delta: float) -> void:
	_poll_setup_task()

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
	_refresh_setup_state()
	_maybe_start_initial_panels_bootstrap()

func _on_output_dialog_file_selected(path: String) -> void:
	output_edit.text = path

func _on_clear_button_pressed() -> void:
	status_label.text = "Ready."
	_clear_results()

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
	if not _species_installed_marker_exists(panels_dir, species):
		_set_status("Species panels are not installed yet. Use Panels Setup first.")
		_refresh_setup_state()
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

	_load_json_result(output_path, sample)
	if raw_json_text.text != "":
		_set_status("Completed successfully using %s." % binary_path)
	_refresh_setup_state()

func _load_json_result(path: String, preferred_sample: String = "sample") -> void:
	if not FileAccess.file_exists(path):
		_set_status("Result JSON was not found at %s." % path)
		return
	var file := FileAccess.open(path, FileAccess.READ)
	if file == null:
		_set_status("Could not open JSON file: %s." % path)
		return
	var text := file.get_as_text()
	file.close()

	var parsed = JSON.parse_string(text)
	if parsed == null:
		raw_json_text.text = text
		_set_status("JSON parsing failed for %s." % path)
		return

	_display_results(preferred_sample, parsed)

func _display_results(sample: String, parsed: Variant) -> void:
	raw_json_text.text = JSON.stringify(parsed, "\t")
	overview_text.text = _format_overview(sample, parsed)
	drugs_text.text = _format_drugs(sample, parsed)
	species_text.text = _format_species(sample, parsed)
	evidence_text.text = _format_evidence(sample, parsed)

func _clear_results() -> void:
	overview_text.text = "Run a sample to see summary output here."
	drugs_text.text = ""
	species_text.text = ""
	evidence_text.text = ""
	raw_json_text.text = ""

func _set_status(message: String) -> void:
	status_label.text = message

func _set_setup_status(message: String) -> void:
	setup_status_label.text = message

func _append_setup_log(message: String) -> void:
	if setup_log_text.text == "":
		setup_log_text.text = message
	else:
		setup_log_text.text += "\n" + message
	setup_log_text.scroll_vertical = setup_log_text.get_line_count()

func _extract_sample(sample: String, parsed: Variant) -> Dictionary:
	if typeof(parsed) != TYPE_DICTIONARY:
		return {}
	var root: Dictionary = parsed
	if not root.has(sample):
		if root.keys().is_empty():
			return {}
		sample = str(root.keys()[0])
	return root.get(sample, {})

func _format_overview(sample: String, parsed: Variant) -> String:
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

func _format_drugs(sample: String, parsed: Variant) -> String:
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

func _format_species(sample: String, parsed: Variant) -> String:
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

func _format_evidence(sample: String, parsed: Variant) -> String:
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

func _temporary_output_path(sample: String) -> String:
	var base := sample.strip_edges()
	if base == "":
		base = "sample"
	return OS.get_user_data_dir().path_join("%s.predict.json" % base)

func _resolve_binary_path() -> String:
	var from_env := OS.get_environment("MYKROBE2_BINARY").strip_edges()
	if from_env != "" and FileAccess.file_exists(from_env):
		return from_env

	if _local_mykrobe2_manager != null and _local_mykrobe2_manager.ensure_local_binary_installed():
		return _local_mykrobe2_manager.installed_binary_path()
	return ""

func _on_update_metadata_button_pressed() -> void:
	_start_panels_task([
		{
			"label": "Updating panel metadata",
			"args": PackedStringArray([
				"panels",
				"update_metadata",
				"--panels_dir", panels_dir_edit.text.strip_edges(),
			]),
		},
	], "Updating panel metadata...", "Panel metadata updated.")

func _on_install_species_button_pressed() -> void:
	var species := species_edit.text.strip_edges()
	if species == "":
		_set_status("Set a species first before installing panels.")
		return
	_start_panels_task([
		{
			"label": "Installing species panels for %s" % species,
			"args": PackedStringArray([
				"panels",
				"update_species",
				"--panels_dir", panels_dir_edit.text.strip_edges(),
				species,
			]),
		},
	], "Installing species panels for %s..." % species, "Species panels installed for %s." % species)

func _on_refresh_setup_button_pressed() -> void:
	_refresh_setup_state()

func _start_panels_task(commands: Array, status_prefix: String, success_status: String) -> void:
	if _setup_task_running:
		_set_status("Panel setup is already running.")
		return
	var binary_path := _resolve_binary_path()
	if binary_path == "":
		_set_status("Could not find mykrobe2 binary for panel setup.")
		return
	var panels_dir := panels_dir_edit.text.strip_edges()
	if panels_dir == "":
		_set_status("Panels directory is required for panel setup.")
		return
	_set_status(status_prefix)
	_set_setup_status(status_prefix)
	_set_setup_busy(true)
	setup_log_text.text = ""
	_setup_log_path = OS.get_user_data_dir().path_join("panels-setup.log")
	if FileAccess.file_exists(_setup_log_path):
		DirAccess.remove_absolute(_setup_log_path)
	_setup_result_path = OS.get_user_data_dir().path_join("panels-setup.result")
	if FileAccess.file_exists(_setup_result_path):
		DirAccess.remove_absolute(_setup_result_path)
	_setup_last_log_text = ""
	_setup_task_running = true
	_setup_task_pid = _start_panels_process(binary_path, commands, success_status, _setup_log_path, _setup_result_path)
	if _setup_task_pid == -1:
		_setup_task_running = false
		_set_setup_busy(false)
		_set_status("Could not start background panel setup.")
		_set_setup_status("Could not start panel setup.")

func _write_setup_log_line(log_path: String, message: String) -> void:
	var open_mode := FileAccess.READ_WRITE if FileAccess.file_exists(log_path) else FileAccess.WRITE_READ
	var file := FileAccess.open(log_path, open_mode)
	if file == null:
		return
	file.seek_end()
	file.store_line(message)
	file.close()

func _poll_setup_task() -> void:
	if not _setup_task_running:
		return
	_refresh_setup_log_from_disk()
	if not FileAccess.file_exists(_setup_result_path):
		return
	_setup_task_running = false
	_setup_task_pid = -1
	_set_setup_busy(false)
	_refresh_setup_log_from_disk()
	var result := _read_setup_result(_setup_result_path)
	if result.get("success", false):
		_set_status(str(result.get("status", "Panel setup complete.")))
		_refresh_setup_state()
		return
	var error_message := str(result.get("error", "Panel setup failed."))
	_set_status(error_message)
	_set_setup_status("Panel setup failed.")

func _start_panels_process(binary_path: String, commands: Array, success_status: String, log_path: String, result_path: String) -> int:
	var script_path := OS.get_user_data_dir().path_join("panels-setup-script")
	if OS.get_name() == "Windows":
		script_path += ".cmd"
		_write_windows_setup_script(script_path, binary_path, commands, success_status, log_path, result_path)
		return OS.create_process("cmd.exe", PackedStringArray(["/C", script_path]), false)
	script_path += ".sh"
	_write_posix_setup_script(script_path, binary_path, commands, success_status, log_path, result_path)
	return OS.create_process("/bin/bash", PackedStringArray([script_path]), false)

func _write_posix_setup_script(script_path: String, binary_path: String, commands: Array, success_status: String, log_path: String, result_path: String) -> void:
	var lines: PackedStringArray = [
		"#!/usr/bin/env bash",
		"set -u",
		"echo \"Starting panel setup.\" >> %s" % _shell_quote(log_path),
	]
	for command in commands:
		var label := str(command.get("label", "Running command"))
		var args: PackedStringArray = command.get("args", PackedStringArray())
		lines.append("echo %s >> %s" % [_shell_quote(label + "..."), _shell_quote(log_path)])
		lines.append("if ! %s %s >> %s 2>&1; then" % [_shell_quote(binary_path), _join_shell_args(args), _shell_quote(log_path)])
		lines.append("  echo %s > %s" % [_shell_quote("success=0"), _shell_quote(result_path)])
		lines.append("  echo %s >> %s" % [_shell_quote("status=Panel setup failed."), _shell_quote(result_path)])
		lines.append("  echo %s >> %s" % [_shell_quote("error=%s failed." % label), _shell_quote(result_path)])
		lines.append("  exit 0")
		lines.append("fi")
		lines.append("echo %s >> %s" % [_shell_quote(label + " complete."), _shell_quote(log_path)])
	lines.append("echo %s > %s" % [_shell_quote("success=1"), _shell_quote(result_path)])
	lines.append("echo %s >> %s" % [_shell_quote("status=%s" % success_status), _shell_quote(result_path)])
	lines.append("echo %s >> %s" % [_shell_quote("error="), _shell_quote(result_path)])
	_write_text_file(script_path, "\n".join(lines) + "\n")
	OS.execute("/bin/chmod", PackedStringArray(["+x", script_path]), [], true)

func _write_windows_setup_script(script_path: String, binary_path: String, commands: Array, success_status: String, log_path: String, result_path: String) -> void:
	var lines: PackedStringArray = [
		"@echo off",
		"echo Starting panel setup.>> %s" % _windows_quote(log_path),
	]
	for command in commands:
		var label := str(command.get("label", "Running command"))
		var args: PackedStringArray = command.get("args", PackedStringArray())
		lines.append("echo %s>> %s" % [label + "...", _windows_quote(log_path)])
		lines.append("%s %s >> %s 2>&1" % [_windows_quote(binary_path), _join_windows_args(args), _windows_quote(log_path)])
		lines.append("if errorlevel 1 (")
		lines.append("  > %s echo success=0" % _windows_quote(result_path))
		lines.append("  >> %s echo status=Panel setup failed." % _windows_quote(result_path))
		lines.append("  >> %s echo error=%s failed." % [_windows_quote(result_path), label])
		lines.append("  exit /b 0")
		lines.append(")")
		lines.append("echo %s>> %s" % [label + " complete.", _windows_quote(log_path)])
	lines.append("> %s echo success=1" % _windows_quote(result_path))
	lines.append(">> %s echo status=%s" % [_windows_quote(result_path), success_status])
	lines.append(">> %s echo error=" % _windows_quote(result_path))
	_write_text_file(script_path, "\r\n".join(lines) + "\r\n")

func _write_text_file(path: String, text: String) -> void:
	var file := FileAccess.open(path, FileAccess.WRITE)
	if file == null:
		return
	file.store_string(text)
	file.close()

func _read_setup_result(path: String) -> Dictionary:
	var out := {"success": false, "status": "Panel setup failed.", "error": "Panel setup failed."}
	var file := FileAccess.open(path, FileAccess.READ)
	if file == null:
		return out
	while not file.eof_reached():
		var line := file.get_line()
		if not line.contains("="):
			continue
		var parts := line.split("=", false, 1)
		if parts.size() != 2:
			continue
		match parts[0]:
			"success":
				out["success"] = parts[1] == "1"
			"status":
				out["status"] = parts[1]
			"error":
				out["error"] = parts[1]
	file.close()
	return out

func _refresh_setup_log_from_disk() -> void:
	if _setup_log_path == "":
		return
	if not FileAccess.file_exists(_setup_log_path):
		return
	var file := FileAccess.open(_setup_log_path, FileAccess.READ)
	if file == null:
		return
	var text := file.get_as_text()
	file.close()
	if text == _setup_last_log_text:
		return
	_setup_last_log_text = text
	setup_log_text.text = text
	setup_log_text.scroll_vertical = setup_log_text.get_line_count()

func _refresh_setup_state() -> void:
	var panels_dir := panels_dir_edit.text.strip_edges()
	var species := species_edit.text.strip_edges()
	var manifest_exists := FileAccess.file_exists(panels_dir.path_join("manifest.json"))
	var species_installed := _species_installed_marker_exists(panels_dir, species)

	setup_panel.visible = (not manifest_exists) or (species != "" and not species_installed)
	if not manifest_exists:
		if _setup_task_running:
			_set_setup_status("Initial panel download is running in the background.")
		else:
			_set_setup_status("Panel metadata is missing. Initial setup will download all species into the shared panels directory.")
	elif species != "" and not species_installed:
		_set_setup_status("Species '%s' is not installed in the shared panels directory." % species)
	else:
		_set_setup_status("Shared panels directory is ready.")
	install_species_button.disabled = (species == "")

func _set_setup_busy(busy: bool) -> void:
	update_metadata_button.disabled = busy
	install_species_button.disabled = busy or species_edit.text.strip_edges() == ""
	refresh_setup_button.disabled = busy
	run_button.disabled = busy

func _maybe_start_initial_panels_bootstrap() -> void:
	if _setup_task_running:
		return
	if DisplayServer.get_name() == "headless":
		return
	var panels_dir := panels_dir_edit.text.strip_edges()
	if panels_dir == "":
		return
	if FileAccess.file_exists(panels_dir.path_join("manifest.json")):
		return
	_start_panels_task([
		{
			"label": "Updating panel metadata",
			"args": PackedStringArray([
				"panels",
				"update_metadata",
				"--panels_dir", panels_dir,
			]),
		},
		{
			"label": "Installing panels for all species",
			"args": PackedStringArray([
				"panels",
				"update_species",
				"--panels_dir", panels_dir,
				"all",
			]),
		},
	], "Downloading panel metadata and all species in the background...", "All species panels are ready.")

func _species_installed_marker_exists(panels_dir: String, species: String) -> bool:
	if species == "":
		return false
	return FileAccess.file_exists(panels_dir.path_join(species).path_join("manifest.json"))

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

func _on_files_dropped(files: PackedStringArray) -> void:
	if files.is_empty():
		return
	var path := files[0]
	if path.to_lower().ends_with(".json"):
		_load_json_result(path, sample_edit.text.strip_edges())
		_set_status("Loaded result JSON from %s." % path)
		return
	reads_edit.text = path
	if sample_edit.text.strip_edges() == "" or sample_edit.text == "sample":
		sample_edit.text = path.get_file().get_basename()
	if panels_dir_edit.text.strip_edges() != "" and species_edit.text.strip_edges() != "":
		_on_run_button_pressed()
	else:
		_set_status("Loaded reads file from drag and drop. Fill remaining fields and run.")

func _join_shell_args(args: PackedStringArray) -> String:
	var parts: PackedStringArray = []
	for arg in args:
		parts.append(_shell_quote(arg))
	return " ".join(parts)

func _shell_quote(value: String) -> String:
	return "'" + value.replace("'", "'\"'\"'") + "'"

func _join_windows_args(args: PackedStringArray) -> String:
	var parts: PackedStringArray = []
	for arg in args:
		parts.append(_windows_quote(arg))
	return " ".join(parts)

func _windows_quote(value: String) -> String:
	return "\"" + value.replace("\"", "\"\"") + "\""
