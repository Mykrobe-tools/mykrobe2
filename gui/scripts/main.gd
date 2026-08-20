extends Control

const LocalMykrobe2ManagerScript = preload("res://scripts/local_mykrobe2_manager.gd")
const GUIHelpersScript = preload("res://scripts/gui_helpers.gd")
const PanelsSetupManagerScript = preload("res://scripts/panels_setup_manager.gd")
const PredictRunManagerScript = preload("res://scripts/predict_run_manager.gd")
const ThemesLibScript = preload("res://scripts/themes.gd")
const LOGO_ICON_PATH = "res://assets/mykrobe-predictor-tb-icon.png"
const SETTINGS_PATH := "user://settings.cfg"
const APPEARANCE_SYSTEM := "System"
const APPEARANCE_LIGHT := "Light"
const APPEARANCE_DARK := "Dark"
const UI_SCALE_MIN := 0.75
const UI_SCALE_MAX := 2.5
const UI_SCALE_DEFAULT := 1.0

const DEFAULT_SAMPLE_NAME := "sample"

@onready var background_rect: ColorRect = $Background
@onready var animated_background: Control = $AnimatedBackground
@onready var landing_view: LandingView = $LandingView
@onready var bootstrap_view: BootstrapView = $BootstrapView
@onready var results_view: ResultsView = $ResultsView
@onready var processing_overlay: ProcessingOverlay = $ProcessingOverlay
@onready var status_label: Label = $StatusLabel
@onready var settings_toggle_button: Button = $SettingsToggleButton
@onready var settings_drawer: SettingsDrawer = $SettingsDrawer
@onready var panels_info_dialog: PanelsInfoDialog = $PanelsInfoDialog
@onready var choose_panel_dialog: ChoosePanelDialog = $ChoosePanelDialog
@onready var output_dialog: FileDialog = $OutputDialog

var _local_mykrobe2_manager: RefCounted
var _helpers: RefCounted
var _panels_setup: RefCounted
var _predict_run: RefCounted
var _themes_lib: RefCounted
var _theme_name := "Light"
var _appearance_mode := APPEARANCE_SYSTEM
var _ui_scale := UI_SCALE_DEFAULT
var _palette: Dictionary = {}
var _species_entries: Array = []
var _selected_species_name := ""
var _selected_panel_name := ""
var _current_result_text := ""
var _current_result_sample := ""
var _current_result_path := ""
var _sample_name := DEFAULT_SAMPLE_NAME
var _panels_dir := ""
var _output_dialog_mode := ""
var _pending_result_path := ""
var _pending_result_attempts := 0

func _ready() -> void:
	_helpers = GUIHelpersScript.new()
	_panels_setup = PanelsSetupManagerScript.new()
	_predict_run = PredictRunManagerScript.new()
	_themes_lib = ThemesLibScript.new()
	_local_mykrobe2_manager = LocalMykrobe2ManagerScript.new()
	_local_mykrobe2_manager.configure("bin")
	_load_settings()
	_apply_ui_scale()
	settings_drawer.set_appearance(_appearance_mode)
	settings_drawer.set_ui_scale(_ui_scale)
	settings_drawer.set_app_version(str(ProjectSettings.get_setting("application/config/version", "dev")))
	_apply_effective_theme()
	if DisplayServer.is_dark_mode_supported():
		DisplayServer.set_system_theme_change_callback(_on_system_theme_changed)
	_panels_dir = _helpers.default_panels_dir()
	_set_notice("")
	_set_window_title_default()
	get_viewport().files_dropped.connect(_on_files_dropped)
	_refresh_species_options()
	_refresh_setup_state()
	_maybe_start_initial_panels_bootstrap()

func _exit_tree() -> void:
	if DisplayServer.is_dark_mode_supported():
		DisplayServer.set_system_theme_change_callback(Callable())

func _effective_theme_name() -> String:
	if _appearance_mode == APPEARANCE_DARK:
		return APPEARANCE_DARK
	if _appearance_mode == APPEARANCE_LIGHT:
		return APPEARANCE_LIGHT
	if DisplayServer.is_dark_mode_supported() and DisplayServer.is_dark_mode():
		return APPEARANCE_DARK
	return APPEARANCE_LIGHT

func _apply_effective_theme() -> void:
	_apply_theme(_effective_theme_name())

