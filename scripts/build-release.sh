#!/usr/bin/env bash
set -euo pipefail

version="${1:-dev}"
dist_dir="${DIST_DIR:-dist}"
name="go-dependency-analysis"
targets=(
  "linux/amd64"
  "linux/arm64"
  "darwin/amd64"
  "darwin/arm64"
  "windows/amd64"
)

rm -rf "$dist_dir"
mkdir -p "$dist_dir"

for target in "${targets[@]}"; do
  goos="${target%/*}"
  goarch="${target#*/}"
  base="${name}_${version}_${goos}_${goarch}"
  stage="$dist_dir/$base"
  binary="$name"
  if [[ "$goos" == "windows" ]]; then
    binary+=".exe"
  fi

  mkdir -p "$stage"
  CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" go build \
    -trimpath \
    -ldflags="-s -w" \
    -o "$stage/$binary" .

  if [[ "$goos" == "windows" ]]; then
    (
      cd "$dist_dir"
      zip -q -X -r "$base.zip" "$base"
    )
  else
    tar -C "$dist_dir" -czf "$dist_dir/$base.tar.gz" "$base"
  fi
  rm -rf "$stage"
done

(
  cd "$dist_dir"
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum ./*.tar.gz ./*.zip > checksums.txt
  else
    shasum -a 256 ./*.tar.gz ./*.zip > checksums.txt
  fi
)
