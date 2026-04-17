<div align="center">

# WhatsApp MCP Server

**Give AI assistants access to your WhatsApp conversations**

[![Go Version](https://img.shields.io/badge/Go-1.25%2B-00ADD8?style=flat&logo=go)](https://go.dev/)
[![MCP Protocol](https://img.shields.io/badge/MCP-Compatible-7C3AED?style=flat)](https://modelcontextprotocol.io)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?style=flat&logo=docker&logoColor=white)](https://www.docker.com/)
[![License: GPL v3](https://img.shields.io/badge/License-GPLv3-blue.svg?style=flat)](LICENSE)
[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/felipeadeildo/whatsapp-mcp)

*Built with [whatsmeow](https://github.com/tulir/whatsmeow) and [mcp-go](https://github.com/mark3labs/mcp-go)*

[Features](#-features) • [Quick Start](#-quick-start) • [Architecture](#-architecture) • [MCP Integration](#-mcp-integration)

</div>

## 🎯 What is This?

A **Model Context Protocol (MCP) server** that bridges WhatsApp and AI assistants like Claude. It exposes your WhatsApp messages through standardized MCP tools, prompts, and resources - allowing AI to read, search, and send messages on your behalf.

**The Vision:** Let AI handle your WhatsApp conversations intelligently, with full context and natural language understanding.

```
You: "Summarize what João said about the budget meeting"
AI:  *searches all your chats* → "João mentioned in the Tech Team group..."

You: "Reply to Maria's last message and schedule lunch"
AI:  *reads context, sends reply* → "Sent! I've proposed Thursday at noon"
```

## ✨ Features

### Core Capabilities

- **📱 Full WhatsApp Integration** - Connect to WhatsApp Web using your existing account
- **💾 Local-First Storage** - All messages stored in SQLite, synced in real-time
- **🔍 Powerful Search** - Pattern matching, cross-chat queries, sender filtering
- **⏱️ Timezone Support** - Messages displayed in your local timezone
- **📥 On-Demand Loading** - Fetch older messages from WhatsApp servers as needed
- **🔐 Secure by Design** - API key authentication, local data storage, HTTPS ready

### MCP Features

This server implements the full MCP specification with:

- **11 Tools** for WhatsApp operations (read, search, send text + files, manage trusted users)
- **4 Prompts** for common workflows
- **1 Resource + 1 Resource Template** - a consolidated usage guide and dynamic media access
- **Server Instructions** for optimal AI interactions
- **@claude Trigger** - Mention `@claude` in any chat to get an AI response, with per-chat sessions and auto-compaction

#### Tools

| Tool | Purpose | Highlights |
|------|---------|-----------|
| `list_chats` | Browse conversations | Ordered by recent activity |
| `get_chat_messages` | Read specific chat | Pagination, sender filtering |
| `search_messages` | Search across all chats | Pattern matching, wildcards |
| `find_chat` | Locate chat by name | Fuzzy search support |
| `send_message` | Send WhatsApp messages | To any chat or group |
| `send_file` | Send a local file to a chat | Auto-detects image/video/audio/document from MIME; `as_document=true` to force generic doc; optional caption |
| `load_more_messages` | Fetch older history | On-demand from servers |
| `get_my_info` | Get your profile info | JID, name, status, picture |
| `add_trusted_user` | Trust a user for @claude | Authorize @claude triggers |
| `remove_trusted_user` | Revoke @claude access | Remove trusted user |
| `list_trusted_users` | List trusted users | View authorized users |

#### Prompts

Pre-built workflows that guide AI assistants:

- **`search_person_messages`** - Find ALL messages from someone across all chats
- **`get_context_about_person`** - Comprehensive analysis of someone's messages
- **`analyze_conversation`** - Summarize recent chat activity
- **`search_keyword`** - Find specific topics across conversations

#### Resources

- **`whatsapp://guide`** - Consolidated usage guide embedded in the server (JID formats, tool scope, search patterns with GLOB wildcards, history loading workflow)
- **`whatsapp://media/{message_id}`** - Dynamic resource template returning a WhatsApp media file (image/video/audio/document) as a base64 blob, scoped to the server's media directory

## 🏗️ Architecture

```mermaid
graph TB
    subgraph "AI Client"
        A[AI Assistant <br/> e.g., Claude Web]
    end

    subgraph "WhatsApp MCP Server"
        B[MCP HTTP Server :8080]
        C[MCP Layer]
        D[WhatsApp Client]
        E[(SQLite Database)]

        B -->|/mcp endpoint| C
        B -->|/health| B

        C -->|Tools| C1[list_chats<br/>get_chat_messages<br/>search_messages<br/>find_chat<br/>send_message<br/>send_file<br/>load_more_messages<br/>get_my_info<br/>add_trusted_user<br/>remove_trusted_user<br/>list_trusted_users]
        C -->|Prompts| C2[search_person_messages<br/>get_context_about_person<br/>analyze_conversation<br/>search_keyword]
        C -->|Resources| C3[whatsapp://guide<br/>whatsapp://media/&#123;id&#125;]

        C1 -.->|read/write| E
        C1 -.->|send| D

        D -->|sync messages| E
        D <-->|WhatsApp Protocol| F
    end

    subgraph "WhatsApp"
        F[WhatsApp Servers]
    end

    A <-->|Streamable HTTP<br/>API Key Auth| B

    style A fill:#4A90E2,stroke:#2E5C8A,stroke-width:2px,color:#000
    style B fill:#F5A623,stroke:#C67E1B,stroke-width:2px,color:#000
    style C fill:#9013FE,stroke:#6B0FC7,stroke-width:2px,color:#fff
    style C1 fill:#50E3C2,stroke:#3AAA94,stroke-width:2px,color:#000
    style C2 fill:#BD10E0,stroke:#9012FE,stroke-width:2px,color:#fff
    style C3 fill:#F5A623,stroke:#C67E1B,stroke-width:2px,color:#000
    style D fill:#50E3C2,stroke:#3AAA94,stroke-width:2px,color:#000
    style E fill:#E85D75,stroke:#B5475C,stroke-width:2px,color:#fff
    style F fill:#25D366,stroke:#1DA851,stroke-width:2px,color:#000
```

### How It Works

1. **Initial Sync** - WhatsApp sends message history on first connection
2. **Real-Time Updates** - All new messages automatically stored in SQLite
3. **MCP Exposure** - Tools, prompts, and resources expose functionality to AI
4. **On-Demand Loading** - Fetch older messages from WhatsApp when needed
5. **AI Integration** - Claude (or any MCP client) accesses WhatsApp through standardized protocol

## 🚀 Quick Start

### Prerequisites

- **Go 1.25.5+** (for local setup) or **Docker** (recommended)
- **WhatsApp account** (will be linked via QR code)
- **MCP-compatible AI client** (Claude, Cursor, etc.)

### Option 1: Docker Setup (Recommended)

1. **Clone and configure**
   ```bash
   git clone https://github.com/felipeadeildo/whatsapp-mcp
   cd whatsapp-mcp
   cp .env.example .env
   # Edit .env with your settings (API key, timezone, etc.)
   ```

2. **Start the server**
   ```bash
   docker compose up -d
   ```

3. **Link WhatsApp**
   ```bash
   # View logs to see QR code
   docker compose logs -f whatsapp-mcp

   # Scan QR code with WhatsApp mobile app:
   # Settings → Linked Devices → Link a Device
   ```

4. **Verify it's running**
   ```bash
   curl http://localhost:8080/health
   # Expected: "OK"
   ```

### Option 2: Local Setup

1. **Install dependencies**
   ```bash
   git clone https://github.com/felipeadeildo/whatsapp-mcp
   cd whatsapp-mcp
   go mod download
   ```

2. **Configure environment**
   ```bash
   cp .env.example .env
   # Edit .env with your settings
   ```

3. **Run the server**
   ```bash
   go run main.go
   ```

4. **Link WhatsApp** (scan QR code shown in terminal)

## 🔌 MCP Integration

### Connect to Claude Desktop

Add to your Claude Desktop config (`~/Library/Application Support/Claude/claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "whatsapp": {
      "url": "http://localhost:8080/mcp/your-secret-api-key",
      "transport": "http"
    }
  }
}
```

### Connect to Other MCP Clients

The server exposes an HTTP+SSE endpoint compatible with any MCP client:

- **URL:** `http://localhost:8080/mcp/{API_KEY}`
- **Transport:** Streamable HTTP
- **Authentication:** API key in URL path

## 🎨 Usage Examples

Once connected, your AI assistant can:

### Search for People
```
You: "Find all messages from Arthur across all my chats"
AI: [Uses search_person_messages prompt]
    → Finds messages in DMs, groups, everywhere
    → Analyzes communication patterns
    → Provides context about Arthur
```

### Analyze Conversations
```
You: "What did we discuss in the Tech Team group this week?"
AI: [Uses analyze_conversation prompt]
    → Reads recent messages
    → Summarizes key topics
    → Lists action items and deadlines
```

### Smart Messaging
```
You: "Tell Maria I'll be 10 minutes late"
AI: [Uses find_chat + send_message]
    → Finds Maria's chat
    → Sends contextual message
    → Confirms delivery
```

### Deep Search
```
You: "Find all mentions of 'budget meeting' in any chat"
AI: [Uses search_keyword prompt]
    → Searches across all conversations
    → Shows context around each mention
    → Orders by relevance/date
```

### @claude Trigger
```
Friend: "@claude what's the capital of France?"
Claude: "Paris! The City of Light 🇫🇷"
```
Mention `@claude` in any WhatsApp chat and a headless Claude Code instance reads the last 40 messages for context and replies in-chat. The owner always triggers it; other users must be added via `add_trusted_user`. Sessions are kept per-chat with `--resume` (24h TTL) and auto-compact when they get long — see [@claude Trigger: How It Actually Works](#claude-trigger-how-it-actually-works) for the full picture.

## 📊 Data & Privacy

### Local Storage

All data is stored in `./data/`:
- **`db/`** - Database files
  - `messages.db` - SQLite database with messages and chats
  - `whatsapp_auth.db` - WhatsApp session credentials
- **`media/`** - Downloaded media files
- **`whatsapp.log`** - WhatsApp client logs

**⚠️ Important:** Database files contain sensitive data. Keep them secure (file permissions `600`) and backed up.

## 🛣️ Roadmap

### ✅ Implemented

- [x] WhatsApp Web integration via whatsmeow
- [x] Real-time message sync to SQLite
- [x] MCP server with Streamable HTTP transport
- [x] Pattern matching and wildcards
- [x] Sender filtering and cross-chat search
- [x] Timestamp-based pagination
- [x] Timezone support
- [x] On-demand message loading from servers
- [x] Docker deployment (with healthcheck!)
- [x] @claude trigger — mention `@claude` in any chat to get an AI response
- [x] Trusted user access control for @claude trigger
- [x] Per-chat @claude sessions with `--resume`, overlap-diff sending, and per-chat serialization
- [x] Auto-compaction of long @claude sessions via haiku summarization + trust-framing replay
- [x] Owner-aware trust framing and prompt-injection fence
- [x] Silent memory-index participant lookup on new sessions
- [x] In-place thinking-message editing (115 thinking lines, 50 attribution signatures)
- [x] Quote-reply capture and edit-history tracking (before/after text)
- [x] Send files to chats via `send_file` (auto-detects image/video/audio/document)
- [x] Broader media auto-download defaults (image/audio/sticker/video/document, incl. HistorySync)

### 🚧 Planned

- [ ] **Media Support**
  - Voice message transcription
  - Image OCR and analysis
  - Video metadata extraction
  - Document parsing
  - Contact card handling

- [ ] **GraphRAG Integration**
  - Entity extraction from conversations
  - Relationship mapping between contacts
  - Semantic search capabilities
  - Context-aware recommendations

- [ ] **Enhanced Tools**
  - Mark messages as read
  - React to messages (emoji reactions)
  - Group management (create, members)
  - Status updates
  - Account management (profile picture, name)

- [ ] **Analytics** (maybe)
  - Message statistics
  - Conversation insights
  - Response time tracking

## 📚 Documentation

### MCP Resources (Built-In)

The server exposes two resources over MCP:
- **`whatsapp://guide`** — a single consolidated guide covering the `find_chat` → JID → tool workflow, JID formats (DM / group / channel), the `get_chat_messages` vs `search_messages` scope distinction, GLOB search patterns, and loading older history from WhatsApp servers.
- **`whatsapp://media/{message_id}`** — a resource template that returns a downloaded media file for a given message as a base64 blob, path-validated to stay inside the server's media directory.

AI assistants can read these through the MCP Resources API.

### @claude Trigger Configuration

The `@claude` trigger feature spawns a headless Claude Code CLI instance when someone mentions `@claude` in a WhatsApp message.

| Variable | Default | Description |
|----------|---------|-------------|
| `CLAUDE_TRIGGER_ENABLED` | `false` | Enable/disable the @claude trigger |
| `CLAUDE_PATH` | `claude` | Path to the Claude CLI binary |
| `CLAUDE_MODEL` | *(default)* | Model override (e.g., `sonnet`, `opus`) |
| `CLAUDE_MAX_BUDGET_USD` | `1.00` | Max spend per invocation (empty = no limit) |
| `CLAUDE_TIMEOUT_SECONDS` | `300` | Max seconds to wait for a response |
| `CLAUDE_COMPACT_THRESHOLD_TOKENS` | `120000` | Prior-turn context above this mints a compacted fresh session on the next trigger |
| `CLAUDE_COMPACT_MODEL` | `haiku` | Model used to summarize the existing session during auto-compaction |

**Access control:** The WhatsApp account owner can always trigger `@claude`. Other users must be added via the `add_trusted_user` MCP tool. Untrusted users get a polite rejection message.

### @claude Trigger: How It Actually Works

The fork adds meaningful machinery around the headless Claude spawn — it's not just "pipe the last N messages to `claude -p`".

- **Per-chat sessions.** Each chat gets its own Claude Code session ID, stored in SQLite. The first trigger starts a new session; subsequent triggers `--resume` it (24h TTL). An overlap check compares timestamps against the last context window and sends only the diff on resume, so Claude doesn't re-read messages it already has. A per-chat mutex serializes concurrent triggers for the same chat; different chats run in parallel.

- **In-place thinking message.** The bot sends a random "thinking…" message (115 variants across classic / cooking / vibes / dramatic / silly / nerdy / pop-culture / wholesome / chaotic categories) and then edits that same message in place with the real response (no delete/resend). Responses carry one of 50 random attribution signatures so replies don't all look identical. The edited message is persisted back to the local DB for consistency.

- **Auto-compaction.** Every `--output-format json` response reports per-turn input / cache_creation / cache_read token usage. When the prior turn's total crosses `CLAUDE_COMPACT_THRESHOLD_TOKENS`, the next trigger first runs a summarization sub-call against the existing session (using `CLAUDE_COMPACT_MODEL`, default `haiku`), mints a fresh session seeded with the summary, and **re-applies the full trust/safety framing** — since headless Claude has no CLAUDE.md to reload, this is how non-owner guardrails survive compaction. Failure falls back gracefully to a fresh session without summary.

- **Trust framing + prompt-injection fence.** The prompt distinguishes owner vs. trusted-user contexts and wraps incoming WhatsApp content in a fence so message bodies can't override system instructions.

- **Silent memory lookup.** On new sessions, the trigger silently consults `memory-index` for context about the chat's participants before Claude replies, so background facts surface without the user asking.

- **Quote-replies and edits.** WhatsApp quote-reply context is captured and passed through, and message edits are tracked with before/after text in the DB rather than being shown as raw `[Protocol]` stubs.

- **Attachments.** Claude reads attachments on incoming messages (media auto-download defaults now cover image / audio / sticker / video / document, including HistorySync), and can send files back via the `send_file` tool.

### Environment Variables

See `.env.example` and be happy!

## 🤝 Contributing

This is a personal project I maintain for daily use. Contributions are welcome!

See [CONTRIBUTING.md](CONTRIBUTING.md) for:
- Development setup and workflow
- Project structure (main server vs migration CLI)
- Database migration system
- Code style guidelines

Quick start:
1. Fork the repository
2. Create your feature branch
3. Follow the guidelines in CONTRIBUTING.md
4. Submit a pull request

## ⚠️ Disclaimer

This project is **not affiliated with WhatsApp or Meta**. It uses the unofficial WhatsApp Web API through the whatsmeow library. Use at your own risk.

**Important Notes:**
- WhatsApp may change their API at any time
- Using unofficial APIs may violate WhatsApp's Terms of Service
- This is provided as-is with no warranties
- Keep your session data secure

---

<div align="center">

**Built with ❤️ for the MCP community**

[Report Bug](https://github.com/steefpls/whatsapp-mcp/issues) • [Request Feature](https://github.com/steefpls/whatsapp-mcp/issues)

</div>
