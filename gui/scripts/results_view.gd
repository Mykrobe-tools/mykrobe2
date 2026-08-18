extends Control
class_name ResultsView

signal save_requested
signal new_requested
signal tab_changed(tab_name: String)

const ResultFormatterScript = preload("res://scripts/result_formatter.gd")

const TAB_ALL := 0
const TAB_DRUGS := 1
const TAB_EVIDENCE := 2
const TAB_SPECIES := 3

@onready var header_bar: ColorRect = $HeaderBar
@onready var header_logo_icon: TextureRect = $HeaderBar/HeaderMargin/HeaderHBox/HeaderLogo/HeaderLogoIcon
@onready var header_logo_text: Label = $HeaderBar/HeaderMargin/HeaderHBox/HeaderLogo/HeaderLogoText
@onready var all_tab_button: Button = $HeaderBar/HeaderMargin/HeaderHBox/TabsRow/AllTabButton
@onready var drugs_tab_button: Button = $HeaderBar/HeaderMargin/HeaderHBox/TabsRow/DrugsTabButton
@onready var evidence_tab_button: Button = $HeaderBar/HeaderMargin/HeaderHBox/TabsRow/EvidenceTabButton
@onready var species_tab_button: Button = $HeaderBar/HeaderMargin/HeaderHBox/TabsRow/SpeciesTabButton
@onready var save_button: Button = $HeaderBar/HeaderMargin/HeaderHBox/SaveButton
@onready var all_view: Control = $ResultsMargin/ResultsStack/AllView
@onready var drugs_view: Control = $ResultsMargin/ResultsStack/DrugsView
@onready var evidence_view: Control = $ResultsMargin/ResultsStack/EvidenceView
@onready var species_view: Control = $ResultsMargin/ResultsStack/SpeciesView
@onready var all_title: Label = $ResultsMargin/ResultsStack/AllView/AllVBox/AllTitle
@onready var susceptible_heading: Label = $ResultsMargin/ResultsStack/AllView/AllVBox/AllColumns/AllSusceptibleColumn/AllSusceptibleHeading
@onready var resistant_heading: Label = $ResultsMargin/ResultsStack/AllView/AllVBox/AllColumns/AllResistantColumn/AllResistantHeading
@onready var all_susceptible_text: RichTextLabel = $ResultsMargin/ResultsStack/AllView/AllVBox/AllColumns/AllSusceptibleColumn/AllSusceptibleText
@onready var all_resistant_text: RichTextLabel = $ResultsMargin/ResultsStack/AllView/AllVBox/AllColumns/AllResistantColumn/AllResistantText
@onready var first_line_title: Label = $ResultsMargin/ResultsStack/DrugsView/DrugsVBox/DrugsColumns/FirstLineColumn/FirstLineTitle
@onready var second_line_title: Label = $ResultsMargin/ResultsStack/DrugsView/DrugsVBox/DrugsColumns/SecondLineColumn/SecondLineTitle
@onready var first_line_text: RichTextLabel = $ResultsMargin/ResultsStack/DrugsView/DrugsVBox/DrugsColumns/FirstLineColumn/FirstLineText
@onready var second_line_text: RichTextLabel = $ResultsMargin/ResultsStack/DrugsView/DrugsVBox/DrugsColumns/SecondLineColumn/SecondLineText
@onready var evidence_title: Label = $ResultsMargin/ResultsStack/EvidenceView/EvidenceVBox/EvidenceTitle
@onready var evidence_text: RichTextLabel = $ResultsMargin/ResultsStack/EvidenceView/EvidenceVBox/EvidenceText
@onready var species_title: Label = $ResultsMargin/ResultsStack/SpeciesView/SpeciesVBox/SpeciesTitle
@onready var species_text: RichTextLabel = $ResultsMargin/ResultsStack/SpeciesView/SpeciesVBox/SpeciesText

var _formatter: RefCounted
var _palette: Dictionary = {}
var _sample := ""

func _ready() -> void:
	_formatter = ResultFormatterScript.new()
	clear()

func set_logo_texture(texture: Texture2D) -> void:
	header_logo_icon.texture = texture

func display(sample: String, parsed: Variant) -> void:
	_sample = sample
	var all_tab: Dictionary = _formatter.format_all_tab(sample, parsed)
	all_susceptible_text.text = str(all_tab.get("susceptible", ""))
	all_resistant_text.text = str(all_tab.get("resistant", ""))
	var drugs_tab: Dictionary = _formatter.format_drugs_tab(sample, parsed)
	first_line_text.text = str(drugs_tab.get("first_line", ""))
	second_line_text.text = str(drugs_tab.get("second_line", ""))
	evidence_text.text = _formatter.format_evidence_tab(sample, parsed)
	species_text.text = _formatter.format_species_tab(sample, parsed)
	save_button.disabled = false
	_set_results_tab(TAB_ALL)

func clear() -> void:
	_sample = ""
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

func _set_results_tab(tab_index: int) -> void:
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
	var selected_bg: Color = _palette.get("accent", Color("3987b5"))
	var selected_fg := Color(1, 1, 1, 1)
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
