# WhatsApp AI-CRM Communication Platform — System Architecture

> **Purpose**: This document is the authoritative architecture specification for an AI agent to build, improve, or implement the system described. Every section includes rationale, contracts, and implementation hints sufficient to generate production code.

---

## Table of Contents

1. [System Overview](#1-system-overview)
2. [Technology Stack](#2-technology-stack)
3. [Service Topology](#3-service-topology)
4. [Go Services — API Gateway & Communication Layer](#4-go-services)
5. [Python Services — AI Orchestration Layer](#5-python-services)
6. [Svelte Frontend — Agent Dashboard](#6-svelte-frontend)
7. [Data Layer](#7-data-layer)
8. [CRM Integration Layer](#8-crm-integration-layer)
9. [AI Agent Design — Tool Calling & RAG](#9-ai-agent-design)
10. [WhatsApp / Meta API Integration](#10-whatsapp--meta-api-integration)
11. [Event Bus Contract](#11-event-bus-contract)
12. [API Contracts (Internal)](#12-api-contracts-internal)
13. [Database Schemas](#13-database-schemas)
14. [Configuration & Environment](#14-configuration--environment)
15. [Deployment & Infrastructure](#15-deployment--infrastructure)
16. [Security Considerations](#16-security-considerations)
17. [Scalability & Failure Modes](#17-scalability--failure-modes)
18. [Implementation Roadmap](#18-implementation-roadmap)

---

## 1. System Overview

### Purpose
A multi-tenant SaaS platform enabling businesses to communicate with customers over WhatsApp (messaging + calling), with an AI agent handling customer queries autonomously by pulling context from external CRM systems, and seamlessly handing off to human agents when needed.

### Core Capabilities
- Receive and send WhatsApp messages and calls via Meta Cloud API
- AI agent autonomously resolves customer queries using LLM + tool calling
- CRM integration (Salesforce, HubSpot, Zendesk) for customer context and write-back
- RAG pipeline over business knowledge base (FAQs, policies, product docs)
- Human agent escalation with real-time dashboard (Svelte)
- Multi-tenant: each business is an isolated tenant with its own WhatsApp number and CRM config

### Architecture Pattern
- **Event-driven microservices** with async message processing
- **Go** owns all latency-sensitive communication paths (webhook ingestion, Meta API calls, WebSocket push)
- **Python** owns all AI/ML workloads (LLM orchestration, RAG, embeddings, classification)
- **Svelte** owns the agent dashboard SPA
- Services communicate via **NATS JetStream** (event bus) and direct HTTP for synchronous calls

---

## 2. Technology Stack

### Backend
| Concern | Technology | Rationale |
|---|---|---|
| API Gateway / Communication | Go 1.22+ | High concurrency, low latency, goroutines for WebSocket |
| AI Orchestration | Python 3.12+ | LLM ecosystem (LangGraph, LiteLLM, sentence-transformers) |
| HTTP framework (Go) | Gin or Chi | Lightweight, production-proven |
| HTTP framework (Python) | FastAPI | Async, Pydantic validation, OpenAPI docs |
| Event bus | NATS JetStream | Simple ops, at-least-once delivery, Go/Python SDKs |
| Task queue (Python) | ARQ (async) or Celery + Redis | Background AI processing |

### Data
| Store | Technology | Owns |
|---|---|---|
| Primary DB | PostgreSQL 16 | Messages, conversations, agents, tenants, audit log |
| Session / Cache | Redis 7 | Conversation sessions, CRM cache, rate limit counters |
| Vector DB | pgvector (start) → Qdrant (scale) | Document embeddings for RAG |
| Object storage | S3 / MinIO | WhatsApp media files (images, audio, documents) |

### AI
| Component | Technology |
|---|---|
| LLM | Anthropic Claude (claude-sonnet-4-20250514) via API |
| Embeddings | sentence-transformers (all-MiniLM-L6-v2) or OpenAI text-embedding-3-small |
| Agent framework | LangGraph or custom async loop |
| Classification | FastText or fine-tuned DistilBERT for intent/sentiment |

### Frontend
| Concern | Technology |
|---|---|
| Framework | SvelteKit |
| Real-time | WebSocket (native browser API) |
| State | Svelte stores |
| Styling | TailwindCSS |

### Infrastructure
| Concern | Technology |
|---|---|
| Containerisation | Docker + Docker Compose (dev), Kubernetes (prod) |
| Reverse proxy | Nginx or Caddy |
| CI/CD | GitHub Actions |
| Secrets | HashiCorp Vault or environment-based (.env) |

---

## 3. Service Topology

```
                          ┌─────────────────────┐
                          │   Meta Cloud API     │
                          │  (WhatsApp Business) │
                          └────────┬────────────┘
                                   │ HTTPS webhook / REST
                          ┌────────▼────────────────────────────┐
                          │         Go API Gateway              │
                          │  ┌──────────────┐ ┌─────────────┐  │
                          │  │  Messaging   │ │   Calling   │  │
                          │  │  Service     │ │   Service   │  │
                          │  └──────┬───────┘ └──────┬──────┘  │
                          │  ┌──────▼───────────────▼───────┐  │
                          │  │     NATS JetStream (Bus)      │  │
                          │  └──────────────┬───────────────┘  │
                          └─────────────────│────────────────────┘
                                            │ subscribe
                          ┌─────────────────▼────────────────────┐
                          │      Python AI Orchestration          │
                          │  ┌──────────────┐ ┌───────────────┐  │
                          │  │    Agent     │ │  RAG Pipeline │  │
                          │  │ Orchestrator │ │  (pgvector)   │  │
                          │  └──────┬───────┘ └───────────────┘  │
                          │  ┌──────▼───────┐ ┌───────────────┐  │
                          │  │ Tool Runner  │ │  Classifier   │  │
                          │  │(CRM adapters)│ │(intent/senti) │  │
                          │  └──────────────┘ └───────────────┘  │
                          └──────────────────────────────────────┘
                                    │           │
                          ┌─────────▼──┐  ┌────▼──────────┐
                          │ PostgreSQL  │  │  CRM Systems  │
                          │ Redis       │  │  (SF/HubSpot) │
                          │ pgvector    │  └───────────────┘
                          └────────────┘
                                    │
                          ┌─────────▼──────────┐
                          │  Svelte Dashboard   │
                          │  (Agent UI / WS)    │
                          └────────────────────┘
```

### Port Map (local development)
| Service | Port |
|---|---|
| Go API Gateway | 8080 |
| Python AI service | 8081 |
| NATS | 4222 |
| PostgreSQL | 5432 |
| Redis | 6379 |
| Svelte dev server | 5173 |
| Qdrant (if used) | 6333 |

---

## 4. Go Services

### 4.1 Project Structure

```
/services/go-gateway/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── config/
│   │   └── config.go
│   ├── messaging/
│   │   ├── handler.go        # HTTP handlers (webhook, send)
│   │   ├── service.go        # Business logic
│   │   ├── meta_client.go    # Meta API HTTP client
│   │   └── types.go          # Request/response types
│   ├── calling/
│   │   ├── handler.go
│   │   ├── service.go
│   │   └── webrtc.go
│   ├── auth/
│   │   ├── middleware.go     # JWT validation
│   │   └── service.go
│   ├── eventbus/
│   │   ├── publisher.go      # NATS publisher
│   │   └── subscriber.go
│   ├── crm/
│   │   ├── proxy.go          # Unified CRM proxy (rate limiting, retry)
│   │   └── cache.go          # Redis CRM cache
│   ├── ws/
│   │   └── hub.go            # WebSocket hub for dashboard push
│   └── store/
│       ├── postgres.go
│       └── redis.go
├── go.mod
└── go.sum
```

### 4.2 Messaging Service

#### Webhook Ingestion (critical path — must return <5s)

```go
// internal/messaging/handler.go

// POST /webhook/whatsapp
func (h *Handler) HandleWebhook(c *gin.Context) {
    // 1. Verify HMAC-SHA256 signature from X-Hub-Signature-256 header
    if !h.verifySignature(c.Request) {
        c.AbortWithStatus(http.StatusUnauthorized)
        return
    }

    var payload WhatsAppWebhookPayload
    if err := c.ShouldBindJSON(&payload); err != nil {
        c.AbortWithStatus(http.StatusBadRequest)
        return
    }

    // 2. Ack immediately — never make Meta wait
    c.Status(http.StatusOK)

    // 3. Persist raw event to Postgres (audit trail)
    // 4. Publish to NATS JetStream (non-blocking)
    go h.processWebhookAsync(payload)
}

func (h *Handler) processWebhookAsync(payload WhatsAppWebhookPayload) {
    for _, entry := range payload.Entry {
        for _, change := range entry.Changes {
            for _, msg := range change.Value.Messages {
                event := MessageReceivedEvent{
                    ID:          msg.ID,
                    TenantID:    h.resolveTenant(change.Value.Metadata.PhoneNumberID),
                    From:        msg.From,
                    Type:        msg.Type,
                    Content:     extractContent(msg),
                    Timestamp:   time.Unix(msg.Timestamp, 0),
                }
                h.publisher.Publish("message.received", event)
            }
        }
    }
}
```

#### Send Message

```go
// internal/messaging/meta_client.go

type MetaClient struct {
    httpClient *http.Client
    baseURL    string
    token      string   // Per-tenant access token
}

func (c *MetaClient) SendTextMessage(phoneNumberID, to, body string) error {
    payload := map[string]interface{}{
        "messaging_product": "whatsapp",
        "recipient_type":    "individual",
        "to":                to,
        "type":              "text",
        "text":              map[string]string{"body": body},
    }
    // POST https://graph.facebook.com/v19.0/{phoneNumberID}/messages
    return c.post(fmt.Sprintf("/v19.0/%s/messages", phoneNumberID), payload)
}

func (c *MetaClient) SendTemplate(phoneNumberID, to, templateName string, components []TemplateComponent) error {
    // For outbound notifications — requires pre-approved template
}

func (c *MetaClient) MarkRead(phoneNumberID, messageID string) error {
    // POST status: read
}
```

### 4.3 Calling Service

```go
// internal/calling/service.go

// WhatsApp Business Calling uses signaling events via webhook
// Real audio goes through Meta's infrastructure (not raw WebRTC on your side)
// You handle: call state machine, routing, recording consent, CRM logging

type CallSession struct {
    ID          string
    TenantID    string
    CustomerPhone string
    State       CallState  // ringing | active | on_hold | ended
    StartedAt   time.Time
    AgentID     *string    // nil = AI handling
}

type CallState string
const (
    CallStateRinging CallState = "ringing"
    CallStateActive  CallState = "active"
    CallStateOnHold  CallState = "on_hold"
    CallStateEnded   CallState = "ended"
)

func (s *CallingService) HandleCallEvent(event WhatsAppCallEvent) {
    switch event.Type {
    case "call.initiated":
        s.createSession(event)
        s.publisher.Publish("call.started", CallStartedEvent{...})
    case "call.ended":
        s.finalizeSession(event)
        s.publisher.Publish("call.ended", CallEndedEvent{...})
    }
}
```

### 4.4 WebSocket Hub (Dashboard Push)

```go
// internal/ws/hub.go

type Hub struct {
    clients    map[string]*Client  // agentID → client
    broadcast  chan []byte
    register   chan *Client
    unregister chan *Client
    mu         sync.RWMutex
}

// Push real-time events to specific agent or all agents of a tenant
func (h *Hub) PushToTenant(tenantID string, event interface{}) {
    data, _ := json.Marshal(event)
    h.mu.RLock()
    defer h.mu.RUnlock()
    for _, client := range h.clients {
        if client.TenantID == tenantID {
            select {
            case client.send <- data:
            default:
                // Client too slow — drop or buffer
            }
        }
    }
}
```

---

## 5. Python Services

### 5.1 Project Structure

```
/services/python-ai/
├── main.py                   # FastAPI app entry point
├── api/
│   └── routes.py             # Internal REST endpoints
├── agent/
│   ├── orchestrator.py       # Main agent loop
│   ├── prompt_builder.py     # System prompt construction
│   ├── tool_runner.py        # Tool execution engine
│   └── tools/
│       ├── base.py           # Tool base class / interface
│       ├── crm_tools.py      # get_customer, create_ticket, etc.
│       ├── knowledge_tools.py # search_knowledge_base
│       └── messaging_tools.py # send_message (calls Go service)
├── rag/
│   ├── pipeline.py           # Embed → retrieve → rerank
│   ├── embedder.py           # Sentence transformer wrapper
│   └── retriever.py          # pgvector / Qdrant queries
├── classifier/
│   └── intent.py             # Intent + sentiment classification
├── crm/
│   ├── adapter.py            # Abstract CRM adapter
│   ├── salesforce.py
│   ├── hubspot.py
│   └── zendesk.py
├── bus/
│   └── subscriber.py         # NATS subscriber + dispatch
├── store/
│   ├── postgres.py
│   └── redis.py
├── config.py
└── requirements.txt
```

### 5.2 Agent Orchestrator

```python
# agent/orchestrator.py

import anthropic
from typing import AsyncIterator

client = anthropic.AsyncAnthropic()

class AgentOrchestrator:
    MAX_TOOL_ITERATIONS = 8

    async def handle_message(self, event: MessageReceivedEvent) -> AgentResult:
        # 1. Load conversation history
        history = await self.store.get_conversation_history(
            event.tenant_id, event.customer_phone, limit=20
        )

        # 2. Classify intent and sentiment (fast, local model)
        classification = await self.classifier.classify(event.content)

        # 3. Check for immediate escalation triggers
        if classification.requires_immediate_escalation:
            return await self.escalate(event, reason=classification.escalation_reason)

        # 4. Fetch customer context (CRM pre-fetch + RAG)
        customer = await self.crm_cache.get_or_fetch(event.tenant_id, event.customer_phone)
        rag_context = await self.rag.retrieve(event.content, tenant_id=event.tenant_id, top_k=5)

        # 5. Build messages array
        system_prompt = self.prompt_builder.build(
            tenant_config=await self.get_tenant_config(event.tenant_id),
            customer=customer,
            rag_context=rag_context,
        )
        messages = self._build_messages(history, event.content)

        # 6. Agentic loop
        result = await self._run_agent_loop(system_prompt, messages, event)

        # 7. Persist and return
        await self.store.save_message(event, result.response)
        return result

    async def _run_agent_loop(self, system_prompt, messages, event) -> AgentResult:
        tools = self.tool_runner.get_tool_definitions(event.tenant_id)

        for iteration in range(self.MAX_TOOL_ITERATIONS):
            response = await client.messages.create(
                model="claude-sonnet-4-20250514",
                max_tokens=1024,
                system=system_prompt,
                tools=tools,
                messages=messages,
            )

            if response.stop_reason == "end_turn":
                # Final text response
                text = next(b.text for b in response.content if b.type == "text")
                return AgentResult(response=text, escalate=False, iterations=iteration+1)

            if response.stop_reason == "tool_use":
                # Execute all tool calls
                tool_results = []
                for block in response.content:
                    if block.type == "tool_use":
                        result = await self.tool_runner.execute(
                            tool_name=block.name,
                            tool_input=block.input,
                            tenant_id=event.tenant_id,
                        )
                        tool_results.append({
                            "type": "tool_result",
                            "tool_use_id": block.id,
                            "content": str(result),
                        })

                # Append assistant turn + tool results, continue loop
                messages.append({"role": "assistant", "content": response.content})
                messages.append({"role": "user", "content": tool_results})

        # Exceeded max iterations — escalate
        return await self.escalate(event, reason="max_iterations_exceeded")
```

### 5.3 Prompt Builder

```python
# agent/prompt_builder.py

SYSTEM_PROMPT_TEMPLATE = """
You are {business_name}'s customer support AI assistant on WhatsApp.
Your role is to help customers resolve their issues quickly and accurately.

## Customer context
- Name: {customer_name}
- Phone: {customer_phone}
- Account status: {account_status}
- Customer since: {customer_since}
- Open tickets: {open_tickets_summary}
- Recent orders: {recent_orders_summary}
- Last interaction: {last_interaction_summary}

## Relevant knowledge base excerpts
{rag_context}

## Guidelines
- Be concise — WhatsApp messages should be short and conversational.
- Use the tools available to look up information before answering.
- If you cannot resolve the issue with available tools, say so and offer to escalate.
- Never make up information about orders, tickets, or account data.
- If the customer is angry or the issue is complex, recommend escalation.
- Always confirm actions (e.g. ticket creation) with the customer.

## Escalation trigger
If you determine human intervention is needed, call the `escalate_to_human` tool.
"""

class PromptBuilder:
    def build(self, tenant_config, customer, rag_context) -> str:
        return SYSTEM_PROMPT_TEMPLATE.format(
            business_name=tenant_config.business_name,
            customer_name=customer.name if customer else "Unknown",
            customer_phone=customer.phone if customer else "Unknown",
            account_status=customer.account_status if customer else "Unknown",
            customer_since=customer.created_at.strftime("%Y-%m-%d") if customer else "N/A",
            open_tickets_summary=self._format_tickets(customer.open_tickets if customer else []),
            recent_orders_summary=self._format_orders(customer.recent_orders if customer else []),
            last_interaction_summary=customer.last_interaction_summary if customer else "No history",
            rag_context="\n\n".join(rag_context) if rag_context else "No relevant documents found.",
        )
```

### 5.4 Tool Definitions

```python
# agent/tools/crm_tools.py

TOOL_DEFINITIONS = [
    {
        "name": "get_customer_details",
        "description": "Get full customer profile including account status, tier, and contact info.",
        "input_schema": {
            "type": "object",
            "properties": {
                "phone": {"type": "string", "description": "Customer phone in E.164 format"}
            },
            "required": ["phone"]
        }
    },
    {
        "name": "get_open_tickets",
        "description": "List all open support tickets for a customer.",
        "input_schema": {
            "type": "object",
            "properties": {
                "customer_id": {"type": "string"}
            },
            "required": ["customer_id"]
        }
    },
    {
        "name": "create_support_ticket",
        "description": "Create a new support ticket. Use when the customer reports a new issue.",
        "input_schema": {
            "type": "object",
            "properties": {
                "customer_id": {"type": "string"},
                "subject": {"type": "string"},
                "description": {"type": "string"},
                "priority": {"type": "string", "enum": ["low", "medium", "high", "urgent"]}
            },
            "required": ["customer_id", "subject", "description", "priority"]
        }
    },
    {
        "name": "get_order_status",
        "description": "Get the current status of a customer order.",
        "input_schema": {
            "type": "object",
            "properties": {
                "order_id": {"type": "string"},
                "customer_id": {"type": "string"}
            },
            "required": ["order_id", "customer_id"]
        }
    },
    {
        "name": "search_knowledge_base",
        "description": "Search the business knowledge base for policies, FAQs, and product info.",
        "input_schema": {
            "type": "object",
            "properties": {
                "query": {"type": "string"}
            },
            "required": ["query"]
        }
    },
    {
        "name": "escalate_to_human",
        "description": "Hand off the conversation to a human agent. Use when the issue is too complex or the customer requests it.",
        "input_schema": {
            "type": "object",
            "properties": {
                "reason": {"type": "string"},
                "priority": {"type": "string", "enum": ["normal", "urgent"]}
            },
            "required": ["reason", "priority"]
        }
    }
]
```

### 5.5 RAG Pipeline

```python
# rag/pipeline.py

from sentence_transformers import SentenceTransformer
import asyncpg

class RAGPipeline:
    def __init__(self):
        self.embedder = SentenceTransformer("all-MiniLM-L6-v2")

    async def retrieve(self, query: str, tenant_id: str, top_k: int = 5) -> list[str]:
        query_embedding = self.embedder.encode(query).tolist()

        # pgvector cosine similarity search
        rows = await self.db.fetch(
            """
            SELECT content, 1 - (embedding <=> $1::vector) AS similarity
            FROM knowledge_chunks
            WHERE tenant_id = $2
            ORDER BY embedding <=> $1::vector
            LIMIT $3
            """,
            query_embedding, tenant_id, top_k
        )
        return [row["content"] for row in rows if row["similarity"] > 0.7]

    async def index_document(self, tenant_id: str, content: str, metadata: dict):
        # Chunk → embed → store
        chunks = self._chunk(content, size=512, overlap=64)
        for chunk in chunks:
            embedding = self.embedder.encode(chunk).tolist()
            await self.db.execute(
                """
                INSERT INTO knowledge_chunks (tenant_id, content, embedding, metadata)
                VALUES ($1, $2, $3::vector, $4)
                """,
                tenant_id, chunk, embedding, json.dumps(metadata)
            )

    def _chunk(self, text: str, size: int, overlap: int) -> list[str]:
        words = text.split()
        chunks = []
        for i in range(0, len(words), size - overlap):
            chunks.append(" ".join(words[i:i+size]))
        return chunks
```

### 5.6 CRM Adapter Interface

```python
# crm/adapter.py

from abc import ABC, abstractmethod
from dataclasses import dataclass

@dataclass
class Customer:
    id: str
    name: str
    phone: str
    email: str
    account_status: str
    tier: str
    created_at: datetime
    open_tickets: list
    recent_orders: list
    last_interaction_summary: str | None

@dataclass
class Ticket:
    id: str
    subject: str
    status: str
    priority: str
    created_at: datetime
    updated_at: datetime

class CRMAdapter(ABC):
    @abstractmethod
    async def get_customer_by_phone(self, phone: str) -> Customer | None: ...

    @abstractmethod
    async def get_open_tickets(self, customer_id: str) -> list[Ticket]: ...

    @abstractmethod
    async def create_ticket(self, customer_id: str, subject: str, description: str, priority: str) -> Ticket: ...

    @abstractmethod
    async def update_ticket(self, ticket_id: str, updates: dict) -> Ticket: ...

    @abstractmethod
    async def get_order(self, order_id: str, customer_id: str) -> dict | None: ...
```

```python
# crm/hubspot.py

import httpx

class HubSpotAdapter(CRMAdapter):
    BASE_URL = "https://api.hubapi.com"

    def __init__(self, api_key: str):
        self.client = httpx.AsyncClient(
            headers={"Authorization": f"Bearer {api_key}"},
            base_url=self.BASE_URL,
            timeout=10.0,
        )

    async def get_customer_by_phone(self, phone: str) -> Customer | None:
        response = await self.client.post("/crm/v3/objects/contacts/search", json={
            "filterGroups": [{
                "filters": [{"propertyName": "phone", "operator": "EQ", "value": phone}]
            }],
            "properties": ["firstname", "lastname", "phone", "email", "hs_lead_status", "createdate"]
        })
        results = response.json().get("results", [])
        if not results:
            return None
        return self._map_contact(results[0])
```

### 5.7 NATS Subscriber

```python
# bus/subscriber.py

import nats
from nats.js.api import ConsumerConfig

async def start_subscriber(orchestrator: AgentOrchestrator):
    nc = await nats.connect("nats://localhost:4222")
    js = nc.jetstream()

    # Durable consumer — survives restarts
    await js.subscribe(
        "message.received",
        durable="ai-agent",
        cb=lambda msg: asyncio.create_task(handle_message(msg, orchestrator)),
        config=ConsumerConfig(ack_wait=30),  # 30s to process before redeliver
    )

async def handle_message(msg, orchestrator):
    try:
        event = MessageReceivedEvent(**json.loads(msg.data))
        await orchestrator.handle_message(event)
        await msg.ack()
    except Exception as e:
        logger.error(f"Agent error: {e}")
        await msg.nak(delay=5)  # Retry after 5s
```

---

## 6. Svelte Frontend

### 6.1 Project Structure

```
/frontend/
├── src/
│   ├── lib/
│   │   ├── stores/
│   │   │   ├── conversations.ts   # Active conversation store
│   │   │   ├── agent.ts           # Current agent state
│   │   │   └── ws.ts              # WebSocket store
│   │   ├── components/
│   │   │   ├── ConversationList.svelte
│   │   │   ├── MessageThread.svelte
│   │   │   ├── CustomerSidebar.svelte  # CRM context panel
│   │   │   ├── AIReasoningTrace.svelte # Show AI tool calls
│   │   │   └── EscalationBanner.svelte
│   │   └── api/
│   │       └── client.ts
│   ├── routes/
│   │   ├── +layout.svelte
│   │   ├── inbox/+page.svelte
│   │   ├── conversation/[id]/+page.svelte
│   │   └── analytics/+page.svelte
│   └── app.html
└── svelte.config.js
```

### 6.2 WebSocket Store

```typescript
// lib/stores/ws.ts

import { writable } from 'svelte/store';

export type WSEvent =
  | { type: 'message.received'; data: Message }
  | { type: 'conversation.escalated'; data: Conversation }
  | { type: 'ai.tool_call'; data: ToolCallTrace }
  | { type: 'agent.typing'; data: { conversationId: string } };

export function createWSStore(agentToken: string) {
  const { subscribe, set, update } = writable<WSEvent | null>(null);
  let ws: WebSocket;

  function connect() {
    ws = new WebSocket(`wss://api.yourdomain.com/ws?token=${agentToken}`);
    ws.onmessage = (e) => set(JSON.parse(e.data));
    ws.onclose = () => setTimeout(connect, 3000); // auto-reconnect
  }

  connect();
  return { subscribe };
}
```

### 6.3 Key UI Views

**Inbox view**: Conversation list sorted by urgency (escalated → AI-active → resolved). Each row shows customer name, last message preview, AI confidence score, and a badge indicating AI vs human handling.

**Conversation detail view**:
- Message thread with AI and customer bubbles
- Collapsible "AI reasoning" panel showing which tools were called and what data was retrieved
- Right sidebar with CRM data: customer tier, open tickets, recent orders
- "Take over" button for human escalation
- Quick reply templates

**Analytics view**: Resolution rate (AI vs human), average handle time, escalation rate, most common intents, CSAT scores.

---

## 7. Data Layer

### 7.1 PostgreSQL Schema

```sql
-- Tenants (businesses using the platform)
CREATE TABLE tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    whatsapp_phone_number_id TEXT NOT NULL UNIQUE,
    whatsapp_access_token TEXT NOT NULL,  -- encrypted at rest
    crm_type TEXT,       -- 'salesforce' | 'hubspot' | 'zendesk' | null
    crm_config JSONB,    -- encrypted credentials
    ai_config JSONB,     -- LLM settings, escalation thresholds
    created_at TIMESTAMPTZ DEFAULT now()
);

-- Agents (human support staff)
CREATE TABLE agents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID REFERENCES tenants(id),
    email TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    role TEXT DEFAULT 'agent',  -- 'agent' | 'admin'
    is_online BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ DEFAULT now()
);

-- Customers (WhatsApp end-users)
CREATE TABLE customers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID REFERENCES tenants(id),
    phone TEXT NOT NULL,
    name TEXT,
    crm_id TEXT,    -- ID in the external CRM
    created_at TIMESTAMPTZ DEFAULT now(),
    UNIQUE(tenant_id, phone)
);

-- Conversations
CREATE TABLE conversations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID REFERENCES tenants(id),
    customer_id UUID REFERENCES customers(id),
    status TEXT DEFAULT 'ai_active',  -- 'ai_active' | 'human_active' | 'resolved' | 'escalated'
    assigned_agent_id UUID REFERENCES agents(id),
    ai_resolution_attempted BOOLEAN DEFAULT false,
    escalation_reason TEXT,
    resolved_by TEXT,   -- 'ai' | 'human'
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);

-- Messages
CREATE TABLE messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID REFERENCES conversations(id),
    whatsapp_message_id TEXT UNIQUE,  -- Meta's message ID
    direction TEXT NOT NULL,          -- 'inbound' | 'outbound'
    sender_type TEXT NOT NULL,        -- 'customer' | 'ai' | 'human_agent'
    content TEXT,
    content_type TEXT DEFAULT 'text', -- 'text' | 'image' | 'audio' | 'document'
    media_url TEXT,
    ai_tool_calls JSONB,   -- Snapshot of tool calls made for this response
    ai_confidence FLOAT,
    status TEXT DEFAULT 'sent',       -- 'sent' | 'delivered' | 'read' | 'failed'
    created_at TIMESTAMPTZ DEFAULT now()
);

-- AI reasoning traces (for dashboard transparency)
CREATE TABLE ai_traces (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id UUID REFERENCES messages(id),
    conversation_id UUID REFERENCES conversations(id),
    tool_calls JSONB,          -- Array of {tool, input, output, duration_ms}
    rag_chunks JSONB,          -- Retrieved chunks
    intent TEXT,
    sentiment TEXT,
    confidence FLOAT,
    total_duration_ms INTEGER,
    created_at TIMESTAMPTZ DEFAULT now()
);

-- Knowledge base chunks (for RAG)
CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE knowledge_chunks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID REFERENCES tenants(id),
    source_document TEXT,
    content TEXT NOT NULL,
    embedding vector(384),   -- all-MiniLM-L6-v2 dimensions
    metadata JSONB,
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX ON knowledge_chunks USING ivfflat (embedding vector_cosine_ops)
    WITH (lists = 100);  -- tune based on dataset size

-- Indexes
CREATE INDEX idx_messages_conversation ON messages(conversation_id, created_at DESC);
CREATE INDEX idx_conversations_tenant_status ON conversations(tenant_id, status);
CREATE INDEX idx_customers_tenant_phone ON customers(tenant_id, phone);
```

### 7.2 Redis Key Schema

```
# Conversation session (TTL: 24h)
session:{tenant_id}:{customer_phone}  →  JSON { conversation_id, last_seen, ai_state }

# CRM customer cache (TTL: 5 min)
crm:customer:{tenant_id}:{phone}  →  JSON Customer object

# CRM tickets cache (TTL: 2 min)
crm:tickets:{tenant_id}:{customer_id}  →  JSON array of tickets

# Rate limit: Meta API sends per tenant (sliding window)
rate:meta:send:{tenant_id}  →  counter (TTL: 1s)

# Agent presence
agent:online:{tenant_id}  →  SET of agent IDs

# Conversation lock (prevent double-processing)
lock:conversation:{conversation_id}  →  "1" (TTL: 30s)
```

---

## 8. CRM Integration Layer

### 8.1 Adapter Registry

```python
# crm/registry.py

from .salesforce import SalesforceAdapter
from .hubspot import HubSpotAdapter
from .zendesk import ZendeskAdapter

ADAPTERS = {
    "salesforce": SalesforceAdapter,
    "hubspot": HubSpotAdapter,
    "zendesk": ZendeskAdapter,
}

def get_adapter(tenant_config: dict) -> CRMAdapter:
    crm_type = tenant_config["crm_type"]
    crm_config = tenant_config["crm_config"]  # decrypted credentials
    adapter_class = ADAPTERS.get(crm_type)
    if not adapter_class:
        raise ValueError(f"Unsupported CRM: {crm_type}")
    return adapter_class(**crm_config)
```

### 8.2 CRM Proxy in Go (Rate Limiting & Audit)

All CRM writes from Python go through a Go proxy endpoint. This centralises rate limiting, retry logic, and provides an audit log.

```go
// internal/crm/proxy.go

// POST /internal/crm/action
// Called by Python tool runner for write operations
func (h *CRMProxyHandler) HandleAction(c *gin.Context) {
    var req CRMActionRequest
    c.ShouldBindJSON(&req)

    // Rate limit check (per tenant, per CRM)
    if !h.rateLimiter.Allow(req.TenantID) {
        c.JSON(429, gin.H{"error": "CRM rate limit exceeded"})
        return
    }

    // Execute with retry
    result, err := h.executeWithRetry(req, maxRetries=3)

    // Audit log
    h.store.LogCRMAction(req, result, err)

    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    c.JSON(200, result)
}
```

### 8.3 CRM Data Sync Strategy

| Operation | Strategy |
|---|---|
| Customer lookup on message | Redis cache first (TTL 5min), then CRM API |
| Ticket read | Redis cache (TTL 2min) |
| Ticket create/update | Direct CRM API via Go proxy, invalidate cache |
| Order read | Redis cache (TTL 1min) |
| Bulk knowledge import | Background job, no cache |

---

## 9. AI Agent Design — Tool Calling & RAG

### 9.1 Agent Loop State Machine

```
RECEIVED_MESSAGE
      │
      ▼
CLASSIFY_INTENT ──── requires_immediate_escalation ──→ ESCALATE
      │
      ▼
LOAD_CONTEXT (Redis session + CRM pre-fetch + RAG)
      │
      ▼
LLM_CALL (system_prompt + tools + history)
      │
      ├── stop_reason = "end_turn" ──→ SEND_RESPONSE ──→ DONE
      │
      └── stop_reason = "tool_use"
              │
              ▼
         EXECUTE_TOOLS
              │
              ├── tool = "escalate_to_human" ──→ ESCALATE
              │
              └── other tools ──→ append results ──→ LLM_CALL (loop)
                                                    (max 8 iterations)
```

### 9.2 Context Window Management

For long conversations, use a sliding window + summarisation strategy:

```python
async def build_message_history(self, conversation_id: str, new_message: str) -> list[dict]:
    messages = await self.store.get_messages(conversation_id, limit=50)

    if len(messages) > 20:
        # Summarise older messages
        older = messages[:-10]
        recent = messages[-10:]
        summary = await self._summarise(older)
        return [
            {"role": "user", "content": f"[Earlier conversation summary: {summary}]"},
            {"role": "assistant", "content": "Understood, I have context from earlier."},
            *self._format_messages(recent),
            {"role": "user", "content": new_message},
        ]
    else:
        return [
            *self._format_messages(messages),
            {"role": "user", "content": new_message},
        ]
```

### 9.3 Escalation Logic

Escalation is triggered by:
1. LLM calling the `escalate_to_human` tool explicitly
2. Classifier detecting urgency score > 0.85 or extreme negative sentiment
3. Agent loop exceeding max iterations (8)
4. Customer explicitly requesting a human (keyword patterns)
5. Issue type not in AI's configured scope (per tenant config)

On escalation:
1. Python publishes `conversation.escalated` event to NATS
2. Go pushes WebSocket event to dashboard
3. Available human agent is assigned (round-robin or priority-based)
4. AI sends acknowledgement message to customer: "I'm connecting you with a specialist..."

---

## 10. WhatsApp / Meta API Integration

### 10.1 Supported Message Types

| Type | Inbound | Outbound |
|---|---|---|
| Text | ✅ | ✅ |
| Image | ✅ | ✅ |
| Audio | ✅ | ✅ |
| Document | ✅ | ✅ |
| Template | ❌ | ✅ (pre-approved) |
| Interactive (buttons) | ✅ | ✅ |
| Location | ✅ | ❌ |
| Reaction | ✅ (log only) | ❌ |

### 10.2 Webhook Verification

```go
// On initial webhook setup, Meta sends a GET with challenge
// GET /webhook/whatsapp?hub.mode=subscribe&hub.verify_token=TOKEN&hub.challenge=CHALLENGE
func (h *Handler) VerifyWebhook(c *gin.Context) {
    mode := c.Query("hub.mode")
    token := c.Query("hub.verify_token")
    challenge := c.Query("hub.challenge")

    if mode == "subscribe" && token == h.config.WebhookVerifyToken {
        c.String(200, challenge)
        return
    }
    c.AbortWithStatus(403)
}

// On all POST webhooks, verify HMAC
func (h *Handler) verifySignature(r *http.Request) bool {
    signature := r.Header.Get("X-Hub-Signature-256")
    body, _ := io.ReadAll(r.Body)
    r.Body = io.NopCloser(bytes.NewBuffer(body))

    mac := hmac.New(sha256.New, []byte(h.config.AppSecret))
    mac.Write(body)
    expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))
    return hmac.Equal([]byte(signature), []byte(expected))
}
```

### 10.3 Media Handling

```go
// WhatsApp sends a media_id, not a direct URL
// Must: 1) retrieve URL from Meta, 2) download, 3) store in S3, 4) reference internally

func (h *Handler) resolveMedia(mediaID string) (string, error) {
    // GET https://graph.facebook.com/v19.0/{media-id}
    metaURL := h.metaClient.GetMediaURL(mediaID)

    // Download from Meta (requires auth header)
    data := h.metaClient.DownloadMedia(metaURL)

    // Upload to S3
    s3Key := fmt.Sprintf("media/%s/%s", h.tenantID, mediaID)
    return h.s3.Upload(s3Key, data)
}
```

### 10.4 Rate Limits (Meta API)

| Limit | Value |
|---|---|
| Messages per second per phone | 80 |
| Template messages per day per phone | Varies by quality rating |
| Media upload size | 16MB (images), 100MB (video) |

Implement a token bucket rate limiter in Go (Redis-backed for multi-instance).

---

## 11. Event Bus Contract

All events published to NATS JetStream follow this envelope:

```json
{
    "event_id": "uuid-v4",
    "event_type": "message.received",
    "tenant_id": "uuid",
    "occurred_at": "2025-01-01T00:00:00Z",
    "payload": { ... }
}
```

### Event Catalogue

| Subject | Publisher | Subscriber | Payload |
|---|---|---|---|
| `message.received` | Go Messaging | Python AI | `{ customer_phone, content, type, whatsapp_msg_id }` |
| `message.sent` | Go Messaging | Python AI (log) | `{ customer_phone, content, direction }` |
| `call.started` | Go Calling | Python AI | `{ customer_phone, call_id }` |
| `call.ended` | Go Calling | Python AI | `{ call_id, duration_s }` |
| `ai.response_ready` | Python AI | Go Messaging | `{ customer_phone, response_text, conversation_id }` |
| `conversation.escalated` | Python AI | Go WS Hub | `{ conversation_id, reason, priority }` |
| `agent.typing` | Go WS Hub | (dashboard only) | `{ conversation_id }` |

### NATS JetStream Configuration

```go
// Create stream on startup
js.AddStream(&nats.StreamConfig{
    Name:      "PLATFORM",
    Subjects:  []string{"message.*", "call.*", "ai.*", "conversation.*", "agent.*"},
    Retention: nats.LimitsPolicy,
    MaxAge:    7 * 24 * time.Hour,
    Storage:   nats.FileStorage,
    Replicas:  1,  // increase for HA
})
```

---

## 12. API Contracts (Internal)

### Go → Python (synchronous, for knowledge base indexing)

```
POST /internal/knowledge/index
Authorization: Bearer {internal_service_token}

{
    "tenant_id": "uuid",
    "document": "base64-encoded text",
    "source": "filename.pdf",
    "metadata": { "type": "faq", "version": "2024-01" }
}

Response 202 Accepted
{ "job_id": "uuid" }
```

### Python → Go (send message back to customer)

```
POST /internal/messaging/send
Authorization: Bearer {internal_service_token}

{
    "tenant_id": "uuid",
    "to": "+254700000000",
    "type": "text",
    "content": "Your order #1234 is on its way!"
}

Response 200 OK
{ "whatsapp_message_id": "wamid.xxx" }
```

### External REST API (for Svelte dashboard)

```
GET  /api/v1/conversations?status=escalated&limit=20
GET  /api/v1/conversations/{id}/messages
GET  /api/v1/conversations/{id}/ai-trace
POST /api/v1/conversations/{id}/take-over
POST /api/v1/conversations/{id}/messages   (human agent sends message)
GET  /api/v1/analytics/summary?from=2025-01-01&to=2025-01-31
GET  /api/v1/knowledge                     (list indexed documents)
POST /api/v1/knowledge                     (upload new document)
DELETE /api/v1/knowledge/{id}
```

All external endpoints require JWT bearer token (agent login).

---

## 13. Database Schemas

> See Section 7.1 for full SQL DDL. Additional notes below.

### Conversation status transitions

```
ai_active ──────────────────────────────────→ resolved (by AI)
    │
    └──→ escalated ──→ human_active ──→ resolved (by human)
```

### Multi-tenancy enforcement

Every query must include `tenant_id` in the WHERE clause. Use Row Level Security (RLS) in PostgreSQL as a safety net:

```sql
ALTER TABLE conversations ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON conversations
    USING (tenant_id = current_setting('app.tenant_id')::uuid);
```

Set `app.tenant_id` at the connection level in Go before querying.

---

## 14. Configuration & Environment

### Go Gateway `.env`

```env
# Server
PORT=8080
ENV=development

# Database
DATABASE_URL=postgres://user:pass@localhost:5432/platform

# Redis
REDIS_URL=redis://localhost:6379

# NATS
NATS_URL=nats://localhost:4222

# Meta / WhatsApp
META_APP_SECRET=your_app_secret
META_WEBHOOK_VERIFY_TOKEN=your_verify_token
# Per-tenant tokens are stored in DB (encrypted), not env

# Internal service auth
INTERNAL_SERVICE_TOKEN=strong_random_secret

# S3
S3_BUCKET=whatsapp-media
S3_REGION=us-east-1
AWS_ACCESS_KEY_ID=...
AWS_SECRET_ACCESS_KEY=...

# JWT
JWT_SECRET=strong_random_secret
JWT_EXPIRY_HOURS=8
```

### Python AI Service `.env`

```env
# Server
PORT=8081
ENV=development

# Database
DATABASE_URL=postgresql+asyncpg://user:pass@localhost:5432/platform

# Redis
REDIS_URL=redis://localhost:6379

# NATS
NATS_URL=nats://localhost:4222

# LLM
ANTHROPIC_API_KEY=sk-ant-...
LLM_MODEL=claude-sonnet-4-20250514
LLM_MAX_TOKENS=1024

# Embeddings
EMBEDDING_MODEL=all-MiniLM-L6-v2

# Internal
INTERNAL_SERVICE_TOKEN=strong_random_secret
GO_GATEWAY_URL=http://localhost:8080
```

---

## 15. Deployment & Infrastructure

### Docker Compose (development)

```yaml
version: "3.9"
services:
  postgres:
    image: pgvector/pgvector:pg16
    environment:
      POSTGRES_DB: platform
      POSTGRES_USER: user
      POSTGRES_PASSWORD: pass
    ports: ["5432:5432"]
    volumes: [postgres_data:/var/lib/postgresql/data]

  redis:
    image: redis:7-alpine
    ports: ["6379:6379"]

  nats:
    image: nats:2.10-alpine
    command: -js -sd /data
    ports: ["4222:4222", "8222:8222"]
    volumes: [nats_data:/data]

  go-gateway:
    build: ./services/go-gateway
    ports: ["8080:8080"]
    env_file: ./services/go-gateway/.env
    depends_on: [postgres, redis, nats]

  python-ai:
    build: ./services/python-ai
    ports: ["8081:8081"]
    env_file: ./services/python-ai/.env
    depends_on: [postgres, redis, nats]

  frontend:
    build: ./frontend
    ports: ["5173:5173"]
    depends_on: [go-gateway]

volumes:
  postgres_data:
  nats_data:
```

### Production Kubernetes (outline)

```
Namespaces:
  platform-prod
    go-gateway        (Deployment, 3 replicas, HPA on CPU)
    python-ai         (Deployment, 2 replicas, HPA on CPU)
    frontend          (Deployment, 2 replicas)
    nats              (StatefulSet, 3 replicas, JetStream)
  platform-data
    postgres          (StatefulSet + PVC, or managed RDS)
    redis             (StatefulSet, or managed ElastiCache)
```

Services exposed:
- `go-gateway` → LoadBalancer (public, handles Meta webhook + dashboard API)
- `python-ai` → ClusterIP (internal only)
- `frontend` → LoadBalancer or CDN (static build)

Meta webhook URL must be HTTPS with a valid TLS cert. Use cert-manager with Let's Encrypt.

---

## 16. Security Considerations

| Concern | Mitigation |
|---|---|
| Webhook authenticity | HMAC-SHA256 signature verification on every Meta webhook |
| CRM credentials | Encrypted at rest in Postgres (pgcrypto), decrypted at runtime |
| Internal service auth | Shared secret token in `Authorization: Bearer` header |
| Agent authentication | JWT with short expiry (8h), refresh token rotation |
| Multi-tenant isolation | `tenant_id` on all queries + PostgreSQL RLS |
| LLM prompt injection | Sanitise customer input before injecting into prompts; never trust customer-provided structured data directly |
| PII in logs | Mask phone numbers and customer names in all log output |
| Rate limiting | Token bucket on Meta API sends; NATS per-consumer flow control |
| Media security | Signed S3 URLs with short TTL for media access |

---

## 17. Scalability & Failure Modes

### Critical Failure Modes & Mitigations

| Failure | Impact | Mitigation |
|---|---|---|
| LLM API timeout | Customer gets no response | 30s timeout → auto-escalate to human |
| CRM API down | Agent has no context | Serve from Redis cache; continue with degraded context |
| NATS partition | Messages not processed | JetStream durability; messages replayed on reconnect |
| Python AI crash | Queue backs up | NATS redelivers after ack_wait (30s); multiple Python instances |
| Meta rate limit hit | Messages fail to send | Queue sends; exponential backoff; alert tenant |
| DB connection exhaustion | All services degrade | pgbouncer connection pooling |

### Horizontal Scaling Notes

- **Go gateway**: stateless, scale freely behind a load balancer. WebSocket connections are sticky (use `lb_hash $remote_addr` in Nginx).
- **Python AI**: stateless (all state in Redis/Postgres), scale freely. Add more consumers on the NATS subject.
- **NATS**: single node sufficient to ~50k msg/s; upgrade to 3-node cluster for HA.
- **PostgreSQL**: read replicas for analytics queries; main instance for writes.

---

## 18. Implementation Roadmap

An AI agent implementing this system should follow this phased order to ensure a working system at each checkpoint.

### Phase 1 — Foundation (Week 1-2)
- [ ] Docker Compose with Postgres (pgvector), Redis, NATS
- [ ] Database migrations (Section 7.1 schemas)
- [ ] Go gateway skeleton: HTTP server, config, DB/Redis/NATS connections
- [ ] Go webhook endpoint: signature verification + NATS publish
- [ ] Python AI service skeleton: FastAPI, NATS subscriber, DB connection
- [ ] Basic end-to-end: webhook → NATS → Python consumer logs the message

### Phase 2 — AI Core (Week 3-4)
- [ ] Prompt builder with static customer context placeholder
- [ ] LLM call via Anthropic API (no tools yet)
- [ ] Python → Go send message (AI response delivered to customer)
- [ ] Conversation history load from Postgres
- [ ] Tool calling loop (start with `search_knowledge_base` tool only)
- [ ] RAG pipeline: embedder + pgvector indexing + retrieval

### Phase 3 — CRM Integration (Week 5-6)
- [ ] HubSpot adapter (most common starting point)
- [ ] Redis CRM cache layer
- [ ] Full tool set: `get_customer`, `get_open_tickets`, `create_ticket`, `get_order_status`
- [ ] CRM proxy in Go with rate limiting
- [ ] Escalation flow: tool call → NATS event → Go WS push

### Phase 4 — Dashboard (Week 7-8)
- [ ] Agent login (JWT)
- [ ] WebSocket hub in Go
- [ ] Svelte: inbox view, conversation detail, CRM sidebar
- [ ] Human agent send message
- [ ] AI reasoning trace display
- [ ] Knowledge base upload UI

### Phase 5 — Calling & Polish (Week 9-10)
- [ ] WhatsApp Calling service in Go
- [ ] Call state machine + CRM logging
- [ ] Multi-tenancy hardening (RLS, per-tenant Meta tokens)
- [ ] Analytics API + Svelte analytics view
- [ ] Alerting: escalation rate spike, LLM error rate
- [ ] Load testing: 100 concurrent conversations

---

## Appendix A — Directory Structure (Full Monorepo)

```
/
├── services/
│   ├── go-gateway/
│   └── python-ai/
├── frontend/
├── infrastructure/
│   ├── docker-compose.yml
│   ├── k8s/
│   └── migrations/
│       └── 001_initial_schema.sql
├── docs/
│   └── architecture.md    ← this file
└── README.md
```

## Appendix B — Key External Documentation

- Meta Cloud API: https://developers.facebook.com/docs/whatsapp/cloud-api
- Anthropic Tool Use: https://docs.anthropic.com/en/docs/build-with-claude/tool-use
- NATS JetStream: https://docs.nats.io/nats-concepts/jetstream
- pgvector: https://github.com/pgvector/pgvector
- LangGraph: https://langchain-ai.github.io/langgraph/
- WhatsApp Business Calling: https://developers.facebook.com/docs/whatsapp/cloud-api/calling
