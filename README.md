# 🍯 HoneyRAG

**A sweet, fully-integrated local RAG stack.**

One command. Full RAG pipeline. No cloud. No API keys. Just pure local AI goodness.

![License](https://img.shields.io/badge/license-MIT-blue.svg)
![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go)
![Python](https://img.shields.io/badge/Python-3.11+-3776AB?logo=python)

![HoneyRAG Screenshot](asset/screenshot-20260116-214919.png)

---

## What's in the Stack?

| Service | Purpose | Port |
|---------|---------|------|
| **Ollama** | Embedding generation (nomic-embed-text) | 11434 |
| **vLLM** | Local LLM inference | 8000 |
| **LightRAG** | RAG pipeline (chunking, indexing, retrieval) | 9621 |
| **Agno Agent** | Web UI + Agent framework | 8081 |

All services talk to each other. Upload documents to LightRAG, query through Agno. Simple.

---

## Quick Start

### Prerequisites

- **GPU**: NVIDIA with 8GB+ VRAM
- **CUDA**: 12.0+
- **Go**: 1.22+
- **uv**: Python package manager ([install](https://docs.astral.sh/uv/getting-started/installation/))

### First Time Setup

```bash
git clone https://github.com/Gsirawan/honeyrag.git
cd honeyrag
./scripts/install.sh
```

### Run

```bash
./honeyrag
```

That's it. The TUI will:
1. ✅ Sync Python dependencies (uv sync)
2. ✅ Check/install Ollama
3. ✅ Pull embedding model
4. ✅ Start vLLM (shows model config)
5. ✅ Start LightRAG
6. ✅ Start HoneyRAG Agent

First run takes longer (model downloads). After that, just `./honeyrag`.

---

## What You Get

Once running:

- **http://localhost:8081** — Agent Web UI (chat with your documents)
- **http://localhost:9621** — LightRAG UI (upload & manage documents)
- **http://localhost:8000/docs** — vLLM API docs

### Workflow

1. Open LightRAG UI (port 9621)
2. Upload your PDFs/documents
3. Open Agent UI (port 8081)
4. Ask questions about your documents
5. Get accurate, referenced answers

---

## Configuration

Edit `configs/.env` to customize:

```env
# Model (adjust for your VRAM)
VLLM_MODEL=Qwen/Qwen2.5-1.5B-Instruct

# GPU settings
VLLM_GPU_MEMORY_UTILIZATION=0.8
VLLM_MAX_MODEL_LEN=2048

# Ports
VLLM_PORT=8000
LIGHTRAG_PORT=9621
AGNO_PORT=8081
```

### Model Options by VRAM

| VRAM | Recommended Model | Context |
|------|-------------------|---------|
| 8GB | Qwen/Qwen2.5-1.5B-Instruct | 2048 |
| 16GB | Qwen/Qwen2.5-3B-Instruct | 4096 |
| 24GB | Qwen/Qwen2.5-7B-Instruct | 8192 |

---

## Logs

All service logs are in the `logs/` folder:
- `logs/ollama.log`
- `logs/vllm.log`
- `logs/lightrag.log`
- `logs/agent.log`

---

## Stopping Services

Press `q` in the TUI, or:

```bash
pkill -f "ollama serve"
pkill -f "vllm serve"
pkill -f "lightrag-server"
pkill -f "uvicorn app:app"
```

---

## Why "HoneyRAG"?

Because good things are sweet, and this stack just works. 🍯

---

## License

MIT License - do whatever you want with it.

---

*Made with 💜*
