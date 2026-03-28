extends Control

const LocalMykrobe2ManagerScript = preload("res://scripts/local_mykrobe2_manager.gd")
const ResultFormatterScript = preload("res://scripts/result_formatter.gd")
const GUIHelpersScript = preload("res://scripts/gui_helpers.gd")
const PanelsSetupManagerScript = preload("res://scripts/panels_setup_manager.gd")
const PredictRunManagerScript = preload("res://scripts/predict_run_manager.gd")
const ThemesLibScript = preload("res://scripts/themes.gd")
const BACKGROUND_IMAGE_PATH = "res://assets/background.png"
const LOGO_ICON_PATH = "res://assets/mykrobe-predictor-tb-icon.png"

const TAB_ALL := 0
const TAB_DRUGS := 1
const TAB_EVIDENCE := 2
const TAB_SPECIES := 3

@onready var background_texture: TextureRect = $Background
@onready var landing_circle: PanelContainer = $LandingView/LandingCenter/LandingCard/LandingCircle
@onready var bootstrap_circle: PanelContainer = $BootstrapView/BootstrapCenter/BootstrapCard/BootstrapCircle
@onready var processing_circle: PanelContainer = $ProcessingOverlay/ProcessingCenter/ProcessingCard/ProcessingCircle
@onready var landing_logo_icon: TextureRect = $LandingView/LandingCenter/LandingCard/LandingMargin/LandingVBox/LandingLogo/LandingLogoIcon
@onready var bootstrap_logo_icon: TextureRect = $BootstrapView/BootstrapCenter/BootstrapCard/BootstrapMargin/BootstrapVBox/BootstrapLogo/BootstrapLogoIcon
@onready var header_logo_icon: TextureRect = $AppView/HeaderBar/HeaderMargin/HeaderHBox/HeaderLogo/HeaderLogoIcon
@onready var landing_view: Control = $LandingView
@onready var bootstrap_view: Control = $BootstrapView
@onready var bootstrap_status_label: Label = $BootstrapView/BootstrapCenter/BootstrapCard/BootstrapMargin/BootstrapVBox/BootstrapStatus
@onready var bootstrap_log_text: RichTextLabel = $BootstrapView/BootstrapCenter/BootstrapCard/BootstrapMargin/BootstrapVBox/BootstrapLog
@onready var app_view: Control = $AppView
@onready var analyse_button: Button = $LandingView/LandingCenter/LandingCard/LandingMargin/LandingVBox/LandingButtons/AnalyseButton
@onready var all_tab_button: Button = $AppView/HeaderBar/HeaderMargin/HeaderHBox/TabsRow/AllTabButton
@onready var drugs_tab_button: Button = $AppView/HeaderBar/HeaderMargin/HeaderHBox/TabsRow/DrugsTabButton
@onready var evidence_tab_button: Button = $AppView/HeaderBar/HeaderMargin/HeaderHBox/TabsRow/EvidenceTabButton
@onready var species_tab_button: Button = $AppView/HeaderBar/HeaderMargin/HeaderHBox/TabsRow/SpeciesTabButton
@onready var save_button: Button = $AppView/HeaderBar/HeaderMargin/HeaderHBox/SaveButton
@onready var new_button: Button = $AppView/HeaderBar/HeaderMargin/HeaderHBox/NewButton
@onready var all_view: Control = $AppView/ResultsMargin/ResultsStack/AllView
@onready var drugs_view: Control = $AppView/ResultsMargin/ResultsStack/DrugsView
@onready var evidence_view: Control = $AppView/ResultsMargin/ResultsStack/EvidenceView
@onready var species_view: Control = $AppView/ResultsMargin/ResultsStack/SpeciesView
@onready var all_susceptible_text: RichTextLabel = $AppView/ResultsMargin/ResultsStack/AllView/AllVBox/AllColumns/AllSusceptibleColumn/AllSusceptibleText
@onready var all_resistant_text: RichTextLabel = $AppView/ResultsMargin/ResultsStack/AllView/AllVBox/AllColumns/AllResistantColumn/AllResistantText
@onready var first_line_text: RichTextLabel = $AppView/ResultsMargin/ResultsStack/DrugsView/DrugsVBox/DrugsColumns/FirstLineColumn/FirstLineText
@onready var second_line_text: RichTextLabel = $AppView/ResultsMargin/ResultsStack/DrugsView/DrugsVBox/DrugsColumns/SecondLineColumn/SecondLineText
@onready var evidence_text: RichTextLabel = $AppView/ResultsMargin/ResultsStack/EvidenceView/EvidenceVBox/EvidenceText
@onready var species_text: RichTextLabel = $AppView/ResultsMargin/ResultsStack/SpeciesView/SpeciesVBox/SpeciesText
@onready var processing_overlay: Control = $ProcessingOverlay
@onready var processing_label: Label = $ProcessingOverlay/ProcessingCenter/ProcessingCard/ProcessingMargin/ProcessingVBox/ProcessingLabel
@onready var dot_1: Label = $ProcessingOverlay/ProcessingCenter/ProcessingCard/ProcessingMargin/ProcessingVBox/ProcessingDots/Dot1
@onready var dot_2: Label = $ProcessingOverlay/ProcessingCenter/ProcessingCard/ProcessingMargin/ProcessingVBox/ProcessingDots/Dot2
@onready var dot_3: Label = $ProcessingOverlay/ProcessingCenter/ProcessingCard/ProcessingMargin/ProcessingVBox/ProcessingDots/Dot3
@onready var cancel_button: Button = $ProcessingOverlay/ProcessingCenter/ProcessingCard/ProcessingMargin/ProcessingVBox/CancelButton
@onready var status_label: Label = $StatusLabel
@onready var options_dialog: AcceptDialog = $OptionsDialog
@onready var sample_edit: LineEdit = $OptionsDialog/OptionsMargin/OptionsVBox/SampleRow/SampleEdit
@onready var panels_dir_edit: LineEdit = $OptionsDialog/OptionsMargin/OptionsVBox/PanelsRow/PanelsPicker/PanelsDirEdit
@onready var species_option: OptionButton = $OptionsDialog/OptionsMargin/OptionsVBox/SpeciesRow/SpeciesOption
@onready var panel_option: OptionButton = $OptionsDialog/OptionsMargin/OptionsVBox/PanelRow/PanelOption
@onready var report_all_calls_check: CheckBox = $OptionsDialog/OptionsMargin/OptionsVBox/OptionsGrid/ReportAllCallsCheck
@onready var ncbi_names_check: CheckBox = $OptionsDialog/OptionsMargin/OptionsVBox/OptionsGrid/NCBINamesCheck
@onready var ont_check: CheckBox = $OptionsDialog/OptionsMargin/OptionsVBox/OptionsGrid/ONTCheck
@onready var guess_method_check: CheckBox = $OptionsDialog/OptionsMargin/OptionsVBox/OptionsGrid/GuessMethodCheck
@onready var reads_dialog: FileDialog = $ReadsDialog
@onready var panels_dir_dialog: FileDialog = $PanelsDirDialog
@onready var output_dialog: FileDialog = $OutputDialog

