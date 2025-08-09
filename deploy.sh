#!/bin/bash
set -euo pipefail

APP_DIR="/projects/ads-go"
RELEASES_DIR="$APP_DIR/releases"
BIN_LINK="$APP_DIR/bin"
WORKTREES_DIR="$APP_DIR/.worktrees"
HEALTH_PATH="/ads?type=home"      # ajuste se quiser outro healthcheck
TEST_URL_8089="http://127.0.0.1:8089${HEALTH_PATH}"
TEST_URL_8088="http://127.0.0.1:8088${HEALTH_PATH}"

cd "$APP_DIR"

# 0) checagens básicas
command -v git >/dev/null || { echo "git não instalado"; exit 1; }
command -v go  >/dev/null || { echo "go não instalado"; exit 1; }

mkdir -p "$RELEASES_DIR" "$WORKTREES_DIR"

# 1) descobrir tag mais recente no repositório
echo "[1/7] Buscando tags..."
git fetch origin --tags --prune

# tenta ordenar semver; se não houver, cai para a mais recente por data
LATEST_TAG="$(git tag --sort=-version:refname | head -n1 || true)"
if [[ -z "${LATEST_TAG}" ]]; then
  # fallback por data do commit da tag
  LATEST_TAG="$(git describe --tags "$(git rev-list --tags --max-count=1)")"
fi
if [[ -z "${LATEST_TAG}" ]]; then
  echo "Nenhuma tag encontrada no repositório."
  exit 1
fi
echo "Última tag: ${LATEST_TAG}"

# 2) descobrir tag atualmente em produção (pelo symlink bin/VERSION)
CURRENT_TAG=""
if [[ -L "$BIN_LINK" ]]; then
  if [[ -f "$BIN_LINK/VERSION" ]]; then
    CURRENT_TAG="$(cat "$BIN_LINK/VERSION" || true)"
  fi
fi
echo "Tag atual:  ${CURRENT_TAG:-<nenhuma>}"

if [[ "$CURRENT_TAG" == "$LATEST_TAG" ]]; then
  echo "Nada para fazer: já estamos na tag $LATEST_TAG."
  exit 0
fi

# 3) preparar worktree para a tag (não mexe no diretório principal)
WT_PATH="$WORKTREES_DIR/$LATEST_TAG"
if [[ -d "$WT_PATH" ]]; then
  echo "[2/7] Worktree já existe: $WT_PATH"
else
  echo "[2/7] Criando worktree: $WT_PATH"
  git worktree add -f "$WT_PATH" "$LATEST_TAG"
fi

# 4) build para releases/<TAG>
NEW_RELEASE="$RELEASES_DIR/$LATEST_TAG"
mkdir -p "$NEW_RELEASE"
echo "[3/7] Build da tag $LATEST_TAG..."
pushd "$WT_PATH" >/dev/null
GOFLAGS="-trimpath" CGO_ENABLED=0 go build -ldflags="-s -w" -o "$NEW_RELEASE/ads-go" ./cmd/server
popd >/dev/null

# grava a versão da release
echo "$LATEST_TAG" > "$NEW_RELEASE/VERSION"

# 5) troca atômica do symlink bin -> releases/<TAG>
echo "[4/7] Apontando bin -> $NEW_RELEASE"
ln -sfn "$NEW_RELEASE" "$BIN_LINK"

# 6) restart sem downtime: 8089 -> testa -> 8088
echo "[5/7] Reiniciando 8089..."
sudo systemctl restart ads-go@8089
sleep 2
if curl -fsS "$TEST_URL_8089" >/dev/null; then
  echo "8089 OK ✅"
else
  echo "Falha no healthcheck de 8089 ❌"
  exit 1
fi

echo "[6/7] Reiniciando 8088..."
sudo systemctl restart ads-go@8088
sleep 2
if curl -fsS "$TEST_URL_8088" >/dev/null; then
  echo "8088 OK ✅"
else
  echo "Falha no healthcheck de 8088 ❌"
  exit 1
fi

echo "[7/7] Deploy concluído! Atualizado para a tag: $LATEST_TAG"
