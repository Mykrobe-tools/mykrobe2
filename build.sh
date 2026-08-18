#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN_NAME="mykrobe2"
CMD_PATH="./cmd/mykrobe2"
DEFAULT_LOCAL_DIR="$ROOT_DIR/build"
DEFAULT_RELEASE_DIR="$ROOT_DIR/dist"
DEFAULT_GOCACHE_DIR="$ROOT_DIR/.gocache"
DEFAULT_GOMODCACHE_DIR="$ROOT_DIR/.gomodcache"

usage() {
	cat <<'EOF'
Usage:
  ./build.sh
  ./build.sh --release --version v0.1.0
  ./build.sh --os linux --arch arm64
  ./build.sh --all

Default behavior:
  Builds mykrobe2 for the current GOOS/GOARCH into ./build/.

Options:
  --release           Build the full release matrix:
                      darwin/linux/windows x amd64/arm64.
                      Outputs are .tar.gz (macOS/Linux) or .zip (Windows),
                      with a SHA-256 checksum file.
  --all               Same target matrix as --release, but version is optional.
  --version VERSION   Version string for release artifact filenames.
  --os GOOS           Build only for the specified GOOS.
  --arch GOARCH       Build only for the specified GOARCH.
  --output-dir DIR    Output directory. Defaults to ./build for local builds,
                      ./dist for release/all-platform builds.
  --help              Show this help text.

Examples:
  ./build.sh
  ./build.sh --os linux --arch amd64
  ./build.sh --release --version v1.2.3
EOF
}

release_mode=0
all_mode=0
version=""
output_dir=""
requested_os=""
requested_arch=""
release_artifacts=()

while [[ $# -gt 0 ]]; do
	case "$1" in
		--release)
			release_mode=1
			shift
			;;
		--all)
			all_mode=1
			shift
			;;
		--version)
			version="${2:-}"
			shift 2
			;;
		--os)
			requested_os="${2:-}"
			shift 2
			;;
		--arch)
			requested_arch="${2:-}"
			shift 2
			;;
		--output-dir)
			output_dir="${2:-}"
			shift 2
			;;
		--help|-h)
			usage
			exit 0
			;;
		*)
			echo "Unknown argument: $1" >&2
			usage >&2
			exit 1
			;;
	esac
done

if [[ $release_mode -eq 1 && -z "$version" ]]; then
	echo "--release requires --version" >&2
	exit 1
fi

if [[ $release_mode -eq 1 && ( -n "$requested_os" || -n "$requested_arch" ) ]]; then
	echo "--release builds the full matrix and cannot be combined with --os or --arch" >&2
	exit 1
fi

if [[ $all_mode -eq 1 && ( -n "$requested_os" || -n "$requested_arch" ) ]]; then
	echo "--all builds the full matrix and cannot be combined with --os or --arch" >&2
	exit 1
fi

if [[ -n "$requested_os" && -z "$requested_arch" ]]; then
	requested_arch="$(go env GOARCH)"
fi

if [[ -n "$requested_arch" && -z "$requested_os" ]]; then
	requested_os="$(go env GOOS)"
fi

if [[ -z "$output_dir" ]]; then
	if [[ $release_mode -eq 1 || $all_mode -eq 1 ]]; then
		output_dir="$DEFAULT_RELEASE_DIR"
	else
		output_dir="$DEFAULT_LOCAL_DIR"
	fi
fi

mkdir -p "$output_dir"
mkdir -p "$DEFAULT_GOCACHE_DIR"
mkdir -p "$DEFAULT_GOMODCACHE_DIR"

export GOCACHE="${GOCACHE:-$DEFAULT_GOCACHE_DIR}"
export GOMODCACHE="${GOMODCACHE:-$DEFAULT_GOMODCACHE_DIR}"

build_version="${version:-dev}"

package_release_artifact() {
	local goos="$1"
	local outfile="$2"
	local artifact="$3"
	local package_name=""

	case "$goos" in
		windows)
			package_name="${artifact}.zip"
			(
				cd "$output_dir"
				zip -q -m "$package_name" "$(basename "$outfile")"
			)
			;;
		*)
			package_name="${artifact}.tar.gz"
			tar -C "$output_dir" -czf "$output_dir/$package_name" "$(basename "$outfile")"
			rm -f "$outfile"
			;;
	esac
	release_artifacts+=("$package_name")
}

write_release_checksums() {
	local checksum_file="$output_dir/${BIN_NAME}-${version}-checksums.txt"
	local tmp_file="${checksum_file}.tmp"

	: > "$tmp_file"
	for artifact in "${release_artifacts[@]}"; do
		if command -v sha256sum >/dev/null 2>&1; then
			(cd "$output_dir" && sha256sum "$artifact") >> "$tmp_file"
		elif command -v shasum >/dev/null 2>&1; then
			(cd "$output_dir" && shasum -a 256 "$artifact") >> "$tmp_file"
		else
			echo "No SHA-256 checksum tool found: need sha256sum or shasum" >&2
			exit 1
		fi
	done
	mv "$tmp_file" "$checksum_file"
}

build_one() {
	local goos="$1"
	local goarch="$2"
	local ext=""
	local outfile=""
	local artifact=""

	if [[ "$goos" == "windows" ]]; then
		ext=".exe"
	fi

	if [[ -n "$version" ]]; then
		artifact="${BIN_NAME}-${version}-${goos}-${goarch}${ext}"
	else
		artifact="${BIN_NAME}-${goos}-${goarch}${ext}"
	fi

	if [[ $release_mode -eq 0 && $all_mode -eq 0 && -z "$requested_os" && -z "$requested_arch" ]]; then
		outfile="${output_dir}/${BIN_NAME}${ext}"
	else
		outfile="${output_dir}/${artifact}"
	fi

	echo "building ${goos}/${goarch} -> ${outfile#$ROOT_DIR/}"
	(
		cd "$ROOT_DIR"
		CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" \
			go build -trimpath \
				-ldflags "-X github.com/martinghunt/mykrobe2/internal/buildinfo.Version=${build_version}" \
				-o "$outfile" "$CMD_PATH"
	)

	if [[ $release_mode -eq 1 ]]; then
		package_release_artifact "$goos" "$outfile" "$artifact"
	fi
}

if [[ $release_mode -eq 1 || $all_mode -eq 1 ]]; then
	for goos in darwin linux windows; do
		for goarch in amd64 arm64; do
			build_one "$goos" "$goarch"
		done
	done
	if [[ $release_mode -eq 1 ]]; then
		write_release_checksums
	fi
	exit 0
fi

host_os="${requested_os:-$(go env GOOS)}"
host_arch="${requested_arch:-$(go env GOARCH)}"
build_one "$host_os" "$host_arch"
