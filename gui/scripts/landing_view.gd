extends Control
class_name LandingView

signal analyse_requested
signal change_requested

@onready var circle: PanelContainer = $LandingCenter/LandingCard/LandingCircle
@onready var logo_icon: TextureRect = $LandingCenter/LandingCard/LandingMargin/LandingVBox/LandingLogo/LandingLogoIcon
@onready var logo_text: Label = $LandingCenter/LandingCard/LandingMargin/LandingVBox/LandingLogo/LandingLogoText
@onready var tagline: Label = $LandingCenter/LandingCard/LandingMargin/LandingVBox/LandingTagline
@onready var selection_label: Label = $LandingCenter/LandingCard/LandingMargin/LandingVBox/LandingSelectionRow/LandingSelection
@onready var analyse_button: Button = $LandingCenter/LandingCard/LandingMargin/LandingVBox/LandingButtons/AnalyseButton

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
	analyse_button.disabled = not enabled

func apply_palette(palette: Dictionary) -> void:
	var circle_style := StyleBoxFlat.new()
	circle_style.bg_color = palette.get("circle_bg", Color(1, 1, 1, 0.92))
	circle_style.set_corner_radius_all(400)
	circle.add_theme_stylebox_override("panel", circle_style)
	logo_text.add_theme_color_override("font_color", palette.get("accent", Color("3987b5")))
	tagline.add_theme_color_override("font_color", palette.get("text", Color("6d6a65")))

func _on_analyse_button_pressed() -> void:
	analyse_requested.emit()

func _on_change_button_pressed() -> void:
	change_requested.emit()
