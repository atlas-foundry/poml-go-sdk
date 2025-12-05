#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
UPSTREAM_REPO="${UPSTREAM_REPO:-microsoft/poml}"
TAG="${TAG:-${UPSTREAM_TAG:-}}"

if [[ -z "${TAG}" ]]; then
  cat <<'EOF'
Usage: TAG=vX.Y.Z make sync-upstream

Environment variables:
  TAG            Required. Upstream release tag, e.g., v0.5.0.
  UPSTREAM_REPO  Defaults to poml-lang/poml.
  SYNC_MIRROR_TAGS Set to 1 to force-update the matching tag locally after syncing assets.
EOF
  exit 1
fi

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "❌ Required command not found: $1" >&2
    exit 1
  fi
}

for bin in curl tar go; do
  require_cmd "$bin"
done

ARCHIVE_URL="https://github.com/${UPSTREAM_REPO}/archive/refs/tags/${TAG}.tar.gz"
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "${TMP_DIR}"' EXIT

echo "🔁 Downloading ${UPSTREAM_REPO}@${TAG}"
curl -sSfL "${ARCHIVE_URL}" -o "${TMP_DIR}/upstream.tar.gz"

tar -xzf "${TMP_DIR}/upstream.tar.gz" -C "${TMP_DIR}"
ARCHIVE_DIR="$(find "${TMP_DIR}" -maxdepth 1 -type d -name "$(basename "${UPSTREAM_REPO}")-*" | head -n1)"
if [[ -z "${ARCHIVE_DIR}" ]]; then
  echo "❌ Failed to locate extracted archive directory" >&2
  exit 1
fi

TARGET_ROOT="${ROOT_DIR}/third_party/upstream"
declare -a SYNC_DIRS=("examples" "docs" "gallery")

mkdir -p "${TARGET_ROOT}"

for dir in "${SYNC_DIRS[@]}"; do
  SRC="${ARCHIVE_DIR}/${dir}"
  DST="${TARGET_ROOT}/${dir}"
  if [[ ! -d "${SRC}" ]]; then
    echo "⚠️  Skipping missing upstream directory: ${dir}"
    continue
  fi
  rm -rf "${DST}"
  mkdir -p "$(dirname "${DST}")"
  cp -R "${SRC}" "${DST}"
  echo "✅ Synced ${dir}"
done

cat <<EOF >"${TARGET_ROOT}/MANIFEST"
upstream_repo=${UPSTREAM_REPO}
tag=${TAG}
updated_at=$(date -u +'%%Y-%%m-%%dT%%H:%%M:%%SZ')
EOF

cat <<EOF >"${ROOT_DIR}/UPSTREAM_VERSION"
upstream_repo=${UPSTREAM_REPO}
tag=${TAG}
synced_at=$(date -u +'%%Y-%%m-%%dT%%H:%%M:%%SZ')
EOF

echo "🧪 Running parity tests"
(
  cd "${ROOT_DIR}"
  go test ./poml/... | tee "${TMP_DIR}/parity.log"
)

if [[ "${SYNC_MIRROR_TAGS:-0}" == "1" ]]; then
  (
    cd "${ROOT_DIR}"
    git tag -a "${TAG}" -m "Upstream parity ${TAG} (source ${UPSTREAM_REPO}@${TAG})" -f
    echo "🏷  Updated local tag ${TAG}"
  )
fi

echo "✨ Upstream assets refreshed from ${UPSTREAM_REPO}@${TAG}"
