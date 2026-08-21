extends Control
class_name PanelsInfoDialog

signal update_all_requested

@onready var scrim: ColorRect = $Scrim
@onready var dialog_card: PanelContainer = $DialogCenter/DialogCard
@onready var close_button: Button = $DialogCenter/DialogCard/DialogMargin/DialogLayout/Header/CloseButton
@onready var summary_label: Label = $DialogCenter/DialogCard/DialogMargin/DialogLayout/SummaryLabel
@onready var species_list: ItemList = $DialogCenter/DialogCard/DialogMargin/DialogLayout/Body/SpeciesList
@onready var species_title: Label = $DialogCenter/DialogCard/DialogMargin/DialogLayout/Body/DetailsScroll/Details/SpeciesTitle
@onready var species_status: Label = $DialogCenter/DialogCard/DialogMargin/DialogLayout/Body/DetailsScroll/Details/SpeciesStatus
@onready var default_panel_label: Label = $DialogCenter/DialogCard/DialogMargin/DialogLayout/Body/DetailsScroll/Details/DefaultPanel
@onready var panels_box: VBoxContainer = $DialogCenter/DialogCard/DialogMargin/DialogLayout/Body/DetailsScroll/Details/PanelsBox
@onready var panels_dir_label: Label = $DialogCenter/DialogCard/DialogMargin/DialogLayout/Footer/PanelsDir
@onready var update_confirmation: Control = $UpdateConfirmation
@onready var confirmation_scrim: ColorRect = $UpdateConfirmation/ConfirmationScrim
@onready var confirmation_card: PanelContainer = $UpdateConfirmation/ConfirmationCenter/ConfirmationCard
@onready var confirm_update_button: Button = $UpdateConfirmation/ConfirmationCenter/ConfirmationCard/ConfirmationMargin/ConfirmationLayout/ConfirmationButtons/ConfirmUpdateButton

var _entries: Array = []
var _displayed_entries: Array = []
var _palette: Dictionary = {}

func open_dialog(entries: Array, panels_dir: String, preferred_species: String = "") -> void:
	_entries = entries.duplicate(true)
	panels_dir_label.text = "Data directory: %s" % panels_dir
	panels_dir_label.tooltip_text = panels_dir
	_populate_species_list(preferred_species)
	update_confirmation.visible = false
	visible = true
	close_button.grab_focus()

func apply_palette(palette: Dictionary) -> void:
	_palette = palette
	scrim.color = palette.get("scrim", Color(0.972549, 0.960784, 0.933333, 0.72))
	var modal_style := StyleBoxFlat.new()
	modal_style.bg_color = palette.get("panel_alt", Color("fbf9f3"))
	modal_style.border_color = palette.get("border", Color("ddd7ca"))
	modal_style.set_border_width_all(1)
	modal_style.set_corner_radius_all(14)
	dialog_card.add_theme_stylebox_override("panel", modal_style)
	confirmation_card.add_theme_stylebox_override("panel", modal_style.duplicate())
	confirmation_scrim.color = palette.get("modal_scrim", Color(0, 0, 0, 0.24))
	summary_label.add_theme_color_override("font_color", palette.get("text_muted", Color("8b8478")))
	panels_dir_label.add_theme_color_override("font_color", palette.get("text_muted", Color("8b8478")))
	var selected_items := species_list.get_selected_items()
	if not _displayed_entries.is_empty() and selected_items.size() > 0:
		_show_species(int(selected_items[0]))

func close_dialog() -> void:
	update_confirmation.visible = false
	visible = false

func _populate_species_list(preferred_species: String) -> void:
	species_list.clear()
	_displayed_entries.clear()
	var installed_count := 0
	var panel_count := 0
	var update_count := 0
	var preferred_index := -1
	for entry_variant in _entries:
		if typeof(entry_variant) != TYPE_DICTIONARY:
			continue
		var entry: Dictionary = entry_variant
		var species_name := str(entry.get("species", "")).strip_edges()
		if species_name == "":
			continue
		var installed := bool(entry.get("installed", false))
		var panels_variant: Variant = entry.get("panels", [])
		var count := Array(panels_variant).size() if typeof(panels_variant) == TYPE_ARRAY else 0
		if installed:
			installed_count += 1
			panel_count += count
			if bool(entry.get("update_available", false)):
				update_count += 1
		var status := _short_status(entry)
		species_list.add_item("%s  ·  %s" % [_display_species_name(species_name), status])
		species_list.set_item_tooltip(species_list.item_count - 1, "%d panel%s" % [count, "" if count == 1 else "s"])
		_displayed_entries.append(entry)
		if species_name == preferred_species:
			preferred_index = _displayed_entries.size() - 1
	var update_text := "All installed data is up to date" if update_count == 0 else "%d update%s available" % [update_count, "" if update_count == 1 else "s"]
	summary_label.text = "%d species installed · %d panels · %s" % [installed_count, panel_count, update_text]
	if _displayed_entries.is_empty():
		_clear_details("No panel information is available.")
		return
	var selected_index := preferred_index if preferred_index >= 0 else 0
	species_list.select(selected_index)
	_show_species(selected_index)

