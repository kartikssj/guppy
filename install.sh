#!/bin/sh
# guppy installer — downloads a release binary, verifies its SHA-256
# checksum against the release's checksums.txt, and installs it.
#
#   curl -fsSL https://raw.githubusercontent.com/kartikssj/guppy/main/install.sh | sh
#
# Optional environment:
#   GUPPY_VERSION      release to install (e.g. 1.0.0); default: latest
#   GUPPY_INSTALL_DIR  install location;          default: ~/.local/bin
set -eu

repo="https://github.com/kartikssj/guppy"

die() { echo "install.sh: $*" >&2; exit 1; }

# --- platform ------------------------------------------------------------
case "$(uname -s)" in
	Darwin) os=darwin ;;
	Linux) os=linux ;;
	*) die "unsupported operating system: $(uname -s) (guppy is macOS/Linux only)" ;;
esac

case "$(uname -m)" in
	arm64 | aarch64) arch=arm64 ;;
	x86_64) arch=amd64 ;;
	*) die "unsupported architecture: $(uname -m)" ;;
esac

# --- version -------------------------------------------------------------
if [ -n "${GUPPY_VERSION:-}" ]; then
	case "$GUPPY_VERSION" in v*) tag="$GUPPY_VERSION" ;; *) tag="v$GUPPY_VERSION" ;; esac
	# This string ends up inside URLs and paths — keep its shape tight.
	case "$tag" in
		*[!0-9a-zA-Z.-]*) die "invalid GUPPY_VERSION: $GUPPY_VERSION (expected e.g. 1.0.0)" ;;
		v[0-9]*.[0-9]*.[0-9]*) ;;
		*) die "invalid GUPPY_VERSION: $GUPPY_VERSION (expected e.g. 1.0.0)" ;;
	esac
else
	# Resolve "latest" from the redirect target instead of the API to avoid
	# GitHub's unauthenticated rate limit.
	tag=$(curl -fsSI "$repo/releases/latest" | tr -d '\r' | sed -n 's/^[Ll]ocation: .*\/tag\///p')
	[ -n "$tag" ] || die "could not resolve the latest release"
fi

archive="guppy_${tag}_${os}_${arch}.tar.gz"
base="$repo/releases/download/$tag"

# --- download + verify -----------------------------------------------------
tmp=$(mktemp -d) || die "mktemp failed"
trap 'rm -rf "$tmp"' EXIT

echo "downloading guppy $tag ($os/$arch)"
curl -fsSL -o "$tmp/$archive" "$base/$archive" || die "download failed: $base/$archive"
curl -fsSL -o "$tmp/checksums.txt" "$base/checksums.txt" || die "download failed: $base/checksums.txt"

if command -v sha256sum >/dev/null 2>&1; then
	actual=$(sha256sum "$tmp/$archive" | cut -d' ' -f1)
elif command -v shasum >/dev/null 2>&1; then
	actual=$(shasum -a 256 "$tmp/$archive" | cut -d' ' -f1)
else
	die "no sha256 tool found (need sha256sum or shasum); refusing to install unverified"
fi

expected=$(awk -v f="$archive" '$2 == f { print $1 }' "$tmp/checksums.txt")
[ -n "$expected" ] || die "no checksum for $archive in checksums.txt; refusing to install"
[ "$actual" = "$expected" ] || die "checksum mismatch for $archive; refusing to install"

# --- install ---------------------------------------------------------------
tar -xzf "$tmp/$archive" -C "$tmp" || die "failed to extract $archive"

dest=${GUPPY_INSTALL_DIR:-"$HOME/.local/bin"}
mkdir -p "$dest"
cp "$tmp/guppy" "$dest/guppy"
chmod 0755 "$dest/guppy"

echo "installed $dest/guppy"

case ":$PATH:" in
	*":$dest:"*) ;;
	*) echo "note: $dest is not on your PATH — add: export PATH=\"$dest:\$PATH\"" ;;
esac
