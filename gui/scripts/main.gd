extends Control

const LocalMykrobe2ManagerScript = preload("res://scripts/local_mykrobe2_manager.gd")
const GUIHelpersScript = preload("res://scripts/gui_helpers.gd")
const PanelsSetupManagerScript = preload("res://scripts/panels_setup_manager.gd")
const PredictRunManagerScript = preload("res://scripts/predict_run_manager.gd")
const ThemesLibScript = preload("res://scripts/themes.gd")
const LOGO_ICON_PATH = "res://assets/mykrobe-predictor-tb-icon.png"

const DEFAULT_SAMPLE_NAME := "sample"

@onready var background_rect: ColorRect = $Background
@onready var animated_background: Control = $AnimatedBackground
@onready var bootstrap_circle: PanelContainer = $BootstrapView/BootstrapCenter/BootstrapCard/BootstrapCircle
@onready var processing_circle: PanelContainer = $ProcessingOverlay/ProcessingCenter/ProcessingCard/ProcessingCircle
@onready var bootstrap_logo_icon: TextureRect = $BootstrapView/BootstrapCenter/BootstrapCard/BootstrapMargin/BootstrapVBox/BootstrapLogo/BootstrapLogoIcon
@onready var landing_view: LandingView = $LandingView
@onready var bootstrap_view: Control = $BootstrapView
@onready var bootstrap_status_label: Label = $BootstrapView/BootstrapCenter/BootstrapCard/BootstrapMargin/BootstrapVBox/BootstrapStatus
@onready var bootstrap_log_text: RichTextLabel = $BootstrapView/BootstrapCenter/BootstrapCard/BootstrapMargin/BootstrapVBox/BootstrapLog
@onready var results_view: ResultsView = $ResultsView
@onready var processing_overlay: Control = $ProcessingOverlay
@onready var processing_label: Label = $ProcessingOverlay/ProcessingCenter/ProcessingCard/ProcessingMargin/ProcessingVBox/ProcessingLabel
@onready var dot_1: Label = $ProcessingOverlay/ProcessingCenter/ProcessingCard/ProcessingMargin/ProcessingVBox/ProcessingDots/Dot1
@onready var dot_2: Label = $ProcessingOverlay/ProcessingCenter/ProcessingCard/ProcessingMargin/ProcessingVBox/ProcessingDots/Dot2
@onready var dot_3: Label = $ProcessingOverlay/ProcessingCenter/ProcessingCard/ProcessingMargin/ProcessingVBox/ProcessingDots/Dot3
@onready var cancel_button: Button = $ProcessingOverlay/ProcessingCenter/ProcessingCard/ProcessingMargin/ProcessingVBox/CancelButton
@onready var status_label: Label = $StatusLabel
@onready var choose_panel_dialog: ChoosePanelDialog = $ChoosePanelDialog
@onready var reads_dialog: FileDialog = $ReadsDialog
@onready var output_dialog: FileDialog = $OutputDialog

var _local_mykrobe2_manager: RefCounted
var _helpers: RefCounted
var _panels_setup: RefCounted
var _predict_run: RefCounted
var _themes_lib: RefCounted
var _theme_name := "Light"
var _palette: Dictionary = {}
var _species_entries: Array = []
var _selected_species_name := ""
var _selected_panel_name := ""
var _current_result_text := ""
var _current_result_sample := ""
var _current_result_path := ""
var _pending_run_after_reads_selection := false
var _sample_name := DEFAULT_SAMPLE_NAME
var _panels_dir := ""
var _output_dialog_mode := ""
var _processing_elapsed := 0.0
var _pending_result_path := ""
var _pending_result_attempts := 0

func _ready() -> void:
	_helpers = GUIHelpersScript.new()
	_panels_setup = PanelsSetupManagerScript.new()
	_predict_run = PredictRunManagerScript.new()
	_themes_lib = ThemesLibScript.new()
	_local_mykrobe2_manager = LocalMykrobe2ManagerScript.new()
	_local_mykrobe2_manager.configure("bin")
	_apply_theme(_theme_name)
	_panels_dir = _helpers.default_panels_dir()
	_set_notice("")
	_set_window_title_default()
	get_viewport().files_dropped.connect(_on_files_dropped)
	_refresh_species_options()
	_refresh_setup_state()
	_maybe_start_initial_panels_bootstrap()

