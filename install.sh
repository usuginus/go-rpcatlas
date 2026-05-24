#!/usr/bin/env sh
set -eu

repo="github.com/usuginus/go-rpcatlas"
package="$repo/cmd/rpcatlas"
binary_name="rpcatlas"

version="${VERSION:-latest}"
source_dir="${SOURCE_DIR:-}"

if ! command -v go >/dev/null 2>&1; then
  echo "error: go is required to install $binary_name" >&2
  exit 1
fi

if [ -n "${INSTALL_DIR:-}" ]; then
  install_dir="$INSTALL_DIR"
elif [ -n "${GOBIN:-}" ]; then
  install_dir="$GOBIN"
else
  install_dir="$(go env GOPATH)/bin"
fi

tmp_dir="$(mktemp -d "${TMPDIR:-/tmp}/rpcatlas-install.XXXXXX")"
cleanup() {
  rm -rf "$tmp_dir"
}
trap cleanup EXIT INT TERM

mkdir -p "$tmp_dir/bin" "$install_dir"

if [ -n "$source_dir" ]; then
  echo "Installing $binary_name from $source_dir..."
  (
    cd "$source_dir"
    GOBIN="$tmp_dir/bin" go install ./cmd/rpcatlas
  )
else
  echo "Installing $package@$version..."
  GOBIN="$tmp_dir/bin" go install "$package@$version"
fi

cp "$tmp_dir/bin/$binary_name" "$install_dir/$binary_name"
chmod +x "$install_dir/$binary_name"

echo "Installed $binary_name to $install_dir/$binary_name"

case ":$PATH:" in
  *":$install_dir:"*) ;;
  *) echo "Note: add $install_dir to PATH to run $binary_name from any directory." ;;
esac
