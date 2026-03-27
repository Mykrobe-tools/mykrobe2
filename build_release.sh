#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GUI_DIR="${ROOT_DIR}/gui"
BUILD_BIN_SCRIPT="${ROOT_DIR}/build_mykrobe2_bins.sh"
DEFAULT_GODOT_BIN="/Users/martin/Applications/Godot.app/Contents/MacOS/Godot"

TARGET=""
PRESET=""
OUTFILE=""
GODOT_BIN="${DEFAULT_GODOT_BIN}"
EXPORT_MODE="release"
SKIP_BACKEND=0

usage() {
	cat <<'EOF'
Usage:
  ./build_release.sh --target <os>/<arch> --preset "<Godot Preset Name>" --out <artifact_path> [options]

Required:
  --target   Target used to build bundled mykrobe2, e.g. darwin/arm64, linux/amd64, windows/arm64
  --preset   Godot export preset name from gui/export_presets.cfg
  --out      Output artifact path for Godot export

Options:
  --godot-bin <path>   Godot executable path
  --mode <release|debug>
  --skip-backend
  -h, --help
EOF
}

while [[ $# -gt 0 ]]; do
	case "$1" in
		--target)
			[[ $# -ge 2 ]] || { echo "Missing value for --target" >&2; exit 1; }
			TARGET="$2"
			shift 2
			;;
		--preset)
			[[ $# -ge 2 ]] || { echo "Missing value for --preset" >&2; exit 1; }
			PRESET="$2"
			shift 2
			;;
		--out)
			[[ $# -ge 2 ]] || { echo "Missing value for --out" >&2; exit 1; }
			OUTFILE="$2"
			shift 2
			;;
		--godot-bin)
			[[ $# -ge 2 ]] || { echo "Missing value for --godot-bin" >&2; exit 1; }
			GODOT_BIN="$2"
			shift 2
			;;
		--mode)
			[[ $# -ge 2 ]] || { echo "Missing value for --mode" >&2; exit 1; }
			EXPORT_MODE="$2"
			shift 2
			;;
		--skip-backend)
			SKIP_BACKEND=1
			shift
			;;
		-h|--help)
			usage
			exit 0
			;;
		*)
			echo "Unknown argument: $1" >&2
			usage
			exit 1
			;;
	esac
done

[[ -n "${TARGET}" ]] || { echo "--target is required" >&2; exit 1; }
[[ -n "${PRESET}" ]] || { echo "--preset is required" >&2; exit 1; }
[[ -n "${OUTFILE}" ]] || { echo "--out is required" >&2; exit 1; }
[[ "${EXPORT_MODE}" == "release" || "${EXPORT_MODE}" == "debug" ]] || {
	echo "--mode must be release or debug" >&2
	exit 1
}

if [[ ! -x "${BUILD_BIN_SCRIPT}" ]]; then
	echo "Missing or non-executable: ${BUILD_BIN_SCRIPT}" >&2
	exit 1
fi
if [[ ! -x "${GODOT_BIN}" ]]; then
	echo "Godot executable not found or not executable: ${GODOT_BIN}" >&2
	exit 1
fi
if [[ ! -f "${GUI_DIR}/project.godot" ]]; then
	echo "Missing project.godot in ${GUI_DIR}" >&2
	exit 1
fi
if [[ ! -f "${GUI_DIR}/export_presets.cfg" ]]; then
	echo "Missing ${GUI_DIR}/export_presets.cfg" >&2
	echo "Create or edit export presets in Godot Editor first, then re-run." >&2
	exit 1
fi
if ! grep -q "name=\"${PRESET}\"" "${GUI_DIR}/export_presets.cfg"; then
	echo "Preset not found in gui/export_presets.cfg: ${PRESET}" >&2
	exit 1
fi

if [[ "${SKIP_BACKEND}" -eq 0 ]]; then
	echo "[1/2] Building bundled mykrobe2 for ${TARGET}"
	"${BUILD_BIN_SCRIPT}" --target "${TARGET}"
else
	echo "[1/2] Skipping bundled mykrobe2 build (--skip-backend)"
fi

echo "[2/2] Exporting mykrobe2 GUI (${EXPORT_MODE}) preset='${PRESET}' -> ${OUTFILE}"
mkdir -p "$(dirname "${OUTFILE}")"
"${GODOT_BIN}" --headless --path "${GUI_DIR}" "--export-${EXPORT_MODE}" "${PRESET}" "${OUTFILE}"
echo "Done: ${OUTFILE}"
