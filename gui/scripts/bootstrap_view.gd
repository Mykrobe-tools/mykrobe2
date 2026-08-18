extends Control
class_name BootstrapView

@onready var circle: PanelContainer = $BootstrapCenter/BootstrapCard/BootstrapCircle
@onready var logo_icon: TextureRect = $BootstrapCenter/BootstrapCard/BootstrapMargin/BootstrapVBox/BootstrapLogo/BootstrapLogoIcon
@onready var logo_text: Label = $BootstrapCenter/BootstrapCard/BootstrapMargin/BootstrapVBox/BootstrapLogo/BootstrapLogoText
@onready var title_label: Label = $BootstrapCenter/BootstrapCard/BootstrapMargin/BootstrapVBox/BootstrapTitle
@onready var status_label: Label = $BootstrapCenter/BootstrapCard/BootstrapMargin/BootstrapVBox/BootstrapStatus
@onready var log_text: RichTextLabel = $BootstrapCenter/BootstrapCard/BootstrapMargin/BootstrapVBox/BootstrapLog

func set_logo_texture(texture: Texture2D) -> void:
	logo_icon.texture = texture

func set_status(message: String) -> void:
	status_label.text = message

func set_log(message: String) -> void:
	log_text.text = message

func apply_palette(palette: Dictionary) -> void:
	var circle_style := StyleBoxFlat.new()
	circle_style.bg_color = palette.get("circle_bg", Color(1, 1, 1, 0.92))
	circle_style.set_corner_radius_all(400)
	circle.add_theme_stylebox_override("panel", circle_style)
	logo_text.add_theme_color_override("font_color", palette.get("accent", Color("3987b5")))
	title_label.add_theme_color_override("font_color", palette.get("text", Color("6d6a65")))
	status_label.add_theme_color_override("font_color", palette.get("text_muted", Color("8b8478")))
	log_text.add_theme_color_override("default_color", palette.get("text", Color("6d6a65")))
