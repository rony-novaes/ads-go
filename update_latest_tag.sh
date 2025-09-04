#!/bin/bash
set -euo pipefail

APP_DIR="/projects/search-go"
BIN_PATH="$APP_DIR/bin/search-go"
ENV_SRC="/projects/envs/search-go/.env"     # de onde copiar
ENV_DST="/projects/search-go/.env"          # para onde copiar (lido pelo systemd)
SERVICES=("search-go@8081" "search-go@8082" "search-go@8083")   # reinicia B->A para evitar downtime
HEALTH_PATH="?q=prefeitura"
HEALTH_8081="http://127.0.0.1:8081${HEALTH_PATH}"
HEALTH_8082="http://127.0.0.1:8082${HEALTH_PATH}"
HEALTH_8082="http://127.0.0.1:8083${HEALTH_PATH}"

REQ_TAG="${1:-}"  # opcional: passar a tag desejada

echo "[1/8] Indo para $APP_DIR"
cd "$APP_DIR"

echo "[2/8] Buscando tags do remoto..."
git fetch --tags

if [[ -n "$REQ_TAG" ]]; then
  if git show-ref --tags --verify --quiet "refs/tags/$REQ_TAG"; then
    LATEST_TAG="$REQ_TAG"
  else
    echo "✖ Tag '$REQ_TAG' não existe no remoto"; exit 1
  fi
else
  LATEST_TAG=$(git describe --tags $(git rev-list --tags --max-count=1))
fi
echo "[3/8] Tag alvo: $LATEST_TAG"

echo "[4/8] Checando a tag..."
git reset --hard
git checkout "$LATEST_TAG"

echo "[5/8] Copiando .env de $ENV_SRC -> $ENV_DST"
if [[ ! -f "$ENV_SRC" ]]; then
  echo "✖ Arquivo .env de origem não encontrado: $ENV_SRC"; exit 1
fi
cp -f "$ENV_SRC" "$ENV_DST"
chown www-data:www-data "$ENV_DST" || true
chmod 640 "$ENV_DST" || true

echo "[6/8] Build do main (./cmd/server)..."
mkdir -p "$(dirname "$BIN_PATH")"
CGO_ENABLED=0 GOFLAGS="-trimpath" go build -ldflags="-s -w" -o "$BIN_PATH" ./cmd/server
chown www-data:www-data "$BIN_PATH" || true
chmod 755 "$BIN_PATH" || true

restart_and_wait() {
  local unit="$1"
  local url="$2"
  echo " - Reiniciando $unit"
  systemctl restart "$unit"

  # espera até 15s pelo healthcheck
  for i in {1..15}; do
    if curl -fsS "$url" >/dev/null; then
      echo "   ✓ $unit saudável"
      return 0
    fi
    sleep 1
  done
  echo "   ✖ $unit falhou no healthcheck — logs recentes:"
  journalctl -u "$unit" -n 120 --no-pager || true
  exit 1
}

echo "[7/8] Reiniciando serviços com healthcheck..."
restart_and_wait "${SERVICES[0]}" "$HEALTH_8081"
restart_and_wait "${SERVICES[1]}" "$HEALTH_8082"
restart_and_wait "${SERVICES[1]}" "$HEALTH_8083"

echo "[8/8] ✅ Deploy ok. Rodando a tag $LATEST_TAG"