func _apply_theme(theme_name: String) -> void:
	if _themes_lib == null or not _themes_lib.has_theme(theme_name):
		return
	_theme_name = theme_name
	_palette = _themes_lib.palette(theme_name)
	self.theme = _themes_lib.make_theme(theme_name, 16)
	background_rect.color = _palette.get("bg", Color("f8f5ee"))
	var icon_texture: Texture2D = _helpers.load_texture(LOGO_ICON_PATH)
	landing_view.set_logo_texture(icon_texture)
	bootstrap_logo_icon.texture = icon_texture
	results_view.set_logo_texture(icon_texture)
	modulate = Color(1, 1, 1, 1)
	for panel in [bootstrap_circle, processing_circle]:
		var style := StyleBoxFlat.new()
		style.bg_color = _palette.get("circle_bg", Color(1, 1, 1, 0.92))
		style.corner_radius_top_left = 400
		style.corner_radius_top_right = 400
		style.corner_radius_bottom_left = 400
		style.corner_radius_bottom_right = 400
		panel.add_theme_stylebox_override("panel", style)
	landing_view.apply_palette(_palette)
	choose_panel_dialog.apply_palette(_palette)
	results_view.apply_palette(_palette)
	_apply_palette_overrides()

func _apply_palette_overrides() -> void:
	var accent: Color = _palette.get("accent", Color("3987b5"))
	var text: Color = _palette.get("text", Color("6d6a65"))
	var muted: Color = _palette.get("text_muted", Color("8b8478"))
	var dot: Color = _palette.get("dot", Color("c9c4bc"))
	$BootstrapView/BootstrapCenter/BootstrapCard/BootstrapMargin/BootstrapVBox/BootstrapLogo/BootstrapLogoText.add_theme_color_override("font_color", accent)
	$BootstrapView/BootstrapCenter/BootstrapCard/BootstrapMargin/BootstrapVBox/BootstrapTitle.add_theme_color_override("font_color", text)
	$BootstrapView/BootstrapCenter/BootstrapCard/BootstrapMargin/BootstrapVBox/BootstrapStatus.add_theme_color_override("font_color", muted)
	processing_label.add_theme_color_override("font_color", text)
	for label in [dot_1, dot_2, dot_3]:
		label.add_theme_color_override("font_color", dot)
	bootstrap_log_text.add_theme_color_override("default_color", text)
	status_label.add_theme_color_override("font_color", text)

func _process(delta: float) -> void:
	_poll_panels_setup()
	_poll_predict_run(delta)

func _poll_panels_setup() -> void:
	var result: Dictionary = _panels_setup.poll()
	if result.get("running", false):
		bootstrap_log_text.text = str(result.get("log", ""))
		return
	if not result.get("finished", false):
		return
	bootstrap_log_text.text = str(result.get("log", ""))
	if result.get("success", false):
		_refresh_species_options()
		_refresh_setup_state()
		_set_notice("")
		return
	_refresh_setup_state()
	_set_notice(str(result.get("error", "Panel setup failed.")))

func _poll_predict_run(delta: float) -> void:
	if _pending_result_path != "":
		if _load_json_result(_pending_result_path, _current_result_sample, true):
			_pending_result_path = ""
			_pending_result_attempts = 0
			processing_overlay.visible = false
			cancel_button.disabled = false
			_set_notice("")
			return
		_pending_result_attempts += 1
		processing_overlay.visible = true
		processing_label.text = "Loading results"
		if _pending_result_attempts >= 40:
			var failed_path := _pending_result_path
			_pending_result_path = ""
			_pending_result_attempts = 0
			processing_overlay.visible = false
			_show_landing_view()
			_set_notice("Analysis finished but the result JSON could not be loaded from %s." % failed_path)
			_set_window_title_default()
		return
	if _predict_run.is_running():
		_processing_elapsed += delta
		_update_processing_dots()
	var result: Dictionary = _predict_run.poll()
	if result.get("running", false):
		return
	if not result.get("finished", false):
		return
	processing_overlay.visible = false
	cancel_button.disabled = false
	if result.get("success", false):
		_pending_result_path = str(result.get("output_path", _current_result_path))
		_pending_result_attempts = 0
		processing_overlay.visible = true
		processing_label.text = "Loading results"
		return
	_show_landing_view()
	_set_notice("%s\n%s" % [str(result.get("error", "Analysis failed.")), str(result.get("log", ""))])
	_set_window_title_default()

