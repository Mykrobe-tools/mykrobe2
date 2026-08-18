extends Control
class_name ProcessingOverlay

signal cancel_requested

@onready var circle: PanelContainer = $ProcessingCenter/ProcessingCard/ProcessingCircle
@onready var message_label: Label = $ProcessingCenter/ProcessingCard/ProcessingMargin/ProcessingVBox/ProcessingLabel
@onready var progress_bar: ProgressBar = $ProcessingCenter/ProcessingCard/ProcessingMargin/ProcessingVBox/ProgressBar
@onready var cancel_button: Button = $ProcessingCenter/ProcessingCard/ProcessingMargin/ProcessingVBox/CancelButton

func start() -> void:
	visible = true
	cancel_button.disabled = false
	set_progress({"message": "Starting analysis", "fraction": 0.0, "determinate": false})

func stop() -> void:
	visible = false
	cancel_button.disabled = false

func show_loading_results() -> void:
	visible = true
	cancel_button.disabled = true
	set_progress({"message": "Loading results", "fraction": 0.0, "determinate": false})

func set_cancel_enabled(enabled: bool) -> void:
	cancel_button.disabled = not enabled

func set_progress(progress: Dictionary) -> void:
	message_label.text = str(progress.get("message", "Analysing"))
	var determinate := bool(progress.get("determinate", false))
	progress_bar.indeterminate = not determinate
	progress_bar.show_percentage = determinate
	progress_bar.value = clampf(float(progress.get("fraction", 0.0)) * 100.0, 0.0, 100.0)

func apply_palette(palette: Dictionary) -> void:
	var circle_style := StyleBoxFlat.new()
	circle_style.bg_color = palette.get("circle_bg", Color(1, 1, 1, 0.92))
	circle_style.set_corner_radius_all(400)
	circle.add_theme_stylebox_override("panel", circle_style)
	message_label.add_theme_color_override("font_color", palette.get("text", Color("6d6a65")))

func _on_cancel_button_pressed() -> void:
	cancel_requested.emit()
