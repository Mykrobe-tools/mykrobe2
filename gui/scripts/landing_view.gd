extends Control
class_name LandingView

signal analyse_requested(paths: PackedStringArray)
signal change_requested

@onready var circle: PanelContainer = $LandingCenter/LandingCard/LandingCircle
@onready var logo_icon: TextureRect = $LandingCenter/LandingCard/LandingMargin/LandingVBox/LandingLogo/LandingLogoIcon
@onready var logo_text: Label = $LandingCenter/LandingCard/LandingMargin/LandingVBox/LandingLogo/LandingLogoText
@onready var tagline: Label = $LandingCenter/LandingCard/LandingMargin/LandingVBox/LandingTagline
@onready var selection_label: Label = $LandingCenter/LandingCard/LandingMargin/LandingVBox/LandingSelectionRow/LandingSelection
@onready var empty_files_row: HBoxContainer = $LandingCenter/LandingCard/LandingMargin/LandingVBox/InputFiles/EmptyFilesRow
@onready var files_list: ItemList = $LandingCenter/LandingCard/LandingMargin/LandingVBox/InputFiles/FilesList
@onready var files_actions: HBoxContainer = $LandingCenter/LandingCard/LandingMargin/LandingVBox/InputFiles/FilesActions
@onready var remove_file_button: Button = $LandingCenter/LandingCard/LandingMargin/LandingVBox/InputFiles/FilesActions/RemoveFileButton
@onready var analyse_button: Button = $LandingCenter/LandingCard/LandingMargin/LandingVBox/LandingButtons/AnalyseButton
@onready var file_dialog: FileDialog = $FileDialog

var _paths := PackedStringArray()
var _panel_available := false

func set_logo_texture(texture: Texture2D) -> void:
	logo_icon.texture = texture

func set_panel(species: String, panel: String) -> void:
	if species == "":
		selection_label.text = "No panel selected"
	elif panel == "":
		selection_label.text = species.to_upper()
	else:
		selection_label.text = "%s · panel %s" % [species.to_upper(), panel]

func set_analysis_enabled(enabled: bool) -> void:
	_panel_available = enabled
	_update_analysis_enabled()

func add_files(paths: PackedStringArray) -> void:
	for path in paths:
		var clean_path := path.strip_edges()
		if clean_path == "" or not FileAccess.file_exists(clean_path) or _paths.has(clean_path):
			continue
		_paths.append(clean_path)
	_refresh_files()

func clear_files() -> void:
	_paths.clear()
	_refresh_files()

func apply_palette(palette: Dictionary) -> void:
	var circle_style := StyleBoxFlat.new()
	circle_style.bg_color = palette.get("circle_bg", Color(1, 1, 1, 0.92))
	circle_style.set_corner_radius_all(400)
	circle.add_theme_stylebox_override("panel", circle_style)
	logo_text.add_theme_color_override("font_color", palette.get("accent", Color("3987b5")))
	tagline.add_theme_color_override("font_color", palette.get("text", Color("6d6a65")))

func _on_analyse_button_pressed() -> void:
	if _paths.is_empty() or not _panel_available:
		return
	analyse_requested.emit(_paths.duplicate())

func _on_change_button_pressed() -> void:
	change_requested.emit()

func _on_add_file_button_pressed() -> void:
	file_dialog.popup_centered_ratio(0.75)

func _on_files_selected(paths: PackedStringArray) -> void:
	add_files(paths)

func _on_files_list_multi_selected(_index: int, _selected: bool) -> void:
	remove_file_button.disabled = files_list.get_selected_items().is_empty()

func _on_remove_file_button_pressed() -> void:
	var selected := files_list.get_selected_items()
	if selected.is_empty():
		return
	var remaining := PackedStringArray()
	for index in range(_paths.size()):
		if not selected.has(index):
			remaining.append(_paths[index])
	_paths = remaining
	_refresh_files()

func _refresh_files() -> void:
	files_list.clear()
	for path in _paths:
		files_list.add_item(path.get_file())
		files_list.set_item_tooltip(files_list.item_count - 1, path)
	var has_files := not _paths.is_empty()
	empty_files_row.visible = not has_files
	files_list.visible = has_files
	files_actions.visible = has_files
	remove_file_button.disabled = true
	_update_analysis_enabled()

func _update_analysis_enabled() -> void:
	analyse_button.disabled = not _panel_available or _paths.is_empty()
