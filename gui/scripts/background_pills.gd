extends Control
class_name BackgroundPills

const AnimatedPillScript = preload("res://scripts/animated_pill.gd")

var _rng := RandomNumberGenerator.new()

func _ready() -> void:
	_rng.randomize()
	mouse_filter = Control.MOUSE_FILTER_IGNORE
	_initialize_pills()

func _process(delta: float) -> void:
	var view_size := size
	for child in get_children():
		if child is AnimatedPillScript:
			child.advance(delta, view_size)

func reset_layout() -> void:
	_initialize_pills()

func _initialize_pills() -> void:
	var view_size := size
	var index := 0
	for child in get_children():
		if child is AnimatedPillScript:
			if index < 5:
				child.place_initial(view_size)
			else:
				child.queue_spawn(view_size, _rng.randf_range(0.15, 1.6), true)
			index += 1