var _local_mykrobe2_manager: RefCounted
var _formatter: RefCounted
var _helpers: RefCounted
var _panels_setup: RefCounted
var _predict_run: RefCounted
var _themes_lib: RefCounted
var _theme_name := "Light"
var _palette: Dictionary = {}
var _species_entries: Array = []
var _panel_entries: Array = []
var _current_result_text := ""
var _current_result_sample := ""
var _current_result_path := ""
var _current_tab := TAB_ALL
var _pending_run_after_reads_selection := false
var _active_reads_path := ""
var _output_dialog_mode := ""
var _processing_elapsed := 0.0

func _ready() -> void:
	_formatter = ResultFormatterScript.new()
	_helpers = GUIHelpersScript.new()
	_panels_setup = PanelsSetupManagerScript.new()
	_predict_run = PredictRunManagerScript.new()
	_themes_lib = ThemesLibScript.new()
	_local_mykrobe2_manager = LocalMykrobe2ManagerScript.new()
	_local_mykrobe2_manager.configure("bin")
	_apply_theme(_theme_name)
	panels_dir_edit.text = _helpers.default_panels_dir()
	options_dialog.get_ok_button().text = "Close"
	_apply_tab_styles()
	_set_results_tab(TAB_ALL)
	_set_notice("")
	_set_window_title_default()
	save_button.disabled = true
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
	background_texture.texture = _helpers.load_texture(BACKGROUND_IMAGE_PATH)
	var icon_texture: Texture2D = _helpers.load_texture(LOGO_ICON_PATH)
	landing_logo_icon.texture = icon_texture
	bootstrap_logo_icon.texture = icon_texture
	header_logo_icon.texture = icon_texture
	modulate = Color(1, 1, 1, 1)
	for panel in [landing_circle, bootstrap_circle, processing_circle]:
		var style := StyleBoxFlat.new()
		style.bg_color = _palette.get("circle_bg", Color(1, 1, 1, 0.92))
		style.corner_radius_top_left = 400
		style.corner_radius_top_right = 400
		style.corner_radius_bottom_left = 400
		style.corner_radius_bottom_right = 400
		panel.add_theme_stylebox_override("panel", style)
	var header_style := StyleBoxFlat.new()
	header_style.bg_color = _palette.get("header_bg", Color(0.97, 0.96, 0.93, 0.95))
	$AppView/HeaderBar.add_theme_stylebox_override("panel", header_style)
	$AppView/HeaderBar.color = _palette.get("header_bg", Color(0.97, 0.96, 0.93, 0.95))
	_apply_palette_overrides()

