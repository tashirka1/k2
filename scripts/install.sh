#!/usr/bin/env bash
set -euo pipefail

# K2 server monitoring - installer
# Usage: curl -fsSL https://raw.githubusercontent.com/tashirka1/k2/main/scripts/install.sh | sudo bash

REPO="tashirka1/k2"
ASSET="k2-linux-amd64"
BIN="/usr/local/bin/k2"
DATA_DIR="/opt/k2"
ENV_FILE="${DATA_DIR}/k2.env"
DB_FILE="${DATA_DIR}/k2.db"
SERVICE_FILE="/etc/systemd/system/k2.service"
DOWNLOAD_URL="https://github.com/${REPO}/releases/latest/download/${ASSET}"

if [[ ${EUID} -ne 0 ]]; then
	echo "This script must run as root" >&2
	exit 1
fi

if ! command -v openssl >/dev/null 2>&1; then
	echo "openssl is required" >&2
	exit 1
fi

if ! command -v wget >/dev/null 2>&1 && ! command -v curl >/dev/null 2>&1; then
	echo "wget or curl is required" >&2
	exit 1
fi

echo "Downloading ${DOWNLOAD_URL}"
tmp="$(mktemp)"
trap 'rm -f "${tmp}"' EXIT
if command -v wget >/dev/null 2>&1; then
	wget -qO "${tmp}" "${DOWNLOAD_URL}"
else
	curl -fsSL -o "${tmp}" "${DOWNLOAD_URL}"
fi
install -m 0755 "${tmp}" "${BIN}"
echo "Installed ${BIN}"

mkdir -p "${DATA_DIR}"

if [[ ! -f "${ENV_FILE}" ]]; then
	umask 077
	cat > "${ENV_FILE}" <<EOF
K2_SESSION_KEY=$(openssl rand -hex 32)
K2_PORT=8000
K2_DB_NAME=${DB_FILE}
K2_USERNAME=admin
K2_PASSWORD=$(openssl rand -base64 18)
K2_COLLECT_INTERVAL=30s
K2_RETENTION=168h
EOF
	echo "Created ${ENV_FILE}"
else
	echo "${ENV_FILE} already exists, keeping it"
fi

cat > "${SERVICE_FILE}" <<EOF
[Unit]
Description=K2 server monitoring
After=network.target

[Service]
Type=simple
User=root
Group=root
WorkingDirectory=${DATA_DIR}
EnvironmentFile=${ENV_FILE}
ExecStart=${BIN}
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable --now k2

echo
echo "K2 installed. Service status:"
systemctl --no-pager --full status k2

echo
echo "Credentials:"
echo "  Username: $(grep '^K2_USERNAME=' "${ENV_FILE}" | cut -d= -f2-)"
echo "  Password: $(grep '^K2_PASSWORD=' "${ENV_FILE}" | cut -d= -f2-)"