func _show_species(index: int) -> void:
	if index < 0 or index >= _displayed_entries.size():
		return
	var entry: Dictionary = _displayed_entries[index]
	var species_name := str(entry.get("species", ""))
	species_title.text = _display_species_name(species_name)
	var installed := bool(entry.get("installed", false))
	var update_available := bool(entry.get("update_available", false))
	var installed_version := str(entry.get("installed_version", "None"))
	var latest_version := str(entry.get("latest_version", ""))
	if not installed:
		species_status.text = "Not installed%s" % (" · Latest %s" % latest_version if latest_version != "" else "")
		species_status.add_theme_color_override("font_color", _palette.get("text_muted", Color("8b8478")))
	elif update_available:
		species_status.text = "Installed %s · Update available%s" % [installed_version, " (%s)" % latest_version if latest_version != "" else ""]
		species_status.add_theme_color_override("font_color", _palette.get("danger", Color("f55a32")))
	else:
		species_status.text = "Installed %s · Up to date" % installed_version
		species_status.add_theme_color_override("font_color", _palette.get("success", Color("78b13f")))
	var default_panel := str(entry.get("default_panel", "")).strip_edges()
	default_panel_label.text = "Default panel: %s" % (default_panel if default_panel != "" else "None")
	_rebuild_panel_cards(entry, default_panel)

func _rebuild_panel_cards(entry: Dictionary, default_panel: String) -> void:
	for child in panels_box.get_children():
		panels_box.remove_child(child)
		child.queue_free()
	var panels_variant: Variant = entry.get("panels", [])
	if typeof(panels_variant) != TYPE_ARRAY or Array(panels_variant).is_empty():
		panels_box.add_child(_make_label("No panels are installed for this species.", 16, true))
		return
	for panel_variant in Array(panels_variant):
		if typeof(panel_variant) != TYPE_DICTIONARY:
			continue
		var panel: Dictionary = panel_variant
		var panel_name := str(panel.get("name", "")).strip_edges()
		var title := panel_name
		if panel_name == default_panel:
			title += "   ·   DEFAULT"
		var card := PanelContainer.new()
		card.add_theme_stylebox_override("panel", _panel_card_style())
		var margin := MarginContainer.new()
		margin.add_theme_constant_override("margin_left", 14)
		margin.add_theme_constant_override("margin_top", 12)
		margin.add_theme_constant_override("margin_right", 14)
		margin.add_theme_constant_override("margin_bottom", 12)
		card.add_child(margin)
		var content := VBoxContainer.new()
		content.add_theme_constant_override("separation", 5)
		margin.add_child(content)
		var title_label := _make_label(title, 17)
		title_label.add_theme_color_override("font_color", _palette.get("accent", Color("3987b5")))
		content.add_child(title_label)
		var reference := str(panel.get("reference", "")).strip_edges()
		var reference_label := _make_label("Reference: %s" % (reference if reference != "" else "Not specified"), 14)
		reference_label.add_theme_color_override("font_color", _palette.get("text_muted", Color("8b8478")))
		content.add_child(reference_label)
		var description := str(panel.get("description", "")).strip_edges()
		if description == "":
			description = "No description is available."
		content.add_child(_make_label(description, 15, true))
		panels_box.add_child(card)

func _panel_card_style() -> StyleBoxFlat:
	var style := StyleBoxFlat.new()
	style.bg_color = _palette.get("panel", Color("ffffff"))
	style.border_color = _palette.get("border", Color("ddd7ca"))
	style.set_border_width_all(1)
	style.set_corner_radius_all(9)
	return style

func _make_label(text_value: String, font_size: int, wrap: bool = false) -> Label:
	var label := Label.new()
	label.text = text_value
	label.add_theme_font_size_override("font_size", font_size)
	if wrap:
		label.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART
	return label

func _clear_details(message: String) -> void:
	species_title.text = message
	species_status.text = ""
	default_panel_label.text = ""
	for child in panels_box.get_children():
		panels_box.remove_child(child)
		child.queue_free()

func _short_status(entry: Dictionary) -> String:
	if not bool(entry.get("installed", false)):
		return "Not installed"
	if bool(entry.get("update_available", false)):
		return "Update available"
	return str(entry.get("installed_version", "Installed"))

func _display_species_name(species_name: String) -> String:
	if species_name.to_lower() == "tb":
		return "TB"
	return species_name.capitalize()

func _on_species_list_item_selected(index: int) -> void:
	_show_species(index)

func _on_close_button_pressed() -> void:
	close_dialog()

func _on_update_all_button_pressed() -> void:
	update_confirmation.visible = true
	confirm_update_button.grab_focus()

func _on_update_confirmation_cancelled() -> void:
	update_confirmation.visible = false

func _on_update_confirmation_confirmed() -> void:
	update_confirmation.visible = false
	update_all_requested.emit()

func _on_confirmation_scrim_gui_input(event: InputEvent) -> void:
	if event is InputEventMouseButton and event.pressed and event.button_index == MOUSE_BUTTON_LEFT:
		_on_update_confirmation_cancelled()

func _on_scrim_gui_input(event: InputEvent) -> void:
	if event is InputEventMouseButton and event.pressed and event.button_index == MOUSE_BUTTON_LEFT:
		close_dialog()

func _unhandled_input(event: InputEvent) -> void:
	if visible and event.is_action_pressed("ui_cancel"):
		if update_confirmation.visible:
			_on_update_confirmation_cancelled()
		else:
			close_dialog()
		get_viewport().set_input_as_handled()
