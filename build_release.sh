#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GUI_DIR="${ROOT_DIR}/gui"
BUILD_SCRIPT="${ROOT_DIR}/build.sh"
GUI_BIN_DIR="${GUI_DIR}/bin"
DEFAULT_GODOT_BIN="${HOME}/Applications/Godot.app/Contents/MacOS/Godot"

TARGET=""
PRESET=""
OUTFILE=""
GODOT_BIN="${DEFAULT_GODOT_BIN}"
EXPORT_MODE="release"
SKIP_BACKEND=0
VERSION="dev"
STAGE_DIR=""
PACKAGE_DIR=""

usage() {
	cat <<'EOF'
Usage:
  ./build_release.sh --target <os>/<arch> --preset "<Godot Preset Name>" --out <artifact_path> [options]

Required:
  --target   Target used to build bundled mykrobe2, e.g. darwin/universal, linux/amd64, windows/arm64
  --preset   Godot export preset name from gui/export_presets.cfg
  --out      Output artifact path for Godot export

Options:
  --godot-bin <path>   Godot executable path
  --mode <release|debug>
  --version <version>  Version embedded in the bundled mykrobe2 binary
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
		--version)
			[[ $# -ge 2 ]] || { echo "Missing value for --version" >&2; exit 1; }
			VERSION="$2"
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

if [[ ! -x "${BUILD_SCRIPT}" ]]; then
	echo "Missing or non-executable: ${BUILD_SCRIPT}" >&2
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

if [[ "${OUTFILE}" = /* ]]; then
	outfile_abs="${OUTFILE}"
else
	outfile_abs="${ROOT_DIR}/${OUTFILE}"
fi

STAGE_DIR="$(mktemp -d "${TMPDIR:-/tmp}/mykrobe2-export.XXXXXX")"
trap '[[ -n "${PACKAGE_DIR}" && -d "${PACKAGE_DIR}" ]] && rm -rf "${PACKAGE_DIR}"; [[ -n "${STAGE_DIR}" && -d "${STAGE_DIR}" ]] && rm -rf "${STAGE_DIR}"' EXIT
cp -R "${GUI_DIR}/." "${STAGE_DIR}/"
rm -rf "${STAGE_DIR}/.godot" "${STAGE_DIR}/bin"
mkdir -p "${STAGE_DIR}/bin"

if [[ "${SKIP_BACKEND}" -eq 0 ]]; then
	echo "[1/2] Building bundled mykrobe2 for ${TARGET}"
	goos="${TARGET%%/*}"
	goarch="${TARGET##*/}"
	if [[ "${goos}" == "${goarch}" ]]; then
		echo "Invalid --target '${TARGET}'. Expected os/arch." >&2
		exit 1
	fi
	bin_name="mykrobe2"
	if [[ "${TARGET}" == "darwin/universal" ]]; then
		command -v lipo >/dev/null 2>&1 || { echo "Universal macOS builds require lipo" >&2; exit 1; }
		amd64_dir="${STAGE_DIR}/backend-amd64"
		arm64_dir="${STAGE_DIR}/backend-arm64"
		"${BUILD_SCRIPT}" --os darwin --arch amd64 --version "${VERSION}" --output-dir "${amd64_dir}"
		"${BUILD_SCRIPT}" --os darwin --arch arm64 --version "${VERSION}" --output-dir "${arm64_dir}"
		lipo -create \
			"${amd64_dir}/mykrobe2-${VERSION}-darwin-amd64" \
			"${arm64_dir}/mykrobe2-${VERSION}-darwin-arm64" \
			-output "${STAGE_DIR}/bin/${bin_name}"
		lipo "${STAGE_DIR}/bin/${bin_name}" -verify_arch x86_64 arm64
		rm -rf "${amd64_dir}" "${arm64_dir}"
	else
		backend_dir="${STAGE_DIR}/backend"
		"${BUILD_SCRIPT}" --os "${goos}" --arch "${goarch}" --version "${VERSION}" --output-dir "${backend_dir}"
		artifact_name="mykrobe2-${VERSION}-${goos}-${goarch}"
		if [[ "${goos}" == "windows" ]]; then
			bin_name="${bin_name}.exe"
			artifact_name="${artifact_name}.exe"
		fi
		cp "${backend_dir}/${artifact_name}" "${STAGE_DIR}/bin/${bin_name}"
		rm -rf "${backend_dir}"
	fi
else
	echo "[1/2] Skipping bundled mykrobe2 build (--skip-backend)"
	cp -R "${GUI_BIN_DIR}/." "${STAGE_DIR}/bin/"
fi

echo "[2/2] Exporting mykrobe2 GUI (${EXPORT_MODE}) preset='${PRESET}' -> ${outfile_abs}"
mkdir -p "$(dirname "${outfile_abs}")"

if [[ "${TARGET}" == windows/* && "${outfile_abs}" == *.zip ]]; then
	PACKAGE_DIR="$(mktemp -d "${TMPDIR:-/tmp}/mykrobe2-windows.XXXXXX")"
	"${GODOT_BIN}" --headless --path "${STAGE_DIR}" "--export-${EXPORT_MODE}" "${PRESET}" "${PACKAGE_DIR}/mykrobe2.exe"
	(cd "${PACKAGE_DIR}" && zip -qr "${outfile_abs}" .)
elif [[ "${TARGET}" == linux/* && "${outfile_abs}" == *.tar.gz ]]; then
	PACKAGE_DIR="$(mktemp -d "${TMPDIR:-/tmp}/mykrobe2-linux.XXXXXX")"
	"${GODOT_BIN}" --headless --path "${STAGE_DIR}" "--export-${EXPORT_MODE}" "${PRESET}" "${PACKAGE_DIR}/mykrobe2"
	(cd "${PACKAGE_DIR}" && tar -czf "${outfile_abs}" .)
elif [[ "${TARGET}" == darwin/* && "${outfile_abs}" == *.dmg ]]; then
	command -v lipo >/dev/null 2>&1 || { echo "macOS architecture exports require lipo" >&2; exit 1; }
	command -v hdiutil >/dev/null 2>&1 || { echo "macOS DMG exports require hdiutil" >&2; exit 1; }
	PACKAGE_DIR="$(mktemp -d "${TMPDIR:-/tmp}/mykrobe2-macos.XXXXXX")"
	app_path="${PACKAGE_DIR}/mykrobe2.app"
	"${GODOT_BIN}" --headless --path "${STAGE_DIR}" "--export-${EXPORT_MODE}" "${PRESET}" "${app_path}"
	engine_binary="$(find "${app_path}/Contents/MacOS" -maxdepth 1 -type f | head -n1)"
	[[ -n "${engine_binary}" ]] || { echo "Could not find exported Godot executable" >&2; exit 1; }
	if [[ "${TARGET##*/}" == "universal" ]]; then
		lipo "${engine_binary}" -verify_arch x86_64 arm64
	else
		lipo -thin "${TARGET##*/}" "${engine_binary}" -output "${engine_binary}.thin"
		mv "${engine_binary}.thin" "${engine_binary}"
		chmod +x "${engine_binary}"
	fi
	hdiutil create -quiet -volname "mykrobe2" -srcfolder "${PACKAGE_DIR}" -ov -format UDZO "${outfile_abs}"
else
	"${GODOT_BIN}" --headless --path "${STAGE_DIR}" "--export-${EXPORT_MODE}" "${PRESET}" "${outfile_abs}"
fi
echo "Done: ${outfile_abs}"