func _apply_palette_overrides() -> void:
	var accent: Color = _palette.get("accent", Color("3987b5"))
	var text: Color = _palette.get("text", Color("6d6a65"))
	var muted: Color = _palette.get("text_muted", Color("8b8478"))
	var success: Color = _palette.get("success", Color("78b13f"))
	var danger: Color = _palette.get("danger", Color("f55a32"))
	var dot: Color = _palette.get("dot", Color("c9c4bc"))
	$LandingView/LandingCenter/LandingCard/LandingMargin/LandingVBox/LandingLogo/LandingLogoText.add_theme_color_override("font_color", accent)
	$BootstrapView/BootstrapCenter/BootstrapCard/BootstrapMargin/BootstrapVBox/BootstrapLogo/BootstrapLogoText.add_theme_color_override("font_color", accent)
	$AppView/HeaderBar/HeaderMargin/HeaderHBox/HeaderLogo/HeaderLogoText.add_theme_color_override("font_color", accent)
	$LandingView/LandingCenter/LandingCard/LandingMargin/LandingVBox/LandingTagline.add_theme_color_override("font_color", text)
	$LandingView/LandingCenter/LandingCard/LandingMargin/LandingVBox/LandingHint.add_theme_color_override("font_color", muted)
	$BootstrapView/BootstrapCenter/BootstrapCard/BootstrapMargin/BootstrapVBox/BootstrapTitle.add_theme_color_override("font_color", text)
	$BootstrapView/BootstrapCenter/BootstrapCard/BootstrapMargin/BootstrapVBox/BootstrapStatus.add_theme_color_override("font_color", muted)
	processing_label.add_theme_color_override("font_color", text)
	for label in [dot_1, dot_2, dot_3]:
		label.add_theme_color_override("font_color", dot)
	$AppView/ResultsMargin/ResultsStack/AllView/AllVBox/AllColumns/AllSusceptibleColumn/AllSusceptibleHeading.add_theme_color_override("font_color", success)
	$AppView/ResultsMargin/ResultsStack/AllView/AllVBox/AllColumns/AllResistantColumn/AllResistantHeading.add_theme_color_override("font_color", danger)
	for label in [
		$AppView/ResultsMargin/ResultsStack/AllView/AllVBox/AllTitle,
		$AppView/ResultsMargin/ResultsStack/DrugsView/DrugsVBox/DrugsColumns/FirstLineColumn/FirstLineTitle,
		$AppView/ResultsMargin/ResultsStack/DrugsView/DrugsVBox/DrugsColumns/SecondLineColumn/SecondLineTitle,
		$AppView/ResultsMargin/ResultsStack/EvidenceView/EvidenceVBox/EvidenceTitle,
		$AppView/ResultsMargin/ResultsStack/SpeciesView/SpeciesVBox/SpeciesTitle,
	]:
		label.add_theme_color_override("font_color", accent)
	for rich_text in [
		all_susceptible_text,
		all_resistant_text,
		first_line_text,
		second_line_text,
		evidence_text,
		species_text,
		bootstrap_log_text,
	]:
		rich_text.add_theme_color_override("default_color", text)
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
		_set_notice(str(result.get("status", "Panel setup complete.")))
		return
	_refresh_setup_state()
	_set_notice(str(result.get("error", "Panel setup failed.")))

