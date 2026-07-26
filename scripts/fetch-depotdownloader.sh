#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DEST="${ROOT}/third_party/DepotDownloader"
VERSION_FILE="${DEST}/VERSION"
VERSION="$(tr -d '[:space:]' <"${VERSION_FILE}")"
TAG="${VERSION}"

os="$(uname -s | tr '[:upper:]' '[:lower:]')"
arch="$(uname -m)"
case "${os}" in
  linux) os=linux ;;
  darwin) os=macos ;;
  mingw*|msys*|cygwin*|windows*) os=windows ;;
  *) echo "unsupported OS: ${os}" >&2; exit 1 ;;
esac
case "${arch}" in
  x86_64|amd64) arch=x64 ;;
  aarch64|arm64) arch=arm64 ;;
  armv7l|arm) arch=arm ;;
  *) echo "unsupported arch: ${arch}" >&2; exit 1 ;;
esac

ASSET="DepotDownloader-${os}-${arch}.zip"
URL="https://github.com/SteamRE/DepotDownloader/releases/download/${TAG}/${ASSET}"
BIN="${DEST}/DepotDownloader"
if [[ "${os}" == "windows" ]]; then
  BIN="${DEST}/DepotDownloader.exe"
fi

if [[ -x "${BIN}" ]] || [[ -f "${BIN}" ]]; then
  echo "DepotDownloader already present at ${BIN}"
  exit 0
fi

tmpdir="$(mktemp -d)"
trap 'rm -rf "${tmpdir}"' EXIT
echo "Downloading ${URL}"
curl -fsSL -o "${tmpdir}/${ASSET}" "${URL}"
unzip -qo "${tmpdir}/${ASSET}" -d "${tmpdir}/out"
# Find binary in extract tree
found="$(find "${tmpdir}/out" -type f \( -name 'DepotDownloader' -o -name 'DepotDownloader.exe' \) | head -n1)"
if [[ -z "${found}" ]]; then
  echo "DepotDownloader binary not found in ${ASSET}" >&2
  exit 1
fi
mkdir -p "${DEST}"
cp "${found}" "${BIN}"
chmod +x "${BIN}" || true
echo "Installed ${BIN}"
