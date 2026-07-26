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
EOF

# Install and start service
cp contrib/k2.service /etc/systemd/system/
systemctl daemon-reload
systemctl enable --now k2
```

## Docker

See [docker.md](docker.md).
