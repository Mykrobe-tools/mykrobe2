extends Control
class_name ResultsView

signal save_requested
signal new_requested
signal tab_changed(tab_name: String)
signal settings_requested

const ResultFormatterScript = preload("res://scripts/result_formatter.gd")

const TAB_ALL := 0
const TAB_DRUGS := 1
const TAB_EVIDENCE := 2
const TAB_SPECIES := 3

@onready var header_bar: ColorRect = $RootLayout/HeaderBar
@onready var settings_button: Button = $RootLayout/HeaderBar/HeaderMargin/HeaderHBox/SettingsButton
@onready var header_logo_icon: TextureRect = $RootLayout/HeaderBar/HeaderMargin/HeaderHBox/HeaderLogo/HeaderLogoIcon
@onready var header_logo_text: Label = $RootLayout/HeaderBar/HeaderMargin/HeaderHBox/HeaderLogo/HeaderLogoText
@onready var all_tab_button: Button = $RootLayout/HeaderBar/HeaderMargin/HeaderHBox/TabsRow/AllTabButton
@onready var drugs_tab_button: Button = $RootLayout/HeaderBar/HeaderMargin/HeaderHBox/TabsRow/DrugsTabButton
@onready var evidence_tab_button: Button = $RootLayout/HeaderBar/HeaderMargin/HeaderHBox/TabsRow/EvidenceTabButton
@onready var species_tab_button: Button = $RootLayout/HeaderBar/HeaderMargin/HeaderHBox/TabsRow/SpeciesTabButton
@onready var save_button: Button = $RootLayout/HeaderBar/HeaderMargin/HeaderHBox/SaveButton
@onready var all_view: Control = $RootLayout/ResultsMargin/ResultsStack/AllView
@onready var drugs_view: Control = $RootLayout/ResultsMargin/ResultsStack/DrugsView
@onready var evidence_view: Control = $RootLayout/ResultsMargin/ResultsStack/EvidenceView
@onready var species_view: Control = $RootLayout/ResultsMargin/ResultsStack/SpeciesView
@onready var all_title: Label = $RootLayout/ResultsMargin/ResultsStack/AllView/AllScroll/AllVBox/AllTitle
@onready var susceptible_heading: Label = $RootLayout/ResultsMargin/ResultsStack/AllView/AllScroll/AllVBox/AllColumns/AllSusceptibleColumn/AllSusceptibleHeading
@onready var resistant_heading: Label = $RootLayout/ResultsMargin/ResultsStack/AllView/AllScroll/AllVBox/AllColumns/AllResistantColumn/AllResistantHeading
@onready var all_susceptible_text: RichTextLabel = $RootLayout/ResultsMargin/ResultsStack/AllView/AllScroll/AllVBox/AllColumns/AllSusceptibleColumn/AllSusceptibleText
@onready var all_resistant_text: RichTextLabel = $RootLayout/ResultsMargin/ResultsStack/AllView/AllScroll/AllVBox/AllColumns/AllResistantColumn/AllResistantText
@onready var first_line_title: Label = $RootLayout/ResultsMargin/ResultsStack/DrugsView/DrugsScroll/DrugsVBox/DrugsColumns/FirstLineColumn/FirstLineTitle
@onready var second_line_title: Label = $RootLayout/ResultsMargin/ResultsStack/DrugsView/DrugsScroll/DrugsVBox/DrugsColumns/SecondLineColumn/SecondLineTitle
@onready var first_line_text: RichTextLabel = $RootLayout/ResultsMargin/ResultsStack/DrugsView/DrugsScroll/DrugsVBox/DrugsColumns/FirstLineColumn/FirstLineText
@onready var second_line_text: RichTextLabel = $RootLayout/ResultsMargin/ResultsStack/DrugsView/DrugsScroll/DrugsVBox/DrugsColumns/SecondLineColumn/SecondLineText
@onready var evidence_title: Label = $RootLayout/ResultsMargin/ResultsStack/EvidenceView/EvidenceScroll/EvidenceVBox/EvidenceTitle
@onready var evidence_text: RichTextLabel = $RootLayout/ResultsMargin/ResultsStack/EvidenceView/EvidenceScroll/EvidenceVBox/EvidenceText
@onready var species_title: Label = $RootLayout/ResultsMargin/ResultsStack/SpeciesView/SpeciesScroll/SpeciesVBox/SpeciesTitle
@onready var species_text: RichTextLabel = $RootLayout/ResultsMargin/ResultsStack/SpeciesView/SpeciesScroll/SpeciesVBox/SpeciesText

var _formatter: RefCounted
var _palette: Dictionary = {}
var _sample := ""
var _parsed_result: Variant = null
var _active_tab := TAB_ALL

func _ready() -> void:
	_formatter = ResultFormatterScript.new()
	clear()

func set_logo_texture(texture: Texture2D) -> void:
	header_logo_icon.texture = texture

