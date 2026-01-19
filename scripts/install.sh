#!/bin/bash

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BASE_DIR="$(dirname "$SCRIPT_DIR")"

HONEY='\033[0;33m'
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${HONEY}"
echo "🍯 HoneyRAG First-Time Setup"
echo "============================="
echo -e "${NC}"

cd "$BASE_DIR"

if ! command -v go &> /dev/null; then
    echo -e "${RED}✗ Go is not installed. Please install Go 1.22+ first.${NC}"
    exit 1
fi
echo -e "${GREEN}✓${NC} Go found"

if ! command -v uv &> /dev/null; then
    echo -e "${RED}✗ uv is not installed.${NC}"
    echo -e "Install: curl -LsSf https://astral.sh/uv/install.sh | sh"
    exit 1
fi
echo -e "${GREEN}✓${NC} uv found"

echo -e "\n${HONEY}Building HoneyRAG...${NC}"
go mod tidy
go build -o honeyrag ./cmd/honeyrag
echo -e "${GREEN}✓${NC} Binary built: ./honeyrag"

if [ ! -f "$BASE_DIR/configs/.env" ]; then
    cp "$BASE_DIR/configs/.env.example" "$BASE_DIR/configs/.env"
    echo -e "${GREEN}✓${NC} Config created: configs/.env"
fi

BOLD='\033[1m'

echo -e "\n${HONEY}"
echo "🍯 Setup complete!"
echo "=================="
echo -e "${NC}"
echo -e "Run: ${GREEN}./honeyrag${NC}"
echo ""
echo -e "${BOLD}NOTE: You only need to run install.sh once (first time).${NC}"
echo -e "${BOLD}      From now on, just run: ./honeyrag${NC}"
echo ""
