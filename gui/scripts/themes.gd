extends RefCounted
class_name ThemesLib

const THEMES := {
	"Light": {
		"bg": Color("f8f5ee"),
		"panel": Color("ffffff"),
		"panel_alt": Color("fbf9f3"),
		"border": Color("ddd7ca"),
		"text": Color("6d6a65"),
		"text_muted": Color("8b8478"),
		"text_inverse": Color("ffffff"),
		"accent": Color("3987b5"),
		"button_bg": Color("ffffff"),
		"button_hover": Color("f4fbff"),
		"button_pressed": Color("3987b5"),
		"button_border": Color("b9d6ea"),
		"field_bg": Color("ffffff"),
		"field_border": Color("d8d2c4"),
		"field_focus": Color("3987b5"),
		"status_error": Color("d95b33"),
		"success": Color("78b13f"),
		"danger": Color("f55a32"),
		"dot": Color("c9c4bc"),
		"header_bg": Color("f7f4ed"),
		"circle_bg": Color(1, 1, 1, 0.92),
	}
}

const THEME_ORDER := ["Light"]

func theme_names() -> Array:
	return THEME_ORDER.duplicate()

func has_theme(theme_name: String) -> bool:
	return THEMES.has(theme_name)

func palette(theme_name: String) -> Dictionary:
	return THEMES.get(theme_name, THEMES["Light"]).duplicate(true)

func make_theme(theme_name: String, font_size: int = 16) -> Theme:
	var palette_map: Dictionary = palette(theme_name)
	var theme: Theme = Theme.new()

	var default_font_size: int = maxi(8, font_size)
	theme.default_font_size = default_font_size

	var panel_style: StyleBoxFlat = _panel_style(palette_map["panel"], palette_map["border"], 0, 0)
	var muted_panel_style: StyleBoxFlat = _panel_style(palette_map["panel_alt"], palette_map["border"], 0, 0)
	var line_style: StyleBoxFlat = _line_style(palette_map)
	var focused_line_style: StyleBoxFlat = _line_style(palette_map, true)
	var button_normal: StyleBoxFlat = _button_style(palette_map["button_bg"], palette_map["button_border"], palette_map["text"], false)
	var button_hover: StyleBoxFlat = _button_style(palette_map["button_hover"], palette_map["button_border"], palette_map["accent"], false)
	var button_pressed: StyleBoxFlat = _button_style(palette_map["button_pressed"], palette_map["accent"], palette_map["text_inverse"], true)
	var button_flat: StyleBoxFlat = _button_style(Color(1, 1, 1, 0), Color(1, 1, 1, 0), palette_map["accent"], false)

	theme.set_stylebox("panel", "PanelContainer", panel_style)
	theme.set_stylebox("panel", "AcceptDialog", muted_panel_style)

	theme.set_stylebox("normal", "LineEdit", line_style)
	theme.set_stylebox("focus", "LineEdit", focused_line_style)
	theme.set_stylebox("read_only", "LineEdit", line_style)
	theme.set_color("font_color", "LineEdit", palette_map["text"])
	theme.set_color("font_placeholder_color", "LineEdit", palette_map["text_muted"])

	theme.set_stylebox("normal", "OptionButton", line_style)
	theme.set_stylebox("hover", "OptionButton", line_style)
	theme.set_stylebox("pressed", "OptionButton", focused_line_style)
	theme.set_stylebox("focus", "OptionButton", focused_line_style)
	theme.set_color("font_color", "OptionButton", palette_map["text"])

	theme.set_stylebox("normal", "Button", button_normal)
	theme.set_stylebox("hover", "Button", button_hover)
	theme.set_stylebox("pressed", "Button", button_pressed)
	theme.set_stylebox("focus", "Button", button_hover)
	theme.set_color("font_color", "Button", palette_map["text"])
	theme.set_color("font_hover_color", "Button", palette_map["accent"])
	theme.set_color("font_pressed_color", "Button", palette_map["text_inverse"])
	theme.set_constant("h_separation", "Button", 8)

	theme.set_color("font_color", "Label", palette_map["text"])
	theme.set_color("default_color", "RichTextLabel", palette_map["text"])
	theme.set_color("font_color", "CheckBox", palette_map["text"])
	theme.set_color("font_color", "CheckButton", palette_map["text"])

	theme.set_stylebox("normal", "CheckBox", button_flat)
	theme.set_stylebox("hover", "CheckBox", button_flat)
	theme.set_stylebox("pressed", "CheckBox", button_flat)
	theme.set_stylebox("focus", "CheckBox", button_flat)
	theme.set_stylebox("normal", "CheckButton", button_flat)
	theme.set_stylebox("hover", "CheckButton", button_flat)
	theme.set_stylebox("pressed", "CheckButton", button_flat)
	theme.set_stylebox("focus", "CheckButton", button_flat)

	return theme

func _panel_style(bg: Color, border: Color, radius: int, border_width: int) -> StyleBoxFlat:
	var style := StyleBoxFlat.new()
	style.bg_color = bg
	style.border_color = border
	style.set_border_width_all(border_width)
	style.set_corner_radius_all(radius)
	return style

func _line_style(palette_map: Dictionary, focused: bool = false) -> StyleBoxFlat:
	var style := StyleBoxFlat.new()
	style.bg_color = palette_map["field_bg"]
	style.border_color = palette_map["field_focus"] if focused else palette_map["field_border"]
	style.set_border_width_all(1)
	style.set_corner_radius_all(8)
	style.content_margin_left = 10
	style.content_margin_right = 10
	style.content_margin_top = 7
	style.content_margin_bottom = 7
	return style

func _button_style(bg: Color, border: Color, font_color: Color, filled: bool) -> StyleBoxFlat:
	var style := StyleBoxFlat.new()
	style.bg_color = bg
	style.border_color = border
	style.set_border_width_all(1)
	style.set_corner_radius_all(18)
	style.content_margin_left = 18
	style.content_margin_right = 18
	style.content_margin_top = 10
	style.content_margin_bottom = 10
	if filled:
		style.shadow_size = 0
	return style
