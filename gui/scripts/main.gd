extends Control

const LocalMykrobe2ManagerScript = preload("res://scripts/local_mykrobe2_manager.gd")
const ResultFormatterScript = preload("res://scripts/result_formatter.gd")
const GUIHelpersScript = preload("res://scripts/gui_helpers.gd")
const PanelsSetupManagerScript = preload("res://scripts/panels_setup_manager.gd")
const BACKGROUND_IMAGE_PATH = "res://assets/background.png"
const LOGO_IMAGE_PATH = "res://assets/mykrobe-predictor-tb-icon.png"

@onready var background_texture: TextureRect = $Background
@onready var bootstrap_panel: PanelContainer = $RootMargin/RootVBox/BootstrapPanel
@onready var bootstrap_status_label: Label = $RootMargin/RootVBox/BootstrapPanel/BootstrapMargin/BootstrapVBox/BootstrapStatus
@onready var bootstrap_log_text: RichTextLabel = $RootMargin/RootVBox/BootstrapPanel/BootstrapMargin/BootstrapVBox/BootstrapLog
@onready var body_split: HSplitContainer = $RootMargin/RootVBox/BodySplit
@onready var sample_edit: LineEdit = $RootMargin/RootVBox/BodySplit/FormPanel/FormMargin/FormVBox/SampleRow/SampleEdit
@onready var reads_edit: LineEdit = $RootMargin/RootVBox/BodySplit/FormPanel/FormMargin/FormVBox/ReadsRow/ReadsPicker/ReadsEdit
@onready var panels_dir_edit: LineEdit = $RootMargin/RootVBox/BodySplit/FormPanel/FormMargin/FormVBox/PanelsRow/PanelsPicker/PanelsDirEdit
@onready var species_option: OptionButton = $RootMargin/RootVBox/BodySplit/FormPanel/FormMargin/FormVBox/SpeciesRow/SpeciesOption
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
@onready var setup_log_text: RichTextLabel = $RootMargin/RootVBox/BodySplit/FormPanel/FormMargin/FormVBox/SetupPanel/SetupMargin/SetupVBox/SetupLog
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
var _formatter: RefCounted
var _helpers: RefCounted
var _panels_setup: RefCounted
var _species_entries: Array = []

func _ready() -> void:
	_formatter = ResultFormatterScript.new()
	_helpers = GUIHelpersScript.new()
	_panels_setup = PanelsSetupManagerScript.new()
	_local_mykrobe2_manager = LocalMykrobe2ManagerScript.new()
	_local_mykrobe2_manager.configure("bin")
	_apply_branding_assets()
	panels_dir_edit.text = _helpers.default_panels_dir()
	status_label.text = "Ready."
	_clear_results()
	get_viewport().files_dropped.connect(_on_files_dropped)
	_refresh_species_options()
	_refresh_setup_state()
	_maybe_start_initial_panels_bootstrap()

func _process(_delta: float) -> void:
	var result: Dictionary = _panels_setup.poll()
	if result.get("running", false):
		setup_log_text.text = str(result.get("log", ""))
		bootstrap_log_text.text = str(result.get("log", ""))
		return
	if not result.get("finished", false):
		return
	_set_setup_busy(false)
	setup_log_text.text = str(result.get("log", ""))
	bootstrap_log_text.text = str(result.get("log", ""))
	if result.get("success", false):
		_set_status(str(result.get("status", "Panel setup complete.")))
		_refresh_species_options()
		_refresh_setup_state()
		return
	_set_status(str(result.get("error", "Panel setup failed.")))
	_set_setup_status("Panel setup failed.")

func _apply_branding_assets() -> void:
	background_texture.texture = _helpers.load_png_texture(BACKGROUND_IMAGE_PATH)
	var logo_texture_rect: TextureRect = $RootMargin/RootVBox/Header/HeaderMargin/HeaderVBox/HeaderTop/Logo
	logo_texture_rect.texture = _helpers.load_png_texture(LOGO_IMAGE_PATH)

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
	_refresh_species_options()
	_refresh_setup_state()
	_maybe_start_initial_panels_bootstrap()

func _on_output_dialog_file_selected(path: String) -> void:
	output_edit.text = path

func _on_species_option_item_selected(index: int) -> void:
	if index < 0 or index >= _species_entries.size():
		return
	var entry: Dictionary = _species_entries[index]
	var default_panel := str(entry.get("default_panel", ""))
	if default_panel != "":
		panel_edit.text = default_panel

func _on_clear_button_pressed() -> void:
	status_label.text = "Ready."
	_clear_results()