func display(sample: String, parsed: Variant) -> void:
	_sample = sample
	_parsed_result = parsed
	_render_result()
	save_button.disabled = false
	_set_results_tab(TAB_ALL)

func _render_result() -> void:
	if _sample == "" or _parsed_result == null:
		return
	var all_tab: Dictionary = _formatter.format_all_tab(_sample, _parsed_result)
	all_susceptible_text.text = str(all_tab.get("susceptible", ""))
	all_resistant_text.text = str(all_tab.get("resistant", ""))
	var drugs_tab: Dictionary = _formatter.format_drugs_tab(_sample, _parsed_result)
	first_line_text.text = str(drugs_tab.get("first_line", ""))
	second_line_text.text = str(drugs_tab.get("second_line", ""))
	evidence_text.text = _formatter.format_evidence_tab(_sample, _parsed_result)
	species_text.text = _formatter.format_species_tab(_sample, _parsed_result)

func clear() -> void:
	_sample = ""
	_parsed_result = null
	all_susceptible_text.text = ""
	all_resistant_text.text = ""
	first_line_text.text = ""
	second_line_text.text = ""
	evidence_text.text = ""
	species_text.text = ""
	save_button.disabled = true
	_set_results_tab(TAB_ALL)

func apply_palette(palette: Dictionary) -> void:
	_palette = palette
	if _formatter != null:
		_formatter.set_palette(palette)
	var accent: Color = palette.get("accent", Color("3987b5"))
	var text: Color = palette.get("text", Color("6d6a65"))
	header_bar.color = palette.get("header_bg", Color(0.97, 0.96, 0.93, 0.95))
	header_logo_text.add_theme_color_override("font_color", accent)
	susceptible_heading.add_theme_color_override("font_color", palette.get("success", Color("78b13f")))
	resistant_heading.add_theme_color_override("font_color", palette.get("danger", Color("f55a32")))
	for label in [all_title, first_line_title, second_line_title, evidence_title, species_title]:
		label.add_theme_color_override("font_color", accent)
	for rich_text in [all_susceptible_text, all_resistant_text, first_line_text, second_line_text, evidence_text, species_text]:
		rich_text.add_theme_color_override("default_color", text)
	_apply_tab_styles()
	_render_result()

func _set_results_tab(tab_index: int) -> void:
	_active_tab = tab_index
	all_view.visible = tab_index == TAB_ALL
	drugs_view.visible = tab_index == TAB_DRUGS
	evidence_view.visible = tab_index == TAB_EVIDENCE
	species_view.visible = tab_index == TAB_SPECIES
	all_tab_button.button_pressed = tab_index == TAB_ALL
	drugs_tab_button.button_pressed = tab_index == TAB_DRUGS
	evidence_tab_button.button_pressed = tab_index == TAB_EVIDENCE
	species_tab_button.button_pressed = tab_index == TAB_SPECIES
	_apply_tab_styles()
	if _sample != "":
		tab_changed.emit(_tab_name(tab_index))

func _apply_tab_styles() -> void:
	var selected_bg: Color = _palette.get("button_pressed", Color("3987b5"))
	var selected_fg: Color = _palette.get("text_inverse", Color(1, 1, 1, 1))
	var unselected_fg: Color = _palette.get("accent", Color("3987b5"))
	for button in [all_tab_button, drugs_tab_button, evidence_tab_button, species_tab_button]:
		button.flat = not button.button_pressed
		button.add_theme_color_override("font_color", selected_fg if button.button_pressed else unselected_fg)
		button.add_theme_color_override("font_hover_color", selected_fg if button.button_pressed else unselected_fg)
		button.add_theme_color_override("font_pressed_color", selected_fg)
		var style := StyleBoxFlat.new()
		style.set_corner_radius_all(20)
		style.content_margin_left = 16
		style.content_margin_right = 16
		style.content_margin_top = 3
		style.content_margin_bottom = 3
		style.bg_color = selected_bg if button.button_pressed else Color(1, 1, 1, 0)
		button.add_theme_stylebox_override("normal", style)
		button.add_theme_stylebox_override("hover", style)
		button.add_theme_stylebox_override("pressed", style)
		button.add_theme_stylebox_override("focus", style)

func _tab_name(tab_index: int) -> String:
	match tab_index:
		TAB_DRUGS:
			return "Drugs"
		TAB_EVIDENCE:
			return "Evidence"
		TAB_SPECIES:
			return "Species"
		_:
			return "All"

func _on_all_tab_button_pressed() -> void:
	_set_results_tab(TAB_ALL)

func _on_drugs_tab_button_pressed() -> void:
	_set_results_tab(TAB_DRUGS)

func _on_evidence_tab_button_pressed() -> void:
	_set_results_tab(TAB_EVIDENCE)

func _on_species_tab_button_pressed() -> void:
	_set_results_tab(TAB_SPECIES)

func _on_save_button_pressed() -> void:
	save_requested.emit()

func _on_new_button_pressed() -> void:
	new_requested.emit()

func _on_settings_button_pressed() -> void:
	settings_requested.emit()
