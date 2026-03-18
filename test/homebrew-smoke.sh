#!/bin/sh

set -eu

if ! command -v brew >/dev/null 2>&1; then
  echo "Homebrew is not installed; skipping brew smoke test" >&2
  exit 0
fi

TMP_ROOT="$(mktemp -d)"
RELEASE_DIR="${TMP_ROOT}/release"
VERSION="v0.0.0-test"
FORMULA_PATH="${TMP_ROOT}/grafana-cli.rb"
TAP_NAME="local/grafana-cli-smoke-$PPID"

cleanup() {
  HOMEBREW_NO_AUTO_UPDATE=1 brew uninstall --formula grafana-cli >/dev/null 2>&1 || true
  HOMEBREW_NO_AUTO_UPDATE=1 brew untap "${TAP_NAME}" >/dev/null 2>&1 || true
  if [ -n "${SERVER_PID:-}" ]; then
    kill "${SERVER_PID}" >/dev/null 2>&1 || true
  fi
  rm -rf "${TMP_ROOT}"
}
trap cleanup EXIT INT TERM

mkdir -p "${RELEASE_DIR}"

case "$(uname -s)" in
  Darwin) OS="darwin" ;;
  Linux) OS="linux" ;;
  *) echo "unsupported operating system: $(uname -s)" >&2; exit 1 ;;
esac

case "$(uname -m)" in
  x86_64|amd64) ARCH="amd64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  *) echo "unsupported architecture: $(uname -m)" >&2; exit 1 ;;
esac

GOOS="${OS}" GOARCH="${ARCH}" CGO_ENABLED=0 go build -trimpath -o "${RELEASE_DIR}/grafana" ./cmd/grafana

for target in darwin_amd64 darwin_arm64 linux_amd64 linux_arm64; do
  ARCHIVE="grafana_${VERSION}_${target}.tar.gz"
  tar -C "${RELEASE_DIR}" -czf "${RELEASE_DIR}/${ARCHIVE}" grafana
done

(
  cd "${RELEASE_DIR}"
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum *.tar.gz > checksums.txt
  else
    shasum -a 256 *.tar.gz > checksums.txt
  fi
)

PORT="$(python3 - <<'PY'
import socket
s = socket.socket()
s.bind(("127.0.0.1", 0))
print(s.getsockname()[1])
s.close()
PY
)"

python3 -m http.server "${PORT}" --bind 127.0.0.1 --directory "${RELEASE_DIR}" >/dev/null 2>&1 &
SERVER_PID=$!
sleep 1

go run ./cmd/release-assets homebrew \
  --repo matiasvillaverde/grafana-cli \
  --tag "${VERSION}" \
  --download-base-url "http://127.0.0.1:${PORT}" \
  --checksums "${RELEASE_DIR}/checksums.txt" > "${FORMULA_PATH}"

HOMEBREW_NO_AUTO_UPDATE=1 brew tap-new "${TAP_NAME}" --no-git >/dev/null
TAP_DIR="$(brew --repo "${TAP_NAME}")"
mkdir -p "${TAP_DIR}/Formula"
cp "${FORMULA_PATH}" "${TAP_DIR}/Formula/grafana-cli.rb"

HOMEBREW_NO_AUTO_UPDATE=1 brew install "${TAP_NAME}/grafana-cli"
HOMEBREW_NO_AUTO_UPDATE=1 brew test grafana-cli

BREW_BIN="$(brew --prefix)/bin/grafana"
HELP_OUTPUT="$("${BREW_BIN}" help)"
printf '%s\n' "${HELP_OUTPUT}" | grep '"auth"'
