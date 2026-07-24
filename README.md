# k2

## how to run

Development
```bash
cp env-example .env
make dev   # run dev server with autoreload
```

Production Docker
```bash
cp env-example .env
make up    # run docker container
```

Production Binary
```bash
sudo apt-get update && sudo apt-get install -y --no-install-recommends build-essential libsqlite3-dev
cp env-example .env
make build-bin
./bin/k2 # run binary
```

## docs

[tutorial](/docs/tutorial)