func _poll_predict_run(delta: float) -> void:
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
		var output_path := str(result.get("output_path", _current_result_path))
		_load_json_result(output_path, _current_result_sample)
		_set_notice(str(result.get("status", "Analysis complete.")))
		return
	_show_landing_view()
	_set_notice("%s\n%s" % [str(result.get("error", "Analysis failed.")), str(result.get("log", ""))])
	_set_window_title_default()

func _on_analyse_button_pressed() -> void:
	if bootstrap_view.visible:
		return
	_pending_run_after_reads_selection = true
	reads_dialog.popup_centered_ratio(0.7)

func _on_options_button_pressed() -> void:
	options_dialog.popup_centered()

func _on_panels_browse_pressed() -> void:
	panels_dir_dialog.popup_centered_ratio(0.7)

func _on_reads_dialog_file_selected(path: String) -> void:
	_active_reads_path = path
	if sample_edit.text.strip_edges() == "" or sample_edit.text == "sample":
		sample_edit.text = path.get_file().get_basename()
	if _pending_run_after_reads_selection:
		_pending_run_after_reads_selection = false
		_start_predict(path)

func _on_panels_dir_dialog_dir_selected(path: String) -> void:
	panels_dir_edit.text = path
	_refresh_species_options()
	_refresh_setup_state()
	_maybe_start_initial_panels_bootstrap()

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

func _on_species_option_item_selected(index: int) -> void:
	if index < 0 or index >= _species_entries.size():
		return
	_refresh_panel_options(_species_entries[index])

func _on_refresh_setup_button_pressed() -> void:
	_refresh_species_options()
	_refresh_setup_state()
	_maybe_start_initial_panels_bootstrap()

func _on_use_shared_panels_button_pressed() -> void:
	panels_dir_edit.text = _helpers.default_panels_dir()
	_refresh_species_options()
	_refresh_setup_state()

func _on_all_tab_button_pressed() -> void:
	_set_results_tab(TAB_ALL)

func _on_drugs_tab_button_pressed() -> void:
	_set_results_tab(TAB_DRUGS)

func _on_evidence_tab_button_pressed() -> void:
	_set_results_tab(TAB_EVIDENCE)

func _on_species_tab_button_pressed() -> void:
	_set_results_tab(TAB_SPECIES)

func _on_save_button_pressed() -> void:
	if _current_result_text == "":
		_set_notice("No result is loaded.")
		return
	_output_dialog_mode = "save_result"
	output_dialog.popup_centered_ratio(0.7)

func _on_new_button_pressed() -> void:
	_current_result_text = ""
	_current_result_sample = ""
	_current_result_path = ""
	save_button.disabled = true
	_show_landing_view()
	_set_notice("")
	_set_window_title_default()

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
	var sample := sample_edit.text.strip_edges()
	var panels_dir := panels_dir_edit.text.strip_edges()
	var species := _selected_species()
	var panel_name := _selected_panel()
	if sample == "":
		_set_notice("Sample name is required.")
		options_dialog.popup_centered()
		return
	if reads_path.strip_edges() == "":
		_set_notice("Reads file is required.")
		return
	if panels_dir == "":
		_set_notice("Panels directory is required.")
		options_dialog.popup_centered()
		return
	if species == "":
		_set_notice("Species is required.")
		options_dialog.popup_centered()
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
		"--panels_dir", panels_dir,
		"--output", output_path,
		"--format", "json",
	])
	if panel_name != "":
		args.append_array(["--panel", panel_name])
	if report_all_calls_check.button_pressed:
		args.append("--report_all_calls")
	if ncbi_names_check.button_pressed:
		args.append("--ncbi_names")
	if ont_check.button_pressed:
		args.append("--ont")
	if guess_method_check.button_pressed:
		args.append("--guess_sequence_method")

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
	options_dialog.hide()
	_set_notice("")
	_set_window_title_processing(sample)

