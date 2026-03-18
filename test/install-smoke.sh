#!/bin/sh

set -eu

TMP_ROOT="$(mktemp -d)"
RELEASE_DIR="${TMP_ROOT}/release"
BINDIR="${TMP_ROOT}/bin"
VERSION="v0.0.0-test"

cleanup() {
  if [ -n "${SERVER_PID:-}" ]; then
    kill "${SERVER_PID}" >/dev/null 2>&1 || true
  fi
  rm -rf "${TMP_ROOT}"
}
trap cleanup EXIT INT TERM

mkdir -p "${RELEASE_DIR}" "${BINDIR}"

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
ARCHIVE="grafana_${VERSION}_${OS}_${ARCH}.tar.gz"
tar -C "${RELEASE_DIR}" -czf "${RELEASE_DIR}/${ARCHIVE}" grafana

(
  cd "${RELEASE_DIR}"
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "${ARCHIVE}" > checksums.txt
  else
    shasum -a 256 "${ARCHIVE}" > checksums.txt
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

INSTALL_OUTPUT="$(
  BINDIR="${BINDIR}" \
  GRAFANA_INSTALL_VERSION="${VERSION}" \
  GRAFANA_INSTALL_BASE_URL="http://127.0.0.1:${PORT}" \
  sh scripts/install.sh
)"

HELP_OUTPUT="$("${BINDIR}/grafana" help)"

printf '%s\n' "${INSTALL_OUTPUT}" | grep 'Installed grafana'
printf '%s\n' "${HELP_OUTPUT}" | grep '"auth"'
