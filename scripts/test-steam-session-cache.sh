#!/usr/bin/env bash
set -euo pipefail
umask 077

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
TEMP_ROOT="$(cd "${TMPDIR:-/tmp}" && pwd)"
TEST_DIR="$(mktemp -d "${TEMP_ROOT}/steam-session-test.XXXXXXXX")"
cleanup() {
  gpgconf --kill all >/dev/null 2>&1 || true
  if [[ "${TEST_DIR}" == "${TEMP_ROOT}/steam-session-test."* ]]; then
    rm -rf -- "${TEST_DIR}"
  fi
}
trap cleanup EXIT
export GNUPGHOME="${TEST_DIR}/gnupg"
mkdir "${GNUPGHOME}"
# Synthetic fixture only; never read real account credentials in this test.
export STEAM_SHARED_SECRET='c3ludGhldGljLXNlc3Npb24tdGVzdA=='

for runner in first second wrong-key corrupt empty; do
  mkdir -p "${TEST_DIR}/${runner}/scripts"
  cp "${SCRIPT_DIR}/steam-session-cache.sh" "${TEST_DIR}/${runner}/scripts/"
done
FIRST="${TEST_DIR}/first"
SECOND="${TEST_DIR}/second"
mkdir -p "${FIRST}/.steam-session/.local/share/IsolatedStorage/fixture/AssemFiles"
printf 'fixture-refresh-token' >"${FIRST}/.steam-session/.local/share/IsolatedStorage/fixture/AssemFiles/account.config"

bash "${FIRST}/scripts/steam-session-cache.sh" save
if grep -aFq 'fixture-refresh-token' "${FIRST}/.steam-session.tar.gpg"; then
  echo 'Plaintext token appeared in the cached archive' >&2
  exit 1
fi

# A fresh runner gets only the encrypted archive, not the original session tree.
cp "${FIRST}/.steam-session.tar.gpg" "${SECOND}/"
bash "${SECOND}/scripts/steam-session-cache.sh" restore
cmp "${FIRST}/.steam-session/.local/share/IsolatedStorage/fixture/AssemFiles/account.config" \
    "${SECOND}/.steam-session/.local/share/IsolatedStorage/fixture/AssemFiles/account.config"

cp "${FIRST}/.steam-session.tar.gpg" "${TEST_DIR}/wrong-key/"
if STEAM_SHARED_SECRET=wrong-key bash "${TEST_DIR}/wrong-key/scripts/steam-session-cache.sh" restore; then
  echo 'Restored a session with the wrong key' >&2
  exit 1
fi
test ! -e "${TEST_DIR}/wrong-key/.steam-session"

printf 'corrupt ciphertext' >"${TEST_DIR}/corrupt/.steam-session.tar.gpg"
if bash "${TEST_DIR}/corrupt/scripts/steam-session-cache.sh" restore; then
  echo 'Restored a corrupt archive' >&2
  exit 1
fi
test ! -e "${TEST_DIR}/corrupt/.steam-session"

bash "${TEST_DIR}/empty/scripts/steam-session-cache.sh" restore
bash "${TEST_DIR}/empty/scripts/steam-session-cache.sh" save
test ! -e "${TEST_DIR}/empty/.steam-session.tar.gpg"
echo 'Steam session cache round-trip and failure checks passed.'
