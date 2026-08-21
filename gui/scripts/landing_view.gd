extends Control
class_name LandingView

signal analyse_requested(paths: PackedStringArray)
signal change_requested

@onready var circle: PanelContainer = $LandingCenter/LandingCard/LandingCircle
@onready var logo_icon: TextureRect = $LandingCenter/LandingCard/LandingMargin/LandingVBox/LandingLogo/LandingLogoIcon
@onready var logo_text: Label = $LandingCenter/LandingCard/LandingMargin/LandingVBox/LandingLogo/LandingLogoText
@onready var tagline: Label = $LandingCenter/LandingCard/LandingMargin/LandingVBox/LandingTagline
@onready var selection_label: Label = $LandingCenter/LandingCard/LandingMargin/LandingVBox/LandingSelectionRow/LandingSelection
@onready var empty_files_placeholder: CenterContainer = $LandingCenter/LandingCard/LandingMargin/LandingVBox/InputFiles/FilesArea/EmptyFilesPlaceholder
@onready var files_list: ItemList = $LandingCenter/LandingCard/LandingMargin/LandingVBox/InputFiles/FilesArea/FilesList
@onready var clear_files_button: Button = $LandingCenter/LandingCard/LandingMargin/LandingVBox/InputFiles/FilesActions/ClearFilesButton
@onready var analyse_button: Button = $LandingCenter/LandingCard/LandingMargin/LandingVBox/LandingButtons/AnalyseButton
@onready var file_dialog: FileDialog = $FileDialog

var _paths := PackedStringArray()
var _panel_available := false
var _last_input_directory := ""

func circle_diameter() -> float:
	return circle.size.y

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
		_last_input_directory = clean_path.get_base_dir()
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
	var starting_directory := _last_input_directory
	if starting_directory == "":
		starting_directory = _default_input_directory()
	if DirAccess.dir_exists_absolute(starting_directory):
		file_dialog.current_dir = starting_directory
	file_dialog.popup_centered_ratio(0.75)

func _default_input_directory() -> String:
	if OS.get_name() == "Windows":
		return OS.get_system_dir(OS.SYSTEM_DIR_DOCUMENTS)
	var home_directory := OS.get_environment("HOME")
	if home_directory != "":
		return home_directory
	return OS.get_system_dir(OS.SYSTEM_DIR_DOCUMENTS)

func _on_files_selected(paths: PackedStringArray) -> void:
	add_files(paths)

func _on_files_list_item_selected(_index: int) -> void:
	files_list.deselect_all()

func _on_clear_files_button_pressed() -> void:
	clear_files()

func _refresh_files() -> void:
	files_list.clear()
	for path in _paths:
		files_list.add_item(path.get_file())
		var item_index := files_list.item_count - 1
		files_list.set_item_tooltip(item_index, path)
	var has_files := not _paths.is_empty()
	empty_files_placeholder.visible = not has_files
	clear_files_button.disabled = not has_files
	_update_analysis_enabled()

func _update_analysis_enabled() -> void:
	analyse_button.disabled = not _panel_available or _paths.is_empty()