func _on_analyse_requested() -> void:
	if bootstrap_view.visible:
		return
	_pending_run_after_reads_selection = true
	reads_dialog.popup_centered_ratio(0.7)

func _on_change_requested() -> void:
	_show_options_dialog()

func _on_panel_selected(species: String, panel: String) -> void:
	_selected_species_name = species
	_selected_panel_name = panel
	_update_landing_selection()

func _on_reads_dialog_file_selected(path: String) -> void:
	_sample_name = _sample_name_from_reads(path)
	if _pending_run_after_reads_selection:
		_pending_run_after_reads_selection = false
		_start_predict(path)

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
	results_view.clear()
	_show_landing_view()
	_set_notice("")
	_set_window_title_default()

func _on_results_tab_changed(tab_name: String) -> void:
	if _current_result_sample != "":
		_set_window_title_results(_current_result_sample, tab_name)

func _on_cancel_button_pressed() -> void:
	if not _predict_run.is_running():
		return
	cancel_button.disabled = true
	_predict_run.cancel()
	processing_overlay.visible = false
	_show_landing_view()
	_set_notice("Analysis cancelled.")
	_set_window_title_default()

func _start_predict(reads_path: String) -> void:
	var sample := _sample_name.strip_edges()
	var panels_dir := _panels_dir.strip_edges()
	var species := _selected_species()
	var panel_name := _selected_panel()
	if sample == "":
		_set_notice("Sample name is required.")
		return
	if reads_path.strip_edges() == "":
		_set_notice("Reads file is required.")
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
		"--seq", reads_path,
		"--species", species,
		"--panels-dir", panels_dir,
		"--output", output_path,
		"--format", "json",
	])
	if panel_name != "":
		args.append_array(["--panel", panel_name])
	args.append("--guess-sequence-method")

	var start_result: Dictionary = _predict_run.start(binary_path, args, output_path)
	if not start_result.get("started", false):
		_set_notice(str(start_result.get("error", "Could not start analysis.")))
		return

	_current_result_sample = sample
	_current_result_path = output_path
	_processing_elapsed = 0.0
	_update_processing_dots()
	processing_label.text = "Analysing"
	processing_overlay.visible = true
	cancel_button.disabled = false
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

func _show_bootstrap_view() -> void:
	landing_view.visible = false
	bootstrap_view.visible = true
	results_view.visible = false
	animated_background.visible = true

func _show_results_view() -> void:
	landing_view.visible = false
	bootstrap_view.visible = false
	results_view.visible = true
	animated_background.visible = false

func _show_options_dialog() -> void:
	choose_panel_dialog.open_dialog(_species_entries, _selected_species_name, _selected_panel_name)

func _refresh_setup_state() -> void:
	var panels_dir := _panels_dir.strip_edges()
	var manifest_exists := FileAccess.file_exists(panels_dir.path_join("manifest.json"))
	if _panels_setup.is_running() or not manifest_exists:
		bootstrap_status_label.text = "Panel data missing. Downloading and processing data. This may take a few minutes"
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

func _update_processing_dots() -> void:
	var active_index := int(floor(_processing_elapsed * 2.0)) % 4
	var active_count := 0 if active_index == 3 else active_index + 1
	var dots := [dot_1, dot_2, dot_3]
	for i in range(dots.size()):
		dots[i].modulate = Color(1, 1, 1, 1) if i < active_count else Color(1, 1, 1, 0.25)

func _on_files_dropped(files: PackedStringArray) -> void:
	if files.is_empty():
		return
	var path := files[0]
	if path.to_lower().ends_with(".json"):
		if _load_json_result(path, _sample_name):
			_set_notice("Loaded result JSON from %s." % path)
		return
	_sample_name = _sample_name_from_reads(path)
	_start_predict(path)

func _sample_name_from_reads(path: String) -> String:
	var sample := path.get_file().get_basename().strip_edges()
	return DEFAULT_SAMPLE_NAME if sample == "" else sample
