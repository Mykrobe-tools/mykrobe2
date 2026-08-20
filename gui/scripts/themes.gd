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
		"scrollbar_outline": Color("b9d6ea"),
		"accent": Color("3987b5"),
		"button_bg": Color("ffffff"),
		"button_hover": Color("f4fbff"),
		"button_pressed": Color("3987b5"),
		"button_border": Color("b9d6ea"),
		"button_disabled_bg": Color("ffffff"),
		"button_disabled_border": Color("c9c4bc"),
		"field_bg": Color("ffffff"),
		"field_border": Color("d8d2c4"),
		"field_focus": Color("3987b5"),
		"selection_bg": Color("e9f3f8"),
		"selection_border": Color("b9d6ea"),
		"status_error": Color("d95b33"),
		"success": Color("78b13f"),
		"danger": Color("f55a32"),
		"dot": Color("c9c4bc"),
		"header_bg": Color("f7f4ed"),
		"circle_bg": Color(1, 1, 1, 0.92),
		"scrim": Color(0.972549, 0.960784, 0.933333, 0.72),
	},
	"Dark": {
		"bg": Color("15191c"),
		"panel": Color("20262a"),
		"panel_alt": Color("252c31"),
		"border": Color("3b464c"),
		"text": Color("dedbd4"),
		"text_muted": Color("aaa49a"),
		"text_inverse": Color("ffffff"),
		"scrollbar_outline": Color("496675"),
		"accent": Color("65b2dc"),
		"button_bg": Color("20262a"),
		"button_hover": Color("293840"),
		"button_pressed": Color("3987b5"),
		"button_border": Color("496675"),
		"button_disabled_bg": Color("1b2024"),
		"button_disabled_border": Color("354047"),
		"field_bg": Color("1b2024"),
		"field_border": Color("465159"),
		"field_focus": Color("65b2dc"),
		"selection_bg": Color("263c48"),
		"selection_border": Color("4c819e"),
		"status_error": Color("ff8a65"),
		"success": Color("9bd364"),
		"danger": Color("ff795a"),
		"dot": Color("65717a"),
		"header_bg": Color(0.105882, 0.12549, 0.137255, 0.96),
		"circle_bg": Color(0.12549, 0.14902, 0.164706, 0.94),
		"scrim": Color(0.035294, 0.043137, 0.047059, 0.78),
	}
}

