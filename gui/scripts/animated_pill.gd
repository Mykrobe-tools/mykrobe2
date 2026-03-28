extends PanelContainer
class_name AnimatedPill

const PILL_COLORS := [
	Color(0.180392, 0.447059, 0.611765, 0.8),
	Color(0.905882, 0.580392, 0.196078, 0.82),
]

@export var pill_size: Vector2 = Vector2(280, 92)
@export var speed_x: float = 50.0
@export var speed_y: float = 6.0
@export var angle_degrees: float = 28.0
@export var min_angle_multiplier: float = 0.45
@export var max_angle_multiplier: float = 2.35
@export var min_spin_speed_degrees: float = 3.5
@export var max_spin_speed_degrees: float = 9.0
@export var spawn_y_ratio: float = 0.5
@export var start_offset_x: float = 140.0
@export var x_jitter: float = 180.0
@export var min_scale_factor: float = 0.88
@export var max_scale_factor: float = 1.28
@export var mean_respawn_delay: float = 0.9
@export var max_respawn_delay: float = 2.2

var _rng := RandomNumberGenerator.new()
var _respawn_delay_remaining := 0.0
var _spin_speed_degrees := 0.0

func _ready() -> void:
	_rng.randomize()
	var scale_factor := _rng.randf_range(min_scale_factor, max_scale_factor)
	custom_minimum_size = pill_size * scale_factor
	size = custom_minimum_size
	pivot_offset = size / 2.0
	rotation_degrees = angle_degrees * _rng.randf_range(min_angle_multiplier, max_angle_multiplier)
	_spin_speed_degrees = _rng.randf_range(min_spin_speed_degrees, max_spin_speed_degrees)
	mouse_filter = Control.MOUSE_FILTER_IGNORE
	var style := StyleBoxFlat.new()
	style.bg_color = PILL_COLORS[_rng.randi_range(0, PILL_COLORS.size() - 1)]
	var radius := int(round(min(size.x, size.y) / 2.0))
	style.corner_radius_top_left = radius
	style.corner_radius_top_right = radius
	style.corner_radius_bottom_left = radius
	style.corner_radius_bottom_right = radius
	add_theme_stylebox_override("panel", style)

func reset_to_spawn(view_size: Vector2, randomize_y: bool = false) -> void:
	var y := clampf(view_size.y * spawn_y_ratio, -pill_size.y, view_size.y + pill_size.y)
	if randomize_y:
		y += _rng.randf_range(-120.0, 120.0)
	var x := -size.x - absf(start_offset_x) - _rng.randf_range(0.0, x_jitter)
	position = Vector2(x, y)
	_respawn_delay_remaining = 0.0

func queue_spawn(view_size: Vector2, delay: float, randomize_y: bool = true) -> void:
	reset_to_spawn(view_size, randomize_y)
	_respawn_delay_remaining = maxf(0.0, delay)

func place_initial(view_size: Vector2) -> void:
	var y := clampf(view_size.y * spawn_y_ratio, -pill_size.y, view_size.y + pill_size.y)
	y += _rng.randf_range(-80.0, 80.0)
	var x := _rng.randf_range(-size.x * 0.15, view_size.x + size.x * 0.15)
	position = Vector2(x, y)
	_respawn_delay_remaining = 0.0

func sample_respawn_delay() -> float:
	var u := clampf(_rng.randf(), 0.0001, 0.9999)
	return minf(-log(1.0 - u) * mean_respawn_delay, max_respawn_delay)

func advance(delta: float, view_size: Vector2) -> void:
	if _respawn_delay_remaining > 0.0:
		_respawn_delay_remaining = maxf(0.0, _respawn_delay_remaining - delta)
		return
	rotation_degrees += _spin_speed_degrees * delta
	position.x += speed_x * delta
	position.y += speed_y * delta
	if position.x > view_size.x + pill_size.x + 120.0:
		queue_spawn(view_size, sample_respawn_delay(), true)
