#!/bin/bash
set -euo pipefail

APP_DIR="/projects/search-go"
BIN_PATH="$APP_DIR/bin/search-go"
ENV_PATH="/projects/envs/search-go/.env"
SERVICES=("search-go@8081" "search-go@8082" "search-go@8083")

HEALTH_PATH="?q=prefeitura"
HEALTH_8081="http://127.0.0.1:8081${HEALTH_PATH}"
HEALTH_8082="http://127.0.0.1:8082${HEALTH_PATH}"
HEALTH_8083="http://127.0.0.1:8083${HEALTH_PATH}"

REQ_TAG="${1:-}"  # opcional: passar a tag desejada

echo "[1/7] Indo para $APP_DIR"
cd "$APP_DIR"

echo "[2/7] Conferindo .env em $ENV_PATH"
if [[ ! -f "$ENV_PATH" ]]; then
  echo "✖ Arquivo .env não encontrado: $ENV_PATH"
  exit 1
fi
# Permissões recomendadas (não obrigatório, apenas boa prática)
chown root:root "$ENV_PATH" || true
chmod 640 "$ENV_PATH" || true

echo "[3/7] Buscando tags do remoto..."
git fetch --tags

if [[ -n "$REQ_TAG" ]]; then
  if git show-ref --tags --verify --quiet "refs/tags/$REQ_TAG"; then
    LATEST_TAG="$REQ_TAG"
  else
    echo "✖ Tag '$REQ_TAG' não existe no remoto"
    exit 1
  fi
else
  LATEST_TAG=$(git describe --tags "$(git rev-list --tags --max-count=1)")
fi
echo "    Tag alvo: $LATEST_TAG"

echo "[4/7] Checando a tag..."
git reset --hard
git checkout "$LATEST_TAG"

echo "[5/7] Build do main (./cmd/server)..."
mkdir -p "$(dirname "$BIN_PATH")"
CGO_ENABLED=0 GOFLAGS="-trimpath" go build -ldflags="-s -w" -o "$BIN_PATH" ./cmd/server
chown root:root "$BIN_PATH" || true
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

echo "[6/7] Reiniciando serviços com healthcheck..."
restart_and_wait "${SERVICES[0]}" "$HEALTH_8081"
restart_and_wait "${SERVICES[1]}" "$HEALTH_8082"
restart_and_wait "${SERVICES[2]}" "$HEALTH_8083"

echo "[7/7] ✅ Deploy ok. Rodando a tag $LATEST_TAG"