const THEME_ORDER := ["Light", "Dark"]

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

	var panel_style: StyleBoxFlat = _panel_style(palette_map["panel"], palette_map["border"], 10, 1)
	var muted_panel_style: StyleBoxFlat = _panel_style(palette_map["panel_alt"], palette_map["border"], 10, 1)
	var line_style: StyleBoxFlat = _line_style(palette_map)
	var focused_line_style: StyleBoxFlat = _line_style(palette_map, true)
	var button_normal: StyleBoxFlat = _button_style(palette_map["button_bg"], palette_map["button_border"], false)
	var button_hover: StyleBoxFlat = _button_style(palette_map["button_hover"], palette_map["button_border"], false)
	var button_pressed: StyleBoxFlat = _button_style(palette_map["button_pressed"], palette_map["accent"], true)
	var button_disabled: StyleBoxFlat = _button_style(palette_map["button_disabled_bg"], palette_map["button_disabled_border"], false)
	var button_flat: StyleBoxFlat = _button_style(Color(1, 1, 1, 0), Color(1, 1, 1, 0), false)
	var list_selected: StyleBoxFlat = _panel_style(palette_map["selection_bg"], palette_map["selection_border"], 6, 1)
	var popup_panel_style: StyleBoxFlat = _panel_style(palette_map["field_bg"], palette_map["field_border"], 8, 1)
	var popup_hover_style := StyleBoxFlat.new()
	popup_hover_style.bg_color = palette_map["button_hover"]
	popup_hover_style.border_color = palette_map["button_hover"]
	popup_hover_style.set_border_width_all(0)
	popup_hover_style.set_corner_radius_all(4)
	var option_arrow := _make_arrow_icon(10, 7, palette_map["text"])
	var window_border := StyleBoxFlat.new()
	window_border.bg_color = Color(0, 0, 0, 0)
	window_border.border_color = palette_map["border"]
	window_border.set_border_width_all(1)
	var window_border_unfocused := window_border.duplicate()
	window_border_unfocused.border_color = palette_map["field_border"]
	var separator_style := StyleBoxFlat.new()
	separator_style.bg_color = palette_map["border"]
	separator_style.content_margin_top = 1
	separator_style.content_margin_bottom = 0

	theme.set_stylebox("panel", "Panel", panel_style)
	theme.set_stylebox("panel", "PanelContainer", panel_style)
	theme.set_stylebox("panel", "AcceptDialog", muted_panel_style)
	theme.set_stylebox("panel", "PopupPanel", popup_panel_style)
	theme.set_stylebox("panel", "PopupMenu", popup_panel_style)
	theme.set_stylebox("panel", "Window", muted_panel_style)
	theme.set_stylebox("embedded_border", "Window", window_border)
	theme.set_stylebox("embedded_unfocused_border", "Window", window_border_unfocused)
	theme.set_stylebox("hover", "PopupMenu", popup_hover_style)
	theme.set_color("title_color", "Window", palette_map["text"])
	theme.set_color("title_outline_modulate", "Window", Color(0, 0, 0, 0))
	theme.set_color("title_pressed_color", "Window", palette_map["text"])
	theme.set_color("title_focus_color", "Window", palette_map["text"])

	theme.set_stylebox("normal", "LineEdit", line_style)
	theme.set_stylebox("focus", "LineEdit", focused_line_style)
	theme.set_stylebox("read_only", "LineEdit", line_style)
	theme.set_color("font_color", "LineEdit", palette_map["text"])
	theme.set_color("caret_color", "LineEdit", palette_map["text"])
	theme.set_color("font_placeholder_color", "LineEdit", palette_map["text_muted"])

	theme.set_stylebox("normal", "OptionButton", line_style)
	theme.set_stylebox("hover", "OptionButton", line_style)
	theme.set_stylebox("pressed", "OptionButton", focused_line_style)
	theme.set_stylebox("focus", "OptionButton", focused_line_style)
	theme.set_color("font_color", "OptionButton", palette_map["text"])
	theme.set_color("font_focus_color", "OptionButton", palette_map["text"])
	theme.set_color("font_hover_color", "OptionButton", palette_map["text"])
	theme.set_color("font_pressed_color", "OptionButton", palette_map["text"])
	theme.set_color("font_disabled_color", "OptionButton", palette_map["text_muted"])
	theme.set_icon("arrow", "OptionButton", option_arrow)
	theme.set_color("modulate_arrow", "OptionButton", palette_map["text"])
	theme.set_constant("arrow_margin", "OptionButton", 10)

	theme.set_stylebox("normal", "Button", button_normal)
	theme.set_stylebox("hover", "Button", button_hover)
	theme.set_stylebox("pressed", "Button", button_pressed)
	theme.set_stylebox("focus", "Button", button_hover)
	theme.set_stylebox("disabled", "Button", button_disabled)
	theme.set_color("font_color", "Button", palette_map["text"])
	theme.set_color("font_hover_color", "Button", palette_map["accent"])
	theme.set_color("font_pressed_color", "Button", palette_map["text_inverse"])
	theme.set_color("font_focus_color", "Button", palette_map["text"])
	theme.set_color("font_disabled_color", "Button", palette_map["text_muted"])
	theme.set_constant("h_separation", "Button", 8)

	theme.set_stylebox("panel", "ItemList", line_style)
	theme.set_stylebox("focus", "ItemList", focused_line_style)
	theme.set_stylebox("selected", "ItemList", list_selected)
	theme.set_stylebox("selected_focus", "ItemList", list_selected)
	theme.set_color("font_color", "ItemList", palette_map["text"])
	theme.set_color("font_hovered_color", "ItemList", palette_map["text"])
	theme.set_color("font_selected_color", "ItemList", palette_map["text"])
	theme.set_color("font_hovered_selected_color", "ItemList", palette_map["text"])
	theme.set_color("guide_color", "ItemList", palette_map["border"])
	theme.set_constant("v_separation", "ItemList", 4)

	theme.set_color("font_color", "Label", palette_map["text"])
	theme.set_color("font_color", "RichTextLabel", palette_map["text"])
	theme.set_color("default_color", "RichTextLabel", palette_map["text"])
	theme.set_color("font_color", "CheckBox", palette_map["text"])
	theme.set_color("font_color", "CheckButton", palette_map["text"])
	theme.set_color("font_color", "PopupMenu", palette_map["text"])
	theme.set_color("font_disabled_color", "PopupMenu", palette_map["text_muted"])
	theme.set_color("font_hover_color", "PopupMenu", palette_map["text"])
	theme.set_color("font_separator_color", "PopupMenu", palette_map["text_muted"])
	theme.set_color("font_accelerator_color", "PopupMenu", palette_map["text_muted"])
	theme.set_stylebox("separator", "HSeparator", separator_style)

	theme.set_color("icon_normal_color", "Button", palette_map["text_muted"])
	theme.set_color("icon_hover_color", "Button", palette_map["accent"])
	theme.set_color("icon_focus_color", "Button", palette_map["accent"])
	theme.set_color("icon_pressed_color", "Button", palette_map["text"])
	theme.set_color("icon_hover_pressed_color", "Button", palette_map["text"])
	var disabled_icon: Color = palette_map["text_muted"]
	disabled_icon.a = 0.45
	theme.set_color("icon_disabled_color", "Button", disabled_icon)

	theme.set_stylebox("normal", "CheckBox", button_flat)
	theme.set_stylebox("hover", "CheckBox", button_flat)
	theme.set_stylebox("pressed", "CheckBox", button_flat)
	theme.set_stylebox("focus", "CheckBox", button_flat)
	theme.set_stylebox("normal", "CheckButton", button_flat)
	theme.set_stylebox("hover", "CheckButton", button_flat)
	theme.set_stylebox("pressed", "CheckButton", button_flat)
	theme.set_stylebox("focus", "CheckButton", button_flat)
	_set_slider_styles(theme, palette_map)
	_set_scrollbar_styles(theme, palette_map)

	return theme

