extends RefCounted
class_name LocalMykrobe2Manager

var _bin_subdir := "bin"
var _local_binary_path := ""
var _last_error := ""

func configure(bin_subdir: String = "bin") -> void:
	_bin_subdir = bin_subdir

func last_error() -> String:
	return _last_error

func ensure_local_binary_installed() -> bool:
	var bin_name := _binary_name()
	var user_bin_dir_abs := OS.get_user_data_dir().path_join(_bin_subdir)
	var mk_err := DirAccess.make_dir_recursive_absolute(user_bin_dir_abs)
	if mk_err != OK and not DirAccess.dir_exists_absolute(user_bin_dir_abs):
		_last_error = "Failed to create local bin dir: %s" % user_bin_dir_abs
		return false

	var target_abs := user_bin_dir_abs.path_join(bin_name)
	_local_binary_path = target_abs
	var source := _find_binary_source(bin_name)
	if source.is_empty():
		if FileAccess.file_exists(target_abs):
			_last_error = ""
			return true
		_last_error = "No bundled mykrobe2 found at res://bin/%s" % bin_name
		return false

	var src_hash := FileAccess.get_sha256(source)
	var dst_hash := ""
	if FileAccess.file_exists(target_abs):
		dst_hash = FileAccess.get_sha256(target_abs)
	if FileAccess.file_exists(target_abs) and not src_hash.is_empty() and src_hash == dst_hash:
		_last_error = ""
		return true
	if not _copy_file_any_to_abs(source, target_abs):
		_last_error = "Failed to install mykrobe2 into %s" % target_abs
		return false
	if not OS.has_feature("windows"):
		OS.execute("chmod", ["+x", target_abs], [], true)
	_last_error = ""
	return true

func installed_binary_path() -> String:
	if not _local_binary_path.is_empty() and FileAccess.file_exists(_local_binary_path):
		return _local_binary_path
	var target_abs := OS.get_user_data_dir().path_join(_bin_subdir).path_join(_binary_name())
	if FileAccess.file_exists(target_abs):
		_local_binary_path = target_abs
		return target_abs
	return ""

func _find_binary_source(bin_name: String) -> String:
	var packaged := "res://bin/%s" % bin_name
	if FileAccess.file_exists(packaged):
		return packaged
	var dev_abs := ProjectSettings.globalize_path("res://../build/%s" % bin_name)
	if FileAccess.file_exists(dev_abs):
		return dev_abs
	if bin_name == "mykrobe2":
		var alt_dev := ProjectSettings.globalize_path("res://../build/mykrobe2")
		if FileAccess.file_exists(alt_dev):
			return alt_dev
	return ""

func _copy_file_any_to_abs(source: String, target_abs: String) -> bool:
	var src := FileAccess.open(source, FileAccess.READ)
	if src == null:
		return false
	var dst := FileAccess.open(target_abs, FileAccess.WRITE)
	if dst == null:
		src.close()
		return false
	dst.store_buffer(src.get_buffer(src.get_length()))
	dst.close()
	src.close()
	return true

func _binary_name() -> String:
	if OS.has_feature("windows"):
		return "mykrobe2.exe"
	return "mykrobe2"