func _load_settings() -> void:
	var config := ConfigFile.new()
	if config.load(SETTINGS_PATH) != OK:
		return
	var configured := str(config.get_value("ui", "appearance", APPEARANCE_SYSTEM))
	if configured in [APPEARANCE_SYSTEM, APPEARANCE_LIGHT, APPEARANCE_DARK]:
		_appearance_mode = configured
	_ui_scale = clampf(float(config.get_value("ui", "scale", UI_SCALE_DEFAULT)), UI_SCALE_MIN, UI_SCALE_MAX)

func _save_settings() -> void:
	var config := ConfigFile.new()
	config.load(SETTINGS_PATH)
	config.set_value("ui", "appearance", _appearance_mode)
	config.set_value("ui", "scale", _ui_scale)
	config.save(SETTINGS_PATH)

func _apply_ui_scale() -> void:
	get_window().content_scale_factor = _ui_scale

func _apply_theme(theme_name: String) -> void:
	if _themes_lib == null or not _themes_lib.has_theme(theme_name):
		return
	_theme_name = theme_name
	_palette = _themes_lib.palette(theme_name)
	self.theme = _themes_lib.make_theme(theme_name, 16)
	background_rect.color = _palette.get("bg", Color("f8f5ee"))
	var icon_texture: Texture2D = _helpers.load_texture(LOGO_ICON_PATH)
	landing_view.set_logo_texture(icon_texture)
	bootstrap_view.set_logo_texture(icon_texture)
	results_view.set_logo_texture(icon_texture)
	modulate = Color(1, 1, 1, 1)
	landing_view.apply_palette(_palette)
	bootstrap_view.apply_palette(_palette)
	processing_overlay.apply_palette(_palette)
	choose_panel_dialog.apply_palette(_palette)
	results_view.apply_palette(_palette)
	settings_drawer.apply_palette(_palette)
	panels_info_dialog.apply_palette(_palette)
	_apply_palette_overrides()

func _apply_palette_overrides() -> void:
	var text: Color = _palette.get("text", Color("6d6a65"))
	status_label.add_theme_color_override("font_color", text)
	_apply_settings_button_style()

func _apply_settings_button_style() -> void:
	var normal := StyleBoxFlat.new()
	normal.bg_color = Color(0, 0, 0, 0)
	normal.border_color = Color(0, 0, 0, 0)
	normal.set_border_width_all(1)
	normal.set_corner_radius_all(22)
	normal.set_content_margin_all(8)
	var hover := normal.duplicate()
	hover.bg_color = _palette.get("button_hover", Color("f4fbff"))
	hover.border_color = _palette.get("button_border", Color("b9d6ea"))
	var pressed := hover.duplicate()
	pressed.bg_color = _palette.get("selection_bg", Color("e9f3f8"))
	for button in [settings_toggle_button, results_view.settings_button]:
		button.add_theme_stylebox_override("normal", normal)
		button.add_theme_stylebox_override("hover", hover)
		button.add_theme_stylebox_override("focus", hover)
		button.add_theme_stylebox_override("pressed", pressed)

func _process(_delta: float) -> void:
	_poll_panels_setup()
	_poll_predict_run()

func _poll_panels_setup() -> void:
	var result: Dictionary = _panels_setup.poll()
	if result.get("running", false):
		bootstrap_view.set_log(str(result.get("log", "")))
		return
	if not result.get("finished", false):
		return
	bootstrap_view.set_log(str(result.get("log", "")))
	if result.get("success", false):
		_refresh_species_options()
		_refresh_setup_state()
		_set_notice("")
		return
	_refresh_setup_state()
	_set_notice(str(result.get("error", "Panel setup failed.")))

