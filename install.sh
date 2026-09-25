#!/bin/sh
# OpenAlgo CLI installer for macOS and Linux.
#
#   curl -fsSL https://raw.githubusercontent.com/marketcalls/openalgo-cli/main/install.sh | sh
#
# Environment overrides:
#   OPENALGO_CLI_VERSION   release tag to install (default: latest), e.g. v0.0.1
#   OPENALGO_INSTALL_DIR   install directory (default: /usr/local/bin, or ~/.local/bin if not writable)
set -eu

REPO="marketcalls/openalgo-cli"
BINARY="openalgo"

err() {
	echo "error: $*" >&2
	exit 1
}

need() {
	command -v "$1" >/dev/null 2>&1 || err "$1 is required"
}

need uname
need tar
if command -v curl >/dev/null 2>&1; then
	fetch() { curl -fsSL "$1"; }
	download() { curl -fsSL -o "$2" "$1"; }
elif command -v wget >/dev/null 2>&1; then
	fetch() { wget -qO- "$1"; }
	download() { wget -qO "$2" "$1"; }
else
	err "curl or wget is required"
fi

os=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$os" in
darwin | linux) ;;
*) err "unsupported OS: $os (on Windows use install.ps1)" ;;
esac

arch=$(uname -m)
case "$arch" in
x86_64 | amd64) arch="amd64" ;;
arm64 | aarch64) arch="arm64" ;;
*) err "unsupported architecture: $arch" ;;
esac

version="${OPENALGO_CLI_VERSION:-}"
if [ -z "$version" ]; then
	version=$(fetch "https://api.github.com/repos/$REPO/releases/latest" |
		sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' | head -n 1)
	[ -n "$version" ] || err "could not determine the latest release; set OPENALGO_CLI_VERSION"
fi
num="${version#v}"

archive="openalgo-cli_${num}_${os}_${arch}.tar.gz"
base="https://github.com/$REPO/releases/download/$version"

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

echo "Downloading $BINARY $version ($os/$arch)..."
download "$base/$archive" "$tmp/$archive" || err "download failed: $base/$archive"
download "$base/checksums.txt" "$tmp/checksums.txt" || err "download failed: $base/checksums.txt"

expected=$(grep " $archive\$" "$tmp/checksums.txt" | awk '{print $1}')
[ -n "$expected" ] || err "no checksum for $archive"
if command -v sha256sum >/dev/null 2>&1; then
	actual=$(sha256sum "$tmp/$archive" | awk '{print $1}')
else
	actual=$(shasum -a 256 "$tmp/$archive" | awk '{print $1}')
fi
[ "$expected" = "$actual" ] || err "checksum mismatch for $archive"

tar -xzf "$tmp/$archive" -C "$tmp" "$BINARY"

dir="${OPENALGO_INSTALL_DIR:-}"
if [ -z "$dir" ]; then
	if [ -w /usr/local/bin ]; then
		dir=/usr/local/bin
	else
		dir="$HOME/.local/bin"
	fi
fi
mkdir -p "$dir"
install -m 0755 "$tmp/$BINARY" "$dir/$BINARY"

echo "Installed $BINARY $version to $dir/$BINARY"
case ":$PATH:" in
*":$dir:"*) ;;
*) echo "Note: $dir is not on your PATH. Add it with: export PATH=\"$dir:\$PATH\"" ;;
esac
echo "Next: openalgo profile login"
