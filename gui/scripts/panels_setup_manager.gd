extends RefCounted
class_name PanelsSetupManager

var _task_running := false
var _task_pid := -1
var _log_path := ""
var _result_path := ""
var _script_path := ""
var _last_log_text := ""

func is_running() -> bool:
	return _task_running

func log_text() -> String:
	return _last_log_text

func start(binary_path: String, commands: Array, success_status: String) -> Dictionary:
	if _task_running:
		return {"started": false, "error": "Panel setup is already running."}
	var run_id := "%s-%s" % [OS.get_process_id(), Time.get_ticks_usec()]
	var run_prefix := OS.get_user_data_dir().path_join("panels-setup-%s" % run_id)
	_log_path = run_prefix + ".log"
	_result_path = run_prefix + ".result"
	_script_path = run_prefix + (".cmd" if OS.get_name() == "Windows" else ".sh")
	_last_log_text = ""
	_task_running = true
	_task_pid = _start_process(binary_path, commands, success_status, _log_path, _result_path)
	if _task_pid == -1:
		_task_running = false
		_cleanup_run_files()
		return {"started": false, "error": "Could not start background panel setup."}
	return {"started": true}

func poll() -> Dictionary:
	if not _task_running:
		return {"running": false}
	_refresh_log_from_disk()
	if not FileAccess.file_exists(_result_path):
		return {"running": true, "log": _last_log_text}
	_task_running = false
	_task_pid = -1
	_refresh_log_from_disk()
	var result := _read_result(_result_path)
	result["running"] = false
	result["finished"] = true
	result["log"] = _last_log_text
	_cleanup_run_files()
	return result

func _start_process(binary_path: String, commands: Array, success_status: String, log_path: String, result_path: String) -> int:
	if OS.get_name() == "Windows":
		_write_windows_setup_script(_script_path, binary_path, commands, success_status, log_path, result_path)
		return OS.create_process("cmd.exe", PackedStringArray(["/C", _script_path]), false)
	_write_posix_setup_script(_script_path, binary_path, commands, success_status, log_path, result_path)
	return OS.create_process("/bin/bash", PackedStringArray([_script_path]), false)

func _write_posix_setup_script(script_path: String, binary_path: String, commands: Array, success_status: String, log_path: String, result_path: String) -> void:
	var lines: PackedStringArray = [
		"#!/usr/bin/env bash",
		"set -u",
		"echo \"Starting setup.\" >> %s" % _shell_quote(log_path),
	]
	for command in commands:
		var label := str(command.get("label", "Running command"))
		var args: PackedStringArray = command.get("args", PackedStringArray())
		lines.append("echo %s >> %s" % [_shell_quote(label + "..."), _shell_quote(log_path)])
		lines.append("if ! %s %s >> %s 2>&1; then" % [_shell_quote(binary_path), _join_shell_args(args), _shell_quote(log_path)])
		lines.append("  echo %s > %s" % [_shell_quote("success=0"), _shell_quote(result_path)])
		lines.append("  echo %s >> %s" % [_shell_quote("status=Panel setup failed."), _shell_quote(result_path)])
		lines.append("  echo %s >> %s" % [_shell_quote("error=%s failed." % label), _shell_quote(result_path)])
		lines.append("  exit 0")
		lines.append("fi")
		lines.append("echo %s >> %s" % [_shell_quote(label + " complete."), _shell_quote(log_path)])
	lines.append("echo %s > %s" % [_shell_quote("success=1"), _shell_quote(result_path)])
	lines.append("echo %s >> %s" % [_shell_quote("status=%s" % success_status), _shell_quote(result_path)])
	lines.append("echo %s >> %s" % [_shell_quote("error="), _shell_quote(result_path)])
	_write_text_file(script_path, "\n".join(lines) + "\n")

func _write_windows_setup_script(script_path: String, binary_path: String, commands: Array, success_status: String, log_path: String, result_path: String) -> void:
	var lines: PackedStringArray = [
		"@echo off",
		"echo Starting setup.>> %s" % _windows_quote(log_path),
	]
	for command in commands:
		var label := str(command.get("label", "Running command"))
		var args: PackedStringArray = command.get("args", PackedStringArray())
		lines.append("echo %s>> %s" % [label + "...", _windows_quote(log_path)])
		lines.append("%s %s >> %s 2>&1" % [_windows_quote(binary_path), _join_windows_args(args), _windows_quote(log_path)])
		lines.append("if errorlevel 1 (")
		lines.append("  > %s echo success=0" % _windows_quote(result_path))
		lines.append("  >> %s echo status=Panel setup failed." % _windows_quote(result_path))
		lines.append("  >> %s echo error=%s failed." % [_windows_quote(result_path), label])
		lines.append("  exit /b 0")
		lines.append(")")
		lines.append("echo %s>> %s" % [label + " complete.", _windows_quote(log_path)])
	lines.append("> %s echo success=1" % _windows_quote(result_path))
	lines.append(">> %s echo status=%s" % [_windows_quote(result_path), success_status])
	lines.append(">> %s echo error=" % _windows_quote(result_path))
	_write_text_file(script_path, "\r\n".join(lines) + "\r\n")

func _write_text_file(path: String, text: String) -> void:
	var file := FileAccess.open(path, FileAccess.WRITE)
	if file == null:
		return
	file.store_string(text)
	file.close()

func _read_result(path: String) -> Dictionary:
	var out := {"success": false, "status": "Panel setup failed.", "error": "Panel setup failed."}
	var file := FileAccess.open(path, FileAccess.READ)
	if file == null:
		return out
	while not file.eof_reached():
		var line := file.get_line()
		if not line.contains("="):
			continue
		var parts := line.split("=", false, 1)
		if parts.size() != 2:
			continue
		match parts[0]:
			"success":
				out["success"] = parts[1] == "1"
			"status":
				out["status"] = parts[1]
			"error":
				out["error"] = parts[1]
	file.close()
	return out

func _refresh_log_from_disk() -> void:
	if _log_path == "":
		return
	if not FileAccess.file_exists(_log_path):
		return
	var file := FileAccess.open(_log_path, FileAccess.READ)
	if file == null:
		return
	_last_log_text = file.get_as_text()
	file.close()

func _cleanup_run_files() -> void:
	for path in [_log_path, _result_path, _script_path]:
		if path != "" and FileAccess.file_exists(path):
			DirAccess.remove_absolute(path)

func _join_shell_args(args: PackedStringArray) -> String:
	var parts: PackedStringArray = []
	for arg in args:
		parts.append(_shell_quote(arg))
	return " ".join(parts)

func _shell_quote(value: String) -> String:
	return "'" + value.replace("'", "'\"'\"'") + "'"

func _join_windows_args(args: PackedStringArray) -> String:
	var parts: PackedStringArray = []
	for arg in args:
		parts.append(_windows_quote(arg))
	return " ".join(parts)

func _windows_quote(value: String) -> String:
	return "\"" + value.replace("\"", "\"\"") + "\""
