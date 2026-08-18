extends RefCounted
class_name PredictRunManager

var _task_running := false
var _task_pid := -1
var _log_path := ""
var _result_path := ""
var _script_path := ""
var _child_pid_path := ""
var _last_log_text := ""

func is_running() -> bool:
	return _task_running

func log_text() -> String:
	return _last_log_text

func start(binary_path: String, args: PackedStringArray, output_path: String) -> Dictionary:
	if _task_running:
		return {"started": false, "error": "Analysis is already running."}
	var run_id := "%s-%s" % [OS.get_process_id(), Time.get_ticks_usec()]
	var run_prefix := OS.get_user_data_dir().path_join("predict-run-%s" % run_id)
	_log_path = run_prefix + ".log"
	_result_path = run_prefix + ".result"
	_child_pid_path = run_prefix + ".pid"
	_script_path = run_prefix + (".cmd" if OS.get_name() == "Windows" else ".sh")
	_last_log_text = ""
	_task_running = true
	_task_pid = _start_process(binary_path, args, output_path, _log_path, _result_path)
	if _task_pid == -1:
		_task_running = false
		_cleanup_run_files()
		return {"started": false, "error": "Could not start background analysis."}
	return {"started": true}

func cancel() -> void:
	if not _task_running:
		return
	_kill_process_tree()
	_task_running = false
	_task_pid = -1
	_cleanup_run_files()

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

func _start_process(binary_path: String, args: PackedStringArray, output_path: String, log_path: String, result_path: String) -> int:
	if OS.get_name() == "Windows":
		_write_windows_script(_script_path, binary_path, args, output_path, log_path, result_path)
		return OS.create_process("cmd.exe", PackedStringArray(["/C", _script_path]), false)
	_write_posix_script(_script_path, binary_path, args, output_path, log_path, result_path)
	return OS.create_process("/bin/bash", PackedStringArray([_script_path]), false)

func _write_posix_script(script_path: String, binary_path: String, args: PackedStringArray, output_path: String, log_path: String, result_path: String) -> void:
	var lines: PackedStringArray = [
		"#!/usr/bin/env bash",
		"set -u",
		"%s %s >> %s 2>&1 &" % [_shell_quote(binary_path), _join_shell_args(args), _shell_quote(log_path)],
		"child_pid=$!",
		"echo \"$child_pid\" > %s" % _shell_quote(_child_pid_path),
		"wait \"$child_pid\" 2>/dev/null",
		"status=$?",
		"rm -f %s" % _shell_quote(_child_pid_path),
		"if [ \"$status\" -eq 0 ]; then",
		"  echo %s > %s" % [_shell_quote("success=1"), _shell_quote(result_path)],
		"  echo %s >> %s" % [_shell_quote("status=Analysis complete."), _shell_quote(result_path)],
		"  echo %s >> %s" % [_shell_quote("error="), _shell_quote(result_path)],
		"  echo %s >> %s" % [_shell_quote("output_path=%s" % output_path), _shell_quote(result_path)],
		"else",
		"  echo %s > %s" % [_shell_quote("success=0"), _shell_quote(result_path)],
		"  echo %s >> %s" % [_shell_quote("status=Analysis failed."), _shell_quote(result_path)],
		"  echo %s >> %s" % [_shell_quote("error=Analysis failed."), _shell_quote(result_path)],
		"  echo %s >> %s" % [_shell_quote("output_path=%s" % output_path), _shell_quote(result_path)],
		"fi",
	]
	_write_text_file(script_path, "\n".join(lines) + "\n")

func _write_windows_script(script_path: String, binary_path: String, args: PackedStringArray, output_path: String, log_path: String, result_path: String) -> void:
	var lines: PackedStringArray = [
		"@echo off",
		"%s %s >> %s 2>&1" % [_windows_quote(binary_path), _join_windows_args(args), _windows_quote(log_path)],
		"if errorlevel 1 (",
		"  > %s echo success=0" % _windows_quote(result_path),
		"  >> %s echo status=Analysis failed." % _windows_quote(result_path),
		"  >> %s echo error=Analysis failed." % _windows_quote(result_path),
		"  >> %s echo output_path=%s" % [_windows_quote(result_path), output_path],
		") else (",
		"  > %s echo success=1" % _windows_quote(result_path),
		"  >> %s echo status=Analysis complete." % _windows_quote(result_path),
		"  >> %s echo error=" % _windows_quote(result_path),
		"  >> %s echo output_path=%s" % [_windows_quote(result_path), output_path],
		")",
	]
	_write_text_file(script_path, "\r\n".join(lines) + "\r\n")

func _write_text_file(path: String, text: String) -> void:
	var file := FileAccess.open(path, FileAccess.WRITE)
	if file == null:
		return
	file.store_string(text)
	file.close()

func _read_result(path: String) -> Dictionary:
	var out := {"success": false, "status": "Analysis failed.", "error": "Analysis failed.", "output_path": ""}
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
			"output_path":
				out["output_path"] = parts[1]
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

func _kill_process_tree() -> void:
	if _task_pid <= 0:
		return
	if OS.get_name() == "Windows":
		var exit_code := OS.execute("taskkill", PackedStringArray(["/PID", str(_task_pid), "/T", "/F"]), [], true)
		if exit_code != 0:
			OS.kill(_task_pid)
		return
	var child_pid := _read_child_pid()
	if child_pid > 0:
		OS.execute("/bin/kill", PackedStringArray(["-KILL", str(child_pid)]), [], true)
	OS.kill(_task_pid)

func _read_child_pid() -> int:
	if _child_pid_path == "" or not FileAccess.file_exists(_child_pid_path):
		return -1
	var pid_text := FileAccess.get_file_as_string(_child_pid_path).strip_edges()
	if not pid_text.is_valid_int():
		return -1
	return pid_text.to_int()

func _cleanup_run_files() -> void:
	for path in [_log_path, _result_path, _script_path, _child_pid_path]:
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
