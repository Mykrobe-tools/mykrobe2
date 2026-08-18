extends PanelContainer
class_name SettingsDrawer

signal appearance_changed(mode: String)
signal panel_information_requested

const APPEARANCE_MODES := ["System", "Light", "Dark"]
const DRAWER_WIDTH := 360.0

@onready var close_button: Button = $DrawerMargin/DrawerLayout/Header/CloseButton
@onready var appearance_option: OptionButton = $DrawerMargin/DrawerLayout/Content/AppearanceOption
@onready var appearance_help: Label = $DrawerMargin/DrawerLayout/Content/AppearanceHelp
@onready var panels_summary: Label = $DrawerMargin/DrawerLayout/Content/PanelsSummary
@onready var panels_button: Button = $DrawerMargin/DrawerLayout/Content/PanelsButton
@onready var version_label: Label = $DrawerMargin/DrawerLayout/About/VersionLabel

var _open := false
var _tween: Tween

func _ready() -> void:
	appearance_option.clear()
	for mode in APPEARANCE_MODES:
		appearance_option.add_item(mode)
	_apply_offsets(false)
	visible = false

func is_open() -> bool:
	return _open

func toggle_drawer() -> void:
	if _open:
		close_drawer()
	else:
		open_drawer()

func open_drawer(animated: bool = true) -> void:
	_open = true
	visible = true
	_animate_to(true, animated)
	appearance_option.grab_focus()

func close_drawer(animated: bool = true) -> void:
	_open = false
	_animate_to(false, animated)

func set_appearance(mode: String) -> void:
	var index := APPEARANCE_MODES.find(mode)
	if index < 0:
		index = 0
	appearance_option.select(index)
	_update_appearance_help(APPEARANCE_MODES[index])

func set_app_version(version: String) -> void:
	var clean_version := version.strip_edges()
	if clean_version == "":
		clean_version = "dev"
	version_label.text = "Version %s" % clean_version

func set_panel_entries(entries: Array) -> void:
	var installed_count := 0
	var panel_count := 0
	var update_count := 0
	for entry_variant in entries:
		if typeof(entry_variant) != TYPE_DICTIONARY:
			continue
		var entry: Dictionary = entry_variant
		if bool(entry.get("installed", false)):
			installed_count += 1
			var panels_variant: Variant = entry.get("panels", [])
			if typeof(panels_variant) == TYPE_ARRAY:
				panel_count += Array(panels_variant).size()
			if bool(entry.get("update_available", false)):
				update_count += 1
	if entries.is_empty():
		panels_summary.text = "Panel information is not available."
		panels_button.disabled = true
		return
	panels_button.disabled = false
	var panel_word := "panel" if panel_count == 1 else "panels"
	var status := "Up to date" if update_count == 0 else "%d update%s available" % [update_count, "" if update_count == 1 else "s"]
	panels_summary.text = "%d species installed · %d %s\n%s" % [installed_count, panel_count, panel_word, status]

func apply_palette(palette: Dictionary) -> void:
	var drawer_style := StyleBoxFlat.new()
	drawer_style.bg_color = palette.get("panel_alt", Color("fbf9f3"))
	drawer_style.border_color = palette.get("border", Color("ddd7ca"))
	drawer_style.border_width_right = 1
	add_theme_stylebox_override("panel", drawer_style)
	appearance_help.add_theme_color_override("font_color", palette.get("text_muted", Color("8b8478")))
	panels_summary.add_theme_color_override("font_color", palette.get("text_muted", Color("8b8478")))
	version_label.add_theme_color_override("font_color", palette.get("text_muted", Color("8b8478")))

func _animate_to(open: bool, animated: bool) -> void:
	if _tween != null and _tween.is_running():
		_tween.kill()
	var target_left := 0.0 if open else -DRAWER_WIDTH - 2.0
	var target_right := DRAWER_WIDTH if open else -2.0
	if not animated:
		offset_left = target_left
		offset_right = target_right
		visible = open
		return
	_tween = create_tween()
	_tween.set_trans(Tween.TRANS_CUBIC)
	_tween.set_ease(Tween.EASE_OUT)
	_tween.parallel().tween_property(self, "offset_left", target_left, 0.22)
	_tween.parallel().tween_property(self, "offset_right", target_right, 0.22)
	if not open:
		_tween.finished.connect(func() -> void:
			if not _open:
				visible = false
		, CONNECT_ONE_SHOT)

func _apply_offsets(open: bool) -> void:
	offset_left = 0.0 if open else -DRAWER_WIDTH - 2.0
	offset_right = DRAWER_WIDTH if open else -2.0

func _update_appearance_help(mode: String) -> void:
	match mode:
		"System":
			appearance_help.text = "Uses the operating system's app appearance."
		"Dark":
			appearance_help.text = "Always use the dark appearance."
		_:
			appearance_help.text = "Always use the light appearance."

func _on_close_button_pressed() -> void:
	close_drawer()

func _on_appearance_option_item_selected(index: int) -> void:
	if index < 0 or index >= APPEARANCE_MODES.size():
		return
	var mode: String = APPEARANCE_MODES[index]
	_update_appearance_help(mode)
	appearance_changed.emit(mode)

func _on_panels_button_pressed() -> void:
	panel_information_requested.emit()

func _unhandled_input(event: InputEvent) -> void:
	if _open and event.is_action_pressed("ui_cancel"):
		close_drawer()
		get_viewport().set_input_as_handled()
