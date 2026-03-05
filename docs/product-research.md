# Product Research Snapshot (2026-03-05)

This document tracks the product and CLI research used to prioritize agent-first
coverage in `grafana-cli`.

## Grafana Product Surface (Official)

- Grafana products index: https://grafana.com/products/
- Grafana Cloud overview/features: https://grafana.com/products/cloud/features/
- Grafana Assistant plugin docs: https://grafana.com/docs/plugins/grafana-assistant-app/latest/
- Grafana Assistant HTTP API for external systems:
  - `POST /api/plugins/grafana-assistant-app/resources/api/v1/assistant/chats`
  - `GET /api/plugins/grafana-assistant-app/resources/api/v1/chats/{chatId}`
  - `GET /api/plugins/grafana-assistant-app/resources/api/v1/chats/{chatId}/stream`
  - `GET /api/plugins/grafana-assistant-app/resources/api/v1/assistant/skills`
  - `POST /api/plugins/grafana-assistant-app/resources/api/v1/chats/{chatId}/feedback`
- Grafana Assistant remote MCP servers:
  - https://grafana.com/docs/grafana-cloud/machine-learning/assistant/use-assistant/use-remote-mcp-servers/

## Agent-First CLI Inspiration (Official READMEs)

- GitHub CLI: https://github.com/cli/cli
- Cloudflare Wrangler: https://github.com/cloudflare/workers-sdk/tree/main/packages/wrangler
- Supabase CLI: https://github.com/supabase/cli
- Aider (agent coding CLI): https://github.com/Aider-AI/aider

## Design Implications For This CLI

- JSON-first deterministic outputs and explicit projection (`--fields`) for low token usage.
- Stable subcommands for incident triage: logs, metrics, traces, and aggregate snapshots.
- Assistant-native commands (`assistant chat|status|skills`) so agents can offload
  investigation loops to Grafana Assistant.
- Keep raw API escape hatch for immediate access to new Grafana features while
  typed subcommands are added.
