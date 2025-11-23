#!/bin/bash
set -euo pipefail

# Build POML binary for all platforms
# Outputs: vscode-extension/bin/{darwin,linux,windows}/{amd64,arm64}/poml

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
BIN_DIR="$PROJECT_ROOT/vscode-extension/bin"
CMD_DIR="$PROJECT_ROOT/cmd/poml"

# Target platforms and architectures (plain array keeps macOS /bin/bash happy)
TARGETS=(
  "darwin:amd64"
  "darwin:arm64"
  "linux:amd64"
  "linux:arm64"
  "windows:amd64"
  "windows:arm64"
)

echo "Building POML binaries for all platforms..."

for target in "${TARGETS[@]}"; do
  IFS=':' read -r os arch <<< "$target"
  
  output_dir="$BIN_DIR/$os/$arch"
  mkdir -p "$output_dir"
  
  binary_name="poml"
  if [[ "$os" == "windows" ]]; then
    binary_name="poml.exe"
  fi
  
  output_path="$output_dir/$binary_name"
  
  echo "Building $os/$arch -> $output_path"
  
  GOOS="$os" GOARCH="$arch" go build \
    -ldflags="-s -w" \
    -o "$output_path" \
    "$CMD_DIR"
  
  # Make it executable (no-op on Windows)
  chmod +x "$output_path" 2>/dev/null || true
done

echo "✓ All binaries built successfully in $BIN_DIR"
