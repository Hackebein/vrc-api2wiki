#!/usr/bin/env bash
set -euo pipefail
umask 077

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SESSION_DIR="${ROOT}/.steam-session"
ARCHIVE="${ROOT}/.steam-session.tar.gpg"
MODE="${1:?usage: steam-session-cache.sh restore|save}"

case "${MODE}" in
  restore)
    if [[ ! -f "${ARCHIVE}" ]]; then
      echo 'No encrypted Steam session cached; the next login will create one.'
      exit 0
    fi
    ;;
  save)
    if [[ ! -d "${SESSION_DIR}" ]] || [[ -z "$(find "${SESSION_DIR}" -type f -name account.config -print -quit)" ]]; then
      echo 'No remembered Steam login to save.'
      exit 0
    fi
    ;;
  *) echo 'usage: steam-session-cache.sh restore|save' >&2; exit 1 ;;
esac

: "${STEAM_SHARED_SECRET:?STEAM_SHARED_SECRET is required to encrypt the Steam session}"
command -v gpg >/dev/null
TEMP_ARCHIVE="$(mktemp)"
trap 'rm -f "${TEMP_ARCHIVE}"' EXIT

if [[ "${MODE}" == restore ]]; then
  # Verify GPG's integrity protection before extracting any plaintext.
  gpg --batch --yes --no-tty --no-symkey-cache --pinentry-mode loopback --passphrase-fd 3 \
    --output "${TEMP_ARCHIVE}" --decrypt "${ARCHIVE}" 3<<<"${STEAM_SHARED_SECRET}"
  mkdir -p "${SESSION_DIR}"
  tar -xf "${TEMP_ARCHIVE}" -C "${SESSION_DIR}"
  echo 'Restored encrypted Steam session.'
else
  tar -cf "${TEMP_ARCHIVE}" -C "${SESSION_DIR}" .
  gpg --batch --yes --no-tty --no-symkey-cache --pinentry-mode loopback --passphrase-fd 3 \
    --symmetric --cipher-algo AES256 --output "${ARCHIVE}" "${TEMP_ARCHIVE}" 3<<<"${STEAM_SHARED_SECRET}"
  echo 'Encrypted Steam session for the next runner.'
fi