func _poll_predict_run() -> void:
	if _pending_result_path != "":
		if _load_json_result(_pending_result_path, _current_result_sample, true):
			_pending_result_path = ""
			_pending_result_attempts = 0
			processing_overlay.stop()
			_set_notice("")
			return
		_pending_result_attempts += 1
		processing_overlay.show_loading_results()
		if _pending_result_attempts >= 40:
			var failed_path := _pending_result_path
			_pending_result_path = ""
			_pending_result_attempts = 0
			processing_overlay.stop()
			_show_landing_view()
			_set_notice("Analysis finished but the result JSON could not be loaded from %s." % failed_path)
			_set_window_title_default()
		return
	var result: Dictionary = _predict_run.poll()
	if result.get("running", false):
		var progress_variant: Variant = result.get("progress", {})
		if typeof(progress_variant) == TYPE_DICTIONARY:
			processing_overlay.set_progress(Dictionary(progress_variant))
		return
	if not result.get("finished", false):
		return
	processing_overlay.stop()
	if result.get("success", false):
		_pending_result_path = str(result.get("output_path", _current_result_path))
		_pending_result_attempts = 0
		processing_overlay.show_loading_results()
		return
	_show_landing_view()
	_set_notice("%s\n%s" % [str(result.get("error", "Analysis failed.")), str(result.get("log", ""))])
	_set_window_title_default()

func _on_analyse_requested(paths: PackedStringArray) -> void:
	if bootstrap_view.visible:
		return
	_sample_name = _sample_name_from_reads(paths)
	_start_predict(paths)

func _on_change_requested() -> void:
	_show_options_dialog()

func _on_settings_toggle_pressed() -> void:
	settings_drawer.toggle_drawer()

func _on_appearance_changed(mode: String) -> void:
	if mode not in [APPEARANCE_SYSTEM, APPEARANCE_LIGHT, APPEARANCE_DARK]:
		return
	_appearance_mode = mode
	_save_settings()
	_apply_effective_theme()

func _on_ui_scale_changed(value: float) -> void:
	_ui_scale = clampf(value, UI_SCALE_MIN, UI_SCALE_MAX)
	_apply_ui_scale()
	_save_settings()

func _on_system_theme_changed() -> void:
	if _appearance_mode == APPEARANCE_SYSTEM:
		_apply_effective_theme()

func _on_panel_information_requested() -> void:
	settings_drawer.close_drawer()
	panels_info_dialog.open_dialog(_species_entries, _panels_dir, _selected_species_name)

func _on_panel_selected(species: String, panel: String) -> void:
	_selected_species_name = species
	_selected_panel_name = panel
	_update_landing_selection()

func _on_output_dialog_file_selected(path: String) -> void:
	if _output_dialog_mode != "save_result":
		return
	_output_dialog_mode = ""
	if _current_result_text == "":
		_set_notice("No result is loaded.")
		return
	var file := FileAccess.open(path, FileAccess.WRITE)
	if file == null:
		_set_notice("Could not write %s." % path)
		return
	file.store_string(_current_result_text)
	file.close()
	_set_notice("Saved %s." % path)

func _on_save_requested() -> void:
	if _current_result_text == "":
		_set_notice("No result is loaded.")
		return
	_output_dialog_mode = "save_result"
	output_dialog.popup_centered_ratio(0.7)

func _on_new_requested() -> void:
	_current_result_text = ""
	_current_result_sample = ""
	_current_result_path = ""
	_sample_name = DEFAULT_SAMPLE_NAME
	landing_view.clear_files()
	results_view.clear()
	_show_landing_view()
	_set_notice("")
	_set_window_title_default()

func _on_results_tab_changed(tab_name: String) -> void:
	if _current_result_sample != "":
		_set_window_title_results(_current_result_sample, tab_name)

func _on_cancel_requested() -> void:
	if not _predict_run.is_running():
		return
	processing_overlay.set_cancel_enabled(false)
	_predict_run.cancel()
	processing_overlay.stop()
	_show_landing_view()
	_set_notice("Analysis cancelled.")
	_set_window_title_default()

