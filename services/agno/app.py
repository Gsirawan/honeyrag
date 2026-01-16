"""
HoneyRAG - Agno Agent Service
=============================
Web UI + API for the LightRAG-connected agent.

Part of the HoneyRAG stack: A sweet, fully-integrated local RAG system.

Run: uv run uvicorn app:app --host 0.0.0.0 --port 8081
UI:  http://localhost:8081
"""

import os
from agno.agent import Agent
from agno.models.vllm import VLLM
from agno.knowledge.knowledge import Knowledge
from agno.vectordb.lightrag import LightRag
from agno.os import AgentOS

# -----------------------------------------------------------------------------
# Config (from environment or defaults)
# -----------------------------------------------------------------------------
VLLM_MODEL = os.getenv("VLLM_MODEL", "Qwen/Qwen3-8B")
LIGHTRAG_PORT = os.getenv("LIGHTRAG_PORT", "9621")
AGNO_PORT = int(os.getenv("AGNO_PORT", "8081"))

LIGHTRAG_URL = f"http://localhost:{LIGHTRAG_PORT}"

# -----------------------------------------------------------------------------
# LightRAG Vector DB
# -----------------------------------------------------------------------------
vector_db = LightRag(
    server_url=LIGHTRAG_URL,
)

# -----------------------------------------------------------------------------
# Knowledge Base
# -----------------------------------------------------------------------------
knowledge = Knowledge(
    name="HoneyRAG Knowledge",
    description="Knowledge base powered by LightRAG",
    vector_db=vector_db,
)

# -----------------------------------------------------------------------------
# Agent
# -----------------------------------------------------------------------------
agent = Agent(
    id="honeyrag-agent",
    name="HoneyRAG Agent",
    model=VLLM(
        id=VLLM_MODEL,
        api_key="not-needed",
        enable_thinking=False,
    ),
    knowledge=knowledge,
    search_knowledge=True,
    read_chat_history=True,
    markdown=True,
    add_knowledge_to_context=True,
    instructions=[
        "You are a helpful assistant with access to a knowledge base.",
        "ALWAYS search the knowledge base to find information before answering questions.",
        "Be concise and accurate in your responses.",
        "Include references to source documents when available.",
    ],
)

# -----------------------------------------------------------------------------
# AgentOS Application
# -----------------------------------------------------------------------------
agent_os = AgentOS(
    description="HoneyRAG Agent - Local RAG powered by vLLM + LightRAG",
    agents=[agent],
)

app = agent_os.get_app()

if __name__ == "__main__":
    """
    Run AgentOS server.
    
    Endpoints:
    - UI: http://localhost:8081
    - API Docs: http://localhost:8081/docs
    """
    agent_os.serve(app="app:app", host="0.0.0.0", port=AGNO_PORT, reload=True)