func _on_run_button_pressed() -> void:
	var sample := sample_edit.text.strip_edges()
	var reads_path := reads_edit.text.strip_edges()
	var panels_dir := panels_dir_edit.text.strip_edges()
	var species := _selected_species()

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
	if not _helpers.species_installed_marker_exists(panels_dir, species):
		_set_status("Species panels are not installed yet. Use Panels Setup first.")
		_refresh_setup_state()
		return

	var binary_path := _resolve_binary_path()
	if binary_path == "":
		_set_status("Could not find mykrobe2 binary. Set MYKROBE2_BINARY or provide a bundled binary.")
		return

	var output_path := output_edit.text.strip_edges()
	if output_path == "":
		output_path = _helpers.temporary_output_path(sample)

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
		_set_status("mykrobe2 failed with exit code %d.\n%s" % [exit_code, "\n".join(output_lines)])
		return

	_load_json_result(output_path, sample)
	if raw_json_text.text != "":
		_set_status("Completed successfully using %s." % binary_path)
	_refresh_setup_state()
	_refresh_species_options()

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
	overview_text.text = _formatter.format_overview(sample, parsed)
	drugs_text.text = _formatter.format_drugs(sample, parsed)
	species_text.text = _formatter.format_species(sample, parsed)
	evidence_text.text = _formatter.format_evidence(sample, parsed)

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
	var species := _selected_species()
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
	_refresh_species_options()
	_refresh_setup_state()

func _start_panels_task(commands: Array, status_prefix: String, success_status: String) -> void:
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
	bootstrap_status_label.text = status_prefix
	_set_setup_busy(true)
	setup_log_text.text = ""
	bootstrap_log_text.text = ""
	var start_result: Dictionary = _panels_setup.start(binary_path, commands, success_status)
	if not start_result.get("started", false):
		_set_setup_busy(false)
		_set_status(str(start_result.get("error", "Could not start background panel setup.")))
		_set_setup_status("Could not start panel setup.")

func _refresh_setup_state() -> void:
	var panels_dir := panels_dir_edit.text.strip_edges()
	var species := _selected_species()
	var manifest_exists := FileAccess.file_exists(panels_dir.path_join("manifest.json"))
	var species_installed: bool = _helpers.species_installed_marker_exists(panels_dir, species)
	var bootstrap_mode := _should_show_bootstrap(manifest_exists)

	bootstrap_panel.visible = bootstrap_mode
	body_split.visible = not bootstrap_mode
	setup_panel.visible = (not manifest_exists) or (species != "" and not species_installed)
	if not manifest_exists:
		if _panels_setup.is_running():
			_set_setup_status("Initial panel download is running in the background.")
			bootstrap_status_label.text = "Downloading panel metadata and species data. This can take a little while."
		else:
			_set_setup_status("Panel metadata is missing. Initial setup will download all species into the shared panels directory.")
			bootstrap_status_label.text = "Panel metadata is missing. Initial setup will download all species into the shared panels directory."
	elif species != "" and not species_installed:
		_set_setup_status("Species '%s' is not installed in the shared panels directory." % species)
		bootstrap_status_label.text = "Species panels are still being prepared."
	else:
		_set_setup_status("Shared panels directory is ready.")
		bootstrap_status_label.text = "Shared panels directory is ready."
	install_species_button.disabled = (species == "")

func _set_setup_busy(busy: bool) -> void:
	update_metadata_button.disabled = busy
	install_species_button.disabled = busy or _selected_species() == ""
	refresh_setup_button.disabled = busy
	run_button.disabled = busy

func _maybe_start_initial_panels_bootstrap() -> void:
	if _panels_setup.is_running():
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

func _selected_species() -> String:
	if species_option.item_count == 0:
		return ""
	var idx := species_option.selected
	if idx < 0 or idx >= _species_entries.size():
		return ""
	return str(_species_entries[idx].get("species", "")).strip_edges()

func _refresh_species_options() -> void:
	_species_entries = _helpers.load_species_entries(_resolve_binary_path(), panels_dir_edit.text.strip_edges())
	species_option.clear()
	species_option.disabled = true
	species_option.text = "Loading species..."
	if _species_entries.is_empty():
		species_option.text = "No species available"
		return
	for entry in _species_entries:
		species_option.add_item(str(entry.get("species", "")))
	species_option.disabled = false
	var preferred_index := 0
	for i in range(_species_entries.size()):
		if str(_species_entries[i].get("species", "")) == "tb":
			preferred_index = i
			break
	species_option.select(preferred_index)
	_on_species_option_item_selected(preferred_index)

func _should_show_bootstrap(manifest_exists: bool) -> bool:
	return (not manifest_exists) or _panels_setup.is_running()

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
	if panels_dir_edit.text.strip_edges() != "" and _selected_species() != "":
		_on_run_button_pressed()
	else:
		_set_status("Loaded reads file from drag and drop. Fill remaining fields and run.")