func _start_predict(reads_paths: PackedStringArray) -> void:
	settings_drawer.close_drawer(false)
	var sample := _sample_name.strip_edges()
	var panels_dir := _panels_dir.strip_edges()
	var species := _selected_species()
	var panel_name := _selected_panel()
	if sample == "":
		_set_notice("Sample name is required.")
		return
	if reads_paths.is_empty():
		_set_notice("At least one read file is required.")
		return
	for path in reads_paths:
		if path.strip_edges() == "" or not FileAccess.file_exists(path):
			_set_notice("Read file was not found: %s." % path)
			return
	if panels_dir == "":
		_set_notice("Panels directory is required.")
		return
	if species == "":
		_set_notice("Species is required.")
		_show_options_dialog()
		return
	var binary_path := _resolve_binary_path()
	if binary_path == "":
		_set_notice("Could not find mykrobe2 binary.")
		return

	var output_path: String = _helpers.temporary_output_path(sample)
	var args := PackedStringArray([
		"predict",
		"--sample", sample,
		"--species", species,
		"--panels-dir", panels_dir,
		"--output", output_path,
		"--format", "json",
	])
	for path in reads_paths:
		args.append_array(["--seq", path])
	if panel_name != "":
		args.append_array(["--panel", panel_name])
	args.append("--guess-sequence-method")

	var start_result: Dictionary = _predict_run.start(binary_path, args, output_path)
	if not start_result.get("started", false):
		_set_notice(str(start_result.get("error", "Could not start analysis.")))
		return

	_current_result_sample = sample
	_current_result_path = output_path
	settings_toggle_button.visible = false
	processing_overlay.start()
	_set_notice("")
	_set_window_title_processing(sample)

func _load_json_result(path: String, preferred_sample: String = "sample", quiet: bool = false) -> bool:
	if not FileAccess.file_exists(path):
		if not quiet:
			_set_notice("Result JSON was not found at %s." % path)
		return false
	var file := FileAccess.open(path, FileAccess.READ)
	if file == null:
		if not quiet:
			_set_notice("Could not open JSON file: %s." % path)
		return false
	var text := file.get_as_text()
	file.close()
	var parsed = JSON.parse_string(text)
	if parsed == null:
		if not quiet:
			_set_notice("JSON parsing failed for %s." % path)
		return false
	_current_result_text = text
	_current_result_path = path
	var result_sample := _resolve_result_sample(preferred_sample, parsed)
	_current_result_sample = result_sample
	_display_results(result_sample, parsed)
	return true

func _resolve_result_sample(preferred_sample: String, parsed: Variant) -> String:
	if typeof(parsed) != TYPE_DICTIONARY:
		return preferred_sample
	var root: Dictionary = parsed
	if root.has(preferred_sample):
		return preferred_sample
	if root.keys().is_empty():
		return preferred_sample
	return str(root.keys()[0])

func _display_results(sample: String, parsed: Variant) -> void:
	results_view.display(sample, parsed)
	_show_results_view()

func _show_landing_view() -> void:
	landing_view.visible = true
	bootstrap_view.visible = false
	results_view.visible = false
	animated_background.visible = true
	settings_toggle_button.visible = true

func _show_bootstrap_view() -> void:
	landing_view.visible = false
	bootstrap_view.visible = true
	results_view.visible = false
	animated_background.visible = true
	settings_toggle_button.visible = false
	settings_drawer.close_drawer(false)

func _show_results_view() -> void:
	landing_view.visible = false
	bootstrap_view.visible = false
	results_view.visible = true
	animated_background.visible = false
	settings_toggle_button.visible = false

func _show_options_dialog() -> void:
	settings_drawer.close_drawer(false)
	choose_panel_dialog.open_dialog(_species_entries, _selected_species_name, _selected_panel_name)

func _refresh_setup_state() -> void:
	var panels_dir := _panels_dir.strip_edges()
	var manifest_exists := FileAccess.file_exists(panels_dir.path_join("manifest.json"))
	if _panels_setup.is_running() or not manifest_exists:
		bootstrap_view.set_status("Panel data missing. Downloading and processing data. This may take a few minutes")
		_show_bootstrap_view()
		return
	if _current_result_text != "":
		_show_results_view()
	else:
		_show_landing_view()

func _maybe_start_initial_panels_bootstrap() -> void:
	if _panels_setup.is_running():
		return
	if DisplayServer.get_name() == "headless":
		return
	var panels_dir := _panels_dir.strip_edges()
	if panels_dir == "":
		return
	if FileAccess.file_exists(panels_dir.path_join("manifest.json")):
		return
	_show_bootstrap_view()
	var binary_path := _resolve_binary_path()
	if binary_path == "":
		_set_notice("Could not find mykrobe2 binary for panel setup.")
		return
	var start_result: Dictionary = _panels_setup.start(binary_path, [
		{
			"label": "Updating panel metadata",
			"args": PackedStringArray([
				"panels",
				"update-metadata",
				"--panels-dir", panels_dir,
			]),
		},
		{
			"label": "Installing panels for all species",
			"args": PackedStringArray([
				"panels",
				"update-species",
				"--panels-dir", panels_dir,
				"all",
			]),
		},
	], "All species panels are ready.")
	if not start_result.get("started", false):
		_set_notice(str(start_result.get("error", "Could not start panel setup.")))