func _load_json_result(path: String, preferred_sample: String = "sample") -> void:
	if not FileAccess.file_exists(path):
		_set_notice("Result JSON was not found at %s." % path)
		return
	var file := FileAccess.open(path, FileAccess.READ)
	if file == null:
		_set_notice("Could not open JSON file: %s." % path)
		return
	var text := file.get_as_text()
	file.close()
	var parsed = JSON.parse_string(text)
	if parsed == null:
		_set_notice("JSON parsing failed for %s." % path)
		return
	_current_result_text = text
	_current_result_path = path
	_display_results(preferred_sample, parsed)

func _display_results(sample: String, parsed: Variant) -> void:
	var all_tab: Dictionary = _formatter.format_all_tab(sample, parsed)
	all_susceptible_text.text = str(all_tab.get("susceptible", ""))
	all_resistant_text.text = str(all_tab.get("resistant", ""))
	var drugs_tab: Dictionary = _formatter.format_drugs_tab(sample, parsed)
	first_line_text.text = str(drugs_tab.get("first_line", ""))
	second_line_text.text = str(drugs_tab.get("second_line", ""))
	evidence_text.text = _formatter.format_evidence_tab(sample, parsed)
	species_text.text = _formatter.format_species_tab(sample, parsed)
	save_button.disabled = false
	_show_results_view()
	_set_results_tab(TAB_ALL)
	_set_window_title_results(sample, TAB_ALL)

func _show_landing_view() -> void:
	landing_view.visible = true
	bootstrap_view.visible = false
	app_view.visible = false

func _show_bootstrap_view() -> void:
	landing_view.visible = false
	bootstrap_view.visible = true
	app_view.visible = false

func _show_results_view() -> void:
	landing_view.visible = false
	bootstrap_view.visible = false
	app_view.visible = true

func _refresh_setup_state() -> void:
	var panels_dir := panels_dir_edit.text.strip_edges()
	var manifest_exists := FileAccess.file_exists(panels_dir.path_join("manifest.json"))
	if _panels_setup.is_running() or not manifest_exists:
		if _panels_setup.is_running():
			bootstrap_status_label.text = "Downloading panel metadata and species data. This can take a little while."
		else:
			bootstrap_status_label.text = "Panel metadata is missing. Mykrobe is downloading all species into the shared panels directory."
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
	var panels_dir := panels_dir_edit.text.strip_edges()
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
	], "All species panels are ready.")
	if not start_result.get("started", false):
		_set_notice(str(start_result.get("error", "Could not start panel setup.")))

func _selected_species() -> String:
	if species_option.item_count == 0:
		return ""
	var idx := species_option.selected
	if idx < 0 or idx >= _species_entries.size():
		return ""
	return str(_species_entries[idx].get("species", "")).strip_edges()

func _selected_panel() -> String:
	if panel_option.item_count == 0:
		return ""
	var idx := panel_option.selected
	if idx < 0 or idx >= _panel_entries.size():
		return ""
	return str(_panel_entries[idx].get("name", "")).strip_edges()

func _refresh_species_options() -> void:
	_species_entries = _helpers.load_species_entries(_resolve_binary_path(), panels_dir_edit.text.strip_edges())
	species_option.clear()
	species_option.disabled = true
	panel_option.clear()
	panel_option.disabled = true
	_panel_entries.clear()
	if _species_entries.is_empty():
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
	_refresh_panel_options(_species_entries[preferred_index])
	analyse_button.disabled = false

func _refresh_panel_options(species_entry: Dictionary) -> void:
	_panel_entries.clear()
	panel_option.clear()
	panel_option.disabled = true
	var panels_variant: Variant = species_entry.get("panels", [])
	if typeof(panels_variant) != TYPE_ARRAY:
		return
	var default_panel := str(species_entry.get("default_panel", ""))
	for panel_variant in panels_variant:
		if typeof(panel_variant) != TYPE_DICTIONARY:
			continue
		var panel_entry: Dictionary = panel_variant
		var panel_name := str(panel_entry.get("name", "")).strip_edges()
		if panel_name == "":
			continue
		_panel_entries.append(panel_entry)
		panel_option.add_item(panel_name)
	if panel_option.item_count == 0:
		return
	panel_option.disabled = false
	var preferred_index := 0
	for i in range(_panel_entries.size()):
		if str(_panel_entries[i].get("name", "")) == default_panel:
			preferred_index = i
			break
	panel_option.select(preferred_index)

