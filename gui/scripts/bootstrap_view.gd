extends Control
class_name BootstrapView

@onready var circle: PanelContainer = $BootstrapCenter/BootstrapCard/BootstrapCircle
@onready var logo_icon: TextureRect = $BootstrapCenter/BootstrapCard/BootstrapMargin/BootstrapVBox/BootstrapLogo/BootstrapLogoIcon
@onready var logo_text: Label = $BootstrapCenter/BootstrapCard/BootstrapMargin/BootstrapVBox/BootstrapLogo/BootstrapLogoText
@onready var title_label: Label = $BootstrapCenter/BootstrapCard/BootstrapMargin/BootstrapVBox/BootstrapTitle
@onready var status_label: Label = $BootstrapCenter/BootstrapCard/BootstrapMargin/BootstrapVBox/BootstrapStatus
@onready var progress_bar: ProgressBar = $BootstrapCenter/BootstrapCard/BootstrapMargin/BootstrapVBox/ProgressBar
@onready var phase_label: Label = $BootstrapCenter/BootstrapCard/BootstrapMargin/BootstrapVBox/PhaseLabel
@onready var elapsed_label: Label = $BootstrapCenter/BootstrapCard/BootstrapMargin/BootstrapVBox/ElapsedLabel
@onready var details_button: Button = $BootstrapCenter/BootstrapCard/BootstrapMargin/BootstrapVBox/DetailsButton
@onready var log_text: RichTextLabel = $BootstrapCenter/BootstrapCard/BootstrapMargin/BootstrapVBox/BootstrapLog

var _activity_started_msec := 0

func set_logo_texture(texture: Texture2D) -> void:
	logo_icon.texture = texture

func set_status(message: String) -> void:
	status_label.text = message

func set_log(message: String) -> void:
	log_text.text = message

func begin_activity(message: String) -> void:
	set_status(message)
	set_log("")
	phase_label.text = "Starting…"
	progress_bar.indeterminate = true
	progress_bar.show_percentage = false
	progress_bar.value = 0
	_activity_started_msec = Time.get_ticks_msec()
	log_text.visible = false
	details_button.text = "Show details"
	_update_elapsed_time()

func update_activity(progress: Dictionary, fallback_phase: String, log_message: String) -> void:
	var clean_phase := str(progress.get("message", fallback_phase)).strip_edges()
	if clean_phase != "":
		phase_label.text = clean_phase
	var determinate := bool(progress.get("determinate", false))
	progress_bar.indeterminate = not determinate
	progress_bar.show_percentage = determinate
	progress_bar.value = clampf(float(progress.get("fraction", 0.0)) * 100.0, 0.0, 100.0)
	set_log(log_message)

func _process(_delta: float) -> void:
	if visible and _activity_started_msec > 0:
		_update_elapsed_time()

func _update_elapsed_time() -> void:
	var elapsed_seconds := maxi(0, int((Time.get_ticks_msec() - _activity_started_msec) / 1000))
	elapsed_label.text = _format_elapsed_time(elapsed_seconds)

static func _format_elapsed_time(elapsed_seconds: int) -> String:
	if elapsed_seconds < 60:
		return "Working… %ds" % elapsed_seconds
	return "Working… %dm %02ds" % [elapsed_seconds / 60, elapsed_seconds % 60]

func _on_details_button_pressed() -> void:
	log_text.visible = not log_text.visible
	details_button.text = "Hide details" if log_text.visible else "Show details"

func apply_palette(palette: Dictionary) -> void:
	var circle_style := StyleBoxFlat.new()
	circle_style.bg_color = palette.get("circle_bg", Color(1, 1, 1, 0.92))
	circle_style.set_corner_radius_all(400)
	circle.add_theme_stylebox_override("panel", circle_style)
	logo_text.add_theme_color_override("font_color", palette.get("accent", Color("3987b5")))
	title_label.add_theme_color_override("font_color", palette.get("text", Color("6d6a65")))
	status_label.add_theme_color_override("font_color", palette.get("text_muted", Color("8b8478")))
	phase_label.add_theme_color_override("font_color", palette.get("text", Color("6d6a65")))
	elapsed_label.add_theme_color_override("font_color", palette.get("text_muted", Color("8b8478")))
	log_text.add_theme_color_override("default_color", palette.get("text", Color("6d6a65")))
	var progress_background := StyleBoxFlat.new()
	progress_background.bg_color = palette.get("button_hover", Color("f4fbff"))
	progress_background.border_color = palette.get("button_border", Color("b9d6ea"))
	progress_background.set_border_width_all(1)
	progress_background.set_corner_radius_all(10)
	var progress_fill := StyleBoxFlat.new()
	progress_fill.bg_color = palette.get("accent", Color("3987b5"))
	progress_fill.set_corner_radius_all(10)
	progress_bar.add_theme_stylebox_override("background", progress_background)
	progress_bar.add_theme_stylebox_override("fill", progress_fill)
	progress_bar.add_theme_color_override("font_color", palette.get("text", Color("6d6a65")))
	var transparent_log := StyleBoxFlat.new()
	transparent_log.bg_color = Color(0, 0, 0, 0)
	transparent_log.set_border_width_all(0)
	log_text.add_theme_stylebox_override("normal", transparent_log)
