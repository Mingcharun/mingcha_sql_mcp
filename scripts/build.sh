#!/bin/bash

set -euo pipefail

VERSION=${1:-dev}

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
DIST_DIR="${REPO_ROOT}/dist"
OUTPUT_BIN="${DIST_DIR}/database-mcp"

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}  Database MCP Build Script${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""

cd "${REPO_ROOT}"

echo -e "${YELLOW}[1/5]${NC} Preparing output directory..."
mkdir -p "${DIST_DIR}"
rm -f "${OUTPUT_BIN}"

echo -e "${YELLOW}[2/5]${NC} Syncing dependencies..."
go mod tidy

echo -e "${YELLOW}[3/5]${NC} Building project (version: ${VERSION})..."
go build -ldflags "-s -w -X main.Version=${VERSION}" -o "${OUTPUT_BIN}" ./cmd/database-mcp

echo -e "${YELLOW}[4/5]${NC} Setting executable permissions..."
chmod +x "${OUTPUT_BIN}"

echo -e "${YELLOW}[5/5]${NC} Build complete"
echo ""
echo -e "${GREEN}✅ Build finished${NC}"
echo "Binary: ${OUTPUT_BIN}"
echo ""
echo "Version:"
"${OUTPUT_BIN}" --version