func _selected_species() -> String:
	return _selected_species_name

func _selected_panel() -> String:
	return _selected_panel_name

func _refresh_species_options() -> void:
	_species_entries = _helpers.load_species_entries(_resolve_binary_path(), _panels_dir.strip_edges())
	settings_drawer.set_panel_entries(_species_entries)
	if _species_entries.is_empty():
		_selected_species_name = ""
		_selected_panel_name = ""
		landing_view.set_analysis_enabled(false)
		_update_landing_selection()
		return
	var preferred_panel := _selected_panel_name
	var species_entry: Dictionary = _find_species_entry(_selected_species_name)
	if species_entry.is_empty():
		preferred_panel = ""
		species_entry = _find_species_entry("tb")
	if species_entry.is_empty():
		species_entry = Dictionary(_species_entries[0])
	_selected_species_name = str(species_entry.get("species", "")).strip_edges()
	_selected_panel_name = _resolve_panel_name(species_entry, preferred_panel)
	landing_view.set_analysis_enabled(true)
	_update_landing_selection()

func _find_species_entry(species: String) -> Dictionary:
	for entry_variant in _species_entries:
		if typeof(entry_variant) != TYPE_DICTIONARY:
			continue
		var entry: Dictionary = entry_variant
		if str(entry.get("species", "")).strip_edges() == species:
			return entry
	return {}

func _resolve_panel_name(species_entry: Dictionary, preferred_panel: String) -> String:
	var panels_variant: Variant = species_entry.get("panels", [])
	if typeof(panels_variant) != TYPE_ARRAY:
		return ""
	var first_panel := ""
	for panel_variant in panels_variant:
		if typeof(panel_variant) != TYPE_DICTIONARY:
			continue
		var panel_name := str(Dictionary(panel_variant).get("name", "")).strip_edges()
		if panel_name == "":
			continue
		if first_panel == "":
			first_panel = panel_name
		if panel_name == preferred_panel:
			return panel_name
	var default_panel := str(species_entry.get("default_panel", "")).strip_edges()
	for panel_variant in panels_variant:
		if typeof(panel_variant) == TYPE_DICTIONARY and str(Dictionary(panel_variant).get("name", "")).strip_edges() == default_panel:
			return default_panel
	return first_panel

func _update_landing_selection() -> void:
	landing_view.set_panel(_selected_species(), _selected_panel())

func _resolve_binary_path() -> String:
	var from_env := OS.get_environment("MYKROBE2_BINARY").strip_edges()
	if from_env != "" and FileAccess.file_exists(from_env):
		return from_env
	if _local_mykrobe2_manager != null and _local_mykrobe2_manager.ensure_local_binary_installed():
		return _local_mykrobe2_manager.installed_binary_path()
	return ""

func _set_notice(message: String) -> void:
	status_label.visible = message.strip_edges() != ""
	status_label.text = message

func _set_window_title_default() -> void:
	get_window().title = "Mykrobe"

func _set_window_title_processing(sample: String) -> void:
	get_window().title = "%s - Analysing - Mykrobe" % sample

func _set_window_title_results(sample: String, tab_name: String) -> void:
	get_window().title = "%s - Resistance - %s - Mykrobe" % [sample, tab_name]

func _on_files_dropped(files: PackedStringArray) -> void:
	if files.is_empty():
		return
	if files.size() == 1 and files[0].to_lower().ends_with(".json"):
		if _load_json_result(files[0], _sample_name):
			_set_notice("Loaded result JSON from %s." % files[0])
		return
	for path in files:
		if path.to_lower().ends_with(".json"):
			_set_notice("Drop a result JSON by itself, or add only read files.")
			return
	landing_view.add_files(files)
	_show_landing_view()
	_set_notice("")

func _sample_name_from_reads(paths: PackedStringArray) -> String:
	if paths.is_empty():
		return DEFAULT_SAMPLE_NAME
	var sample := paths[0].get_file().get_basename().strip_edges()
	return DEFAULT_SAMPLE_NAME if sample == "" else sample