func _set_slider_styles(theme: Theme, palette_map: Dictionary) -> void:
	var grabber_size := 18
	var grabber := _make_pill_icon(grabber_size, grabber_size, palette_map["accent"])
	var grabber_highlight := _make_pill_icon(grabber_size, grabber_size, palette_map["field_focus"])
	var grabber_disabled := _make_pill_icon(grabber_size, grabber_size, palette_map["button_border"])

	var track := StyleBoxFlat.new()
	track.bg_color = palette_map["button_hover"]
	track.border_color = palette_map["button_border"]
	track.set_border_width_all(1)
	track.set_corner_radius_all(4)
	track.content_margin_top = 4
	track.content_margin_bottom = 4
	for slider_type in ["HSlider", "VSlider"]:
		theme.set_icon("grabber", slider_type, grabber)
		theme.set_icon("grabber_highlight", slider_type, grabber_highlight)
		theme.set_icon("grabber_disabled", slider_type, grabber_disabled)
		theme.set_stylebox("slider", slider_type, track)
		theme.set_stylebox("grabber_area", slider_type, track)
		theme.set_stylebox("grabber_area_highlight", slider_type, track)
		theme.set_constant("grabber_size", slider_type, grabber_size)

func _set_scrollbar_styles(theme: Theme, palette_map: Dictionary) -> void:
	var track := StyleBoxFlat.new()
	track.bg_color = palette_map["field_bg"]
	track.border_color = palette_map["border"]
	track.set_border_width_all(1)
	track.set_corner_radius_all(5)
	track.set_content_margin_all(2)

	var grabber := StyleBoxFlat.new()
	grabber.bg_color = palette_map["button_hover"]
	grabber.border_color = palette_map["scrollbar_outline"]
	grabber.set_border_width_all(2)
	grabber.set_corner_radius_all(5)

	var grabber_hover: StyleBoxFlat = grabber.duplicate()
	grabber_hover.bg_color = palette_map["button_pressed"]

	var grabber_pressed: StyleBoxFlat = grabber.duplicate()
	grabber_pressed.bg_color = palette_map["accent"]

	for scrollbar_type in ["VScrollBar", "HScrollBar"]:
		theme.set_stylebox("scroll", scrollbar_type, track)
		theme.set_stylebox("scroll_focus", scrollbar_type, track)
		theme.set_stylebox("grabber", scrollbar_type, grabber)
		theme.set_stylebox("grabber_highlight", scrollbar_type, grabber_hover)
		theme.set_stylebox("grabber_pressed", scrollbar_type, grabber_pressed)
		theme.set_constant("scroll_size", scrollbar_type, 12)
		theme.set_constant("min_grab_thickness", scrollbar_type, 36)

func _make_arrow_icon(width: int, height: int, color: Color) -> Texture2D:
	var image := Image.create(width, height, false, Image.FORMAT_RGBA8)
	image.fill(Color(0, 0, 0, 0))
	var midpoint := floori(height / 2.0)
	for y in range(height):
		var inset := mini(y, height - 1 - y)
		var start_x := inset
		var end_x := width - inset - 1
		for x in range(start_x, end_x + 1):
			if y >= midpoint:
				image.set_pixel(x, y, color)
	return ImageTexture.create_from_image(image)

func _make_pill_icon(width: int, height: int, color: Color) -> ImageTexture:
	var image := Image.create(width, height, false, Image.FORMAT_RGBA8)
	for y in range(height):
		for x in range(width):
			var rx := (x + 0.5 - 0.5 * width) / (0.5 * width)
			var ry := (y + 0.5 - 0.5 * height) / (0.5 * height)
			image.set_pixel(x, y, color if rx * rx + ry * ry <= 1.0 else Color(0, 0, 0, 0))
	return ImageTexture.create_from_image(image)

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

func _button_style(bg: Color, border: Color, filled: bool) -> StyleBoxFlat:
	var style := StyleBoxFlat.new()
	style.bg_color = bg
	style.border_color = border
	style.set_border_width_all(1)
	style.set_corner_radius_all(20)
	style.content_margin_left = 16
	style.content_margin_right = 16
	style.content_margin_top = 4
	style.content_margin_bottom = 4
	if filled:
		style.shadow_size = 0
	return style
