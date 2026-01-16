#!/bin/bash

# =============================================================================
# HoneyRAG Installation Script
# =============================================================================
# Sets up Python environment and builds the Go binary
# =============================================================================

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BASE_DIR="$(dirname "$SCRIPT_DIR")"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
HONEY='\033[0;33m'
NC='\033[0m' # No Color

echo -e "${HONEY}"
echo "🍯 HoneyRAG Installer"
echo "====================="
echo -e "${NC}"

# -----------------------------------------------------------------------------
# Check Dependencies
# -----------------------------------------------------------------------------
echo -e "${BLUE}Checking dependencies...${NC}"

check_command() {
    if command -v "$1" &> /dev/null; then
        echo -e "  ${GREEN}✓${NC} $1 found"
        return 0
    else
        echo -e "  ${RED}✗${NC} $1 not found"
        return 1
    fi
}

MISSING=0

check_command "python3" || MISSING=1
check_command "uv" || MISSING=1
check_command "go" || MISSING=1
check_command "nvidia-smi" || MISSING=1

if [ $MISSING -eq 1 ]; then
    echo -e "\n${RED}Missing dependencies. Please install them first.${NC}"
    echo -e "${YELLOW}Install uv: curl -LsSf https://astral.sh/uv/install.sh | sh${NC}"
    exit 1
fi

# Check CUDA
echo -e "\n${BLUE}Checking GPU...${NC}"
if nvidia-smi &> /dev/null; then
    GPU_NAME=$(nvidia-smi --query-gpu=name --format=csv,noheader | head -1)
    GPU_MEM=$(nvidia-smi --query-gpu=memory.total --format=csv,noheader | head -1)
    echo -e "  ${GREEN}✓${NC} GPU: $GPU_NAME ($GPU_MEM)"
else
    echo -e "  ${RED}✗${NC} No NVIDIA GPU detected"
    exit 1
fi

# -----------------------------------------------------------------------------
# Copy Config
# -----------------------------------------------------------------------------
echo -e "\n${BLUE}Setting up configuration...${NC}"

if [ ! -f "$BASE_DIR/configs/.env" ]; then
    cp "$BASE_DIR/configs/.env.example" "$BASE_DIR/configs/.env"
    echo -e "  ${GREEN}✓${NC} Created configs/.env from template"
else
    echo -e "  ${YELLOW}!${NC} configs/.env already exists, skipping"
fi

# -----------------------------------------------------------------------------
# Setup Python Environment (single venv with uv)
# -----------------------------------------------------------------------------
echo -e "\n${BLUE}Setting up Python environment...${NC}"

cd "$BASE_DIR"
if [ ! -d ".venv" ]; then
    uv venv
    echo -e "  ${GREEN}✓${NC} Created virtual environment"
fi

echo -e "  ${BLUE}Installing Python packages...${NC}"
uv pip install "lightrag-hku[api]" agno uvicorn openai vllm --quiet
echo -e "  ${GREEN}✓${NC} Python packages installed"

# -----------------------------------------------------------------------------
# Build Go Binary
# -----------------------------------------------------------------------------
echo -e "\n${BLUE}Building HoneyRAG TUI...${NC}"
cd "$BASE_DIR"
go mod tidy
go build -o honeyrag ./cmd/honeyrag
chmod +x honeyrag
echo -e "  ${GREEN}✓${NC} Binary built: ./honeyrag"

# -----------------------------------------------------------------------------
# Done
# -----------------------------------------------------------------------------
echo -e "\n${HONEY}"
echo "🍯 HoneyRAG is ready!"
echo "====================="
echo -e "${NC}"
echo ""
echo -e "To start the stack:"
echo -e "  ${GREEN}./honeyrag${NC}"
echo ""
echo -e "${YELLOW}First run will:${NC}"
echo -e "  - Install Ollama (if needed)"
echo -e "  - Download embedding model (~274MB)"
echo -e "  - Download LLM model (~16GB for Qwen3-8B)"
echo ""
echo -e "Subsequent runs will be fast."
echo ""