func _resolve_binary_path() -> String:
	var from_env := OS.get_environment("MYKROBE2_BINARY").strip_edges()
	if from_env != "" and FileAccess.file_exists(from_env):
		return from_env
	if _local_mykrobe2_manager != null and _local_mykrobe2_manager.ensure_local_binary_installed():
		return _local_mykrobe2_manager.installed_binary_path()
	return ""

func _set_results_tab(tab_index: int) -> void:
	_current_tab = tab_index
	all_view.visible = tab_index == TAB_ALL
	drugs_view.visible = tab_index == TAB_DRUGS
	evidence_view.visible = tab_index == TAB_EVIDENCE
	species_view.visible = tab_index == TAB_SPECIES
	all_tab_button.button_pressed = tab_index == TAB_ALL
	drugs_tab_button.button_pressed = tab_index == TAB_DRUGS
	evidence_tab_button.button_pressed = tab_index == TAB_EVIDENCE
	species_tab_button.button_pressed = tab_index == TAB_SPECIES
	_apply_tab_styles()
	if _current_result_sample != "":
		_set_window_title_results(_current_result_sample, tab_index)

func _apply_tab_styles() -> void:
	var selected_bg := Color(0.23, 0.53, 0.70, 1.0)
	if _palette.has("accent"):
		selected_bg = _palette["accent"]
	var selected_fg := Color(1, 1, 1, 1)
	var unselected_fg := Color(0.23, 0.53, 0.70, 1)
	if _palette.has("accent"):
		unselected_fg = _palette["accent"]
	for button in [all_tab_button, drugs_tab_button, evidence_tab_button, species_tab_button]:
		button.flat = not button.button_pressed
		button.add_theme_color_override("font_color", selected_fg if button.button_pressed else unselected_fg)
		button.add_theme_color_override("font_hover_color", selected_fg if button.button_pressed else unselected_fg)
		button.add_theme_color_override("font_pressed_color", selected_fg)
		var style := StyleBoxFlat.new()
		style.corner_radius_top_left = 18
		style.corner_radius_top_right = 18
		style.corner_radius_bottom_left = 18
		style.corner_radius_bottom_right = 18
		style.content_margin_left = 16
		style.content_margin_right = 16
		style.content_margin_top = 8
		style.content_margin_bottom = 8
		if button.button_pressed:
			style.bg_color = selected_bg
			style.border_width_left = 0
			style.border_width_top = 0
			style.border_width_right = 0
			style.border_width_bottom = 0
		else:
			style.bg_color = Color(1, 1, 1, 0)
			style.border_color = Color(0.23, 0.53, 0.70, 0.18)
			style.border_width_left = 0
			style.border_width_top = 0
			style.border_width_right = 0
			style.border_width_bottom = 0
		button.add_theme_stylebox_override("normal", style)
		button.add_theme_stylebox_override("hover", style)
		button.add_theme_stylebox_override("pressed", style)
		button.add_theme_stylebox_override("focus", style)

func _set_notice(message: String) -> void:
	status_label.visible = message.strip_edges() != ""
	status_label.text = message

func _set_window_title_default() -> void:
	get_window().title = "Mykrobe"

func _set_window_title_processing(sample: String) -> void:
	get_window().title = "%s - Analysing - Mykrobe" % sample

func _set_window_title_results(sample: String, tab_index: int) -> void:
	var tab_name := "All"
	match tab_index:
		TAB_DRUGS:
			tab_name = "Drugs"
		TAB_EVIDENCE:
			tab_name = "Evidence"
		TAB_SPECIES:
			tab_name = "Species"
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
		_load_json_result(path, sample_edit.text.strip_edges())
		_set_notice("Loaded result JSON from %s." % path)
		return
	_active_reads_path = path
	if sample_edit.text.strip_edges() == "" or sample_edit.text == "sample":
		sample_edit.text = path.get_file().get_basename()
	_start_predict(path)
