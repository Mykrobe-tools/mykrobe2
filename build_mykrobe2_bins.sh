#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CMD_DIR="${ROOT_DIR}/cmd/mykrobe2"
GUI_BIN_DIR="${ROOT_DIR}/gui/bin"
TARGETS_DIR="${GUI_BIN_DIR}/targets"

TARGET=""
BUILD_ALL=0

usage() {
	cat <<'EOF'
Usage:
  ./build_mykrobe2_bins.sh
  ./build_mykrobe2_bins.sh --target <os>/<arch>
  ./build_mykrobe2_bins.sh --all

Examples:
  ./build_mykrobe2_bins.sh
  ./build_mykrobe2_bins.sh --target darwin/arm64
  ./build_mykrobe2_bins.sh --target linux/amd64
  ./build_mykrobe2_bins.sh --all

Notes:
  - Default (no args) builds for current host and writes:
      gui/bin/mykrobe2
      gui/bin/mykrobe2.exe (on Windows target)
  - --target builds one target and writes canonical bundle filename above.
  - --all writes matrix artifacts to:
      gui/bin/targets/mykrobe2_<os>_<arch>[.exe]
EOF
}

while [[ $# -gt 0 ]]; do
	case "$1" in
		--target)
			[[ $# -ge 2 ]] || { echo "Missing value for --target" >&2; exit 1; }
			TARGET="$2"
			shift 2
			;;
		--all)
			BUILD_ALL=1
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

if [[ ! -d "${CMD_DIR}" ]]; then
	echo "Missing mykrobe2 command directory: ${CMD_DIR}" >&2
	exit 1
fi

mkdir -p "${GUI_BIN_DIR}"

build_one() {
	local goos="$1"
	local goarch="$2"
	local out="$3"
	echo "Building mykrobe2 for ${goos}/${goarch} -> ${out}"
	(
		cd "${CMD_DIR}"
		GOOS="${goos}" GOARCH="${goarch}" CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o "${out}" .
	)
}

host_goos="$(go env GOOS)"
host_goarch="$(go env GOARCH)"

if [[ "${BUILD_ALL}" -eq 1 ]]; then
	mkdir -p "${TARGETS_DIR}"
	target_matrix=(
		"darwin/amd64"
		"darwin/arm64"
		"linux/amd64"
		"linux/arm64"
		"windows/amd64"
		"windows/arm64"
	)
	for t in "${target_matrix[@]}"; do
		goos="${t%%/*}"
		goarch="${t##*/}"
		ext=""
		if [[ "${goos}" == "windows" ]]; then
			ext=".exe"
		fi
		out="${TARGETS_DIR}/mykrobe2_${goos}_${goarch}${ext}"
		build_one "${goos}" "${goarch}" "${out}"
	done
	echo "Done. Artifacts in: ${TARGETS_DIR}"
	exit 0
fi

if [[ -n "${TARGET}" ]]; then
	host_goos="${TARGET%%/*}"
	host_goarch="${TARGET##*/}"
	if [[ "${host_goos}" == "${host_goarch}" ]]; then
		echo "Invalid --target '${TARGET}'. Expected os/arch." >&2
		exit 1
	fi
fi

out_name="mykrobe2"
if [[ "${host_goos}" == "windows" ]]; then
	out_name="mykrobe2.exe"
fi
build_one "${host_goos}" "${host_goarch}" "${GUI_BIN_DIR}/${out_name}"
echo "Done. Bundled binary: ${GUI_BIN_DIR}/${out_name}"
