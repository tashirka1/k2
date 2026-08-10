# Installation

## Bare-metal

```bash
# Download latest binary
wget -O /usr/local/bin/k2 https://github.com/tashirka1/k2/releases/latest/download/k2-linux-amd64
chmod +x /usr/local/bin/k2

# Create data directory
mkdir -p /var/lib/k2

# Create env file
cat > /etc/k2/.env <<EOF
K2_SESSION_KEY=$(openssl rand -hex 32)
K2_PORT=8000
K2_DB_NAME=/var/lib/k2/data.db
K2_USERNAME=admin
K2_PASSWORD=$(openssl rand -base64 18)
K2_COLLECT_INTERVAL=30s
K2_RETENTION=168h
EOF

# Install and start service
cat > /etc/systemd/system/k2.service <<EOF
[Unit]
Description=K2 server monitoring
After=network.target

[Service]
Type=simple
User=noroot
Group=noroot
WorkingDirectory=/var/lib/k2
EnvironmentFile=/etc/k2/.env
ExecStart=/usr/local/bin/k2
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF
systemctl daemon-reload
systemctl enable --now k2
```

## Docker

See [docker.md](docker.md).
