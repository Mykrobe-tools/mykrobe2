extends Control
class_name ChoosePanelDialog

signal panel_selected(species: String, panel: String)

@onready var options_card: PanelContainer = $OptionsCenter/OptionsCard
@onready var options_scrim: ColorRect = $OptionsScrim
@onready var species_option: OptionButton = $OptionsCenter/OptionsCard/OptionsMargin/OptionsVBox/SpeciesRow/SpeciesOption
@onready var panel_option: OptionButton = $OptionsCenter/OptionsCard/OptionsMargin/OptionsVBox/PanelRow/PanelOption

var _species_entries: Array = []
var _displayed_species_entries: Array = []
var _panel_entries: Array = []
var _selected_species := ""
var _selected_panel := ""

func open_dialog(species_entries: Array, selected_species: String, selected_panel: String) -> void:
	_species_entries = species_entries
	_selected_species = selected_species
	_selected_panel = selected_panel
	_populate_species_options()
	visible = true

func apply_palette(palette: Dictionary) -> void:
	options_scrim.color = palette.get("scrim", Color(0.972549, 0.960784, 0.933333, 0.55))
	var modal_style := StyleBoxFlat.new()
	modal_style.bg_color = palette.get("panel_alt", Color("fbf9f3"))
	modal_style.border_color = palette.get("border", Color("ddd7ca"))
	modal_style.set_border_width_all(1)
	modal_style.set_corner_radius_all(14)
	options_card.add_theme_stylebox_override("panel", modal_style)

func _populate_species_options() -> void:
	species_option.clear()
	species_option.disabled = true
	panel_option.clear()
	panel_option.disabled = true
	_displayed_species_entries.clear()
	_panel_entries.clear()
	for entry_variant in _species_entries:
		if typeof(entry_variant) != TYPE_DICTIONARY:
			continue
		var entry: Dictionary = entry_variant
		var species_name := str(entry.get("species", "")).strip_edges()
		if species_name == "":
			continue
		_displayed_species_entries.append(entry)
		species_option.add_item(species_name)
	if _displayed_species_entries.is_empty():
		return
	species_option.disabled = false
	var selected_index := _find_species_index(_selected_species)
	if selected_index < 0:
		selected_index = 0
	species_option.select(selected_index)
	_populate_panel_options(_displayed_species_entries[selected_index], _selected_panel)

func _populate_panel_options(species_entry: Dictionary, preferred_panel: String) -> void:
	panel_option.clear()
	panel_option.disabled = true
	_panel_entries.clear()
	var panels_variant: Variant = species_entry.get("panels", [])
	if typeof(panels_variant) != TYPE_ARRAY:
		return
	for panel_variant in panels_variant:
		if typeof(panel_variant) != TYPE_DICTIONARY:
			continue
		var panel_entry: Dictionary = panel_variant
		var panel_name := str(panel_entry.get("name", "")).strip_edges()
		if panel_name == "":
			continue
		_panel_entries.append(panel_entry)
		panel_option.add_item(panel_name)
	if _panel_entries.is_empty():
		return
	panel_option.disabled = false
	var selected_index := _find_panel_index(preferred_panel)
	if selected_index < 0:
		selected_index = _find_panel_index(str(species_entry.get("default_panel", "")))
	if selected_index < 0:
		selected_index = 0
	panel_option.select(selected_index)

func _find_species_index(species: String) -> int:
	for index in range(_displayed_species_entries.size()):
		if str(_displayed_species_entries[index].get("species", "")).strip_edges() == species:
			return index
	return -1

func _find_panel_index(panel: String) -> int:
	for index in range(_panel_entries.size()):
		if str(_panel_entries[index].get("name", "")).strip_edges() == panel:
			return index
	return -1

func _current_species() -> String:
	var index := species_option.selected
	if index < 0 or index >= _displayed_species_entries.size():
		return ""
	return str(_displayed_species_entries[index].get("species", "")).strip_edges()

func _current_panel() -> String:
	var index := panel_option.selected
	if index < 0 or index >= _panel_entries.size():
		return ""
	return str(_panel_entries[index].get("name", "")).strip_edges()

func _on_species_option_item_selected(index: int) -> void:
	if index < 0 or index >= _displayed_species_entries.size():
		return
	_populate_panel_options(_displayed_species_entries[index], "")

func _on_cancel_button_pressed() -> void:
	visible = false

func _on_set_panel_button_pressed() -> void:
	panel_selected.emit(_current_species(), _current_panel())
	visible = false
