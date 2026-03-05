# grafana-cli

Agent-first CLI to control Grafana and Grafana Cloud.

This project is a **WIP hackathon build by @matiasvillaverde and @marctc**.

## Why This Exists

Most Grafana tooling is human-first. This CLI is designed for **agents** that need to:

- understand and triage incidents
- query logs/metrics/traces fast
- inspect cloud stacks and data sources
- create dashboards programmatically
- run deterministic workflows with low token usage

## Agent-First Contract

- **Compact JSON by default** (`--output json` implied)
- **Optional readable output**: `--output pretty`
- **Token minimization**: `--fields` projection to return only required keys
- **Deterministic command behavior**: stable flags, stable output shapes
- **Composability**: each command is script/agent safe

## Current Capabilities

- Auth/session
  - `auth login|status|logout`
- Raw API access
  - `api <METHOD> <PATH> [--body JSON]`
- Cloud inventory
  - `cloud stacks list`
- Incident + runtime investigation
  - `incident analyze --goal ...`
  - `runtime metrics query --expr ...`
  - `runtime logs query --query ...`
  - `runtime traces search --query ...`
  - `aggregate snapshot --metric-expr ... --log-query ... --trace-query ...`
- Dashboard and datasource operations
  - `dashboards list --query ... --tag ... --limit ...`
  - `dashboards create --title ...` or `--template-json ...`
  - `datasources list --type ... --name ...`
- Agent workflows
  - `agent plan --goal ...`
  - `agent run --goal ...`

## Quick Start

```bash
go run ./cmd/grafana auth login \
  --token "$GRAFANA_TOKEN" \
  --base-url "https://your-stack.grafana.net" \
  --cloud-url "https://grafana.com" \
  --prom-url "https://prometheus-prod-01-eu-west-0.grafana.net" \
  --logs-url "https://logs-prod-01-eu-west-0.grafana.net" \
  --traces-url "https://tempo-prod-01-eu-west-0.grafana.net"

# Incident analysis (compact JSON)
go run ./cmd/grafana incident analyze --goal "Investigate elevated error rate"

# Return only what the agent needs
go run ./cmd/grafana --fields summary.metrics_series,summary.log_streams incident analyze --goal "Latency spike"

# Create a dashboard from JSON template
go run ./cmd/grafana dashboards create --template-json '{"title":"Incident Overview","schemaVersion":39,"version":0,"panels":[]}'
```

## Design Inspiration

This README and CLI structure were informed by strong open-source CLI patterns from:

- `cli/cli` (GitHub CLI)
- `cloudflare/workers-sdk` (`wrangler`)
- `supabase/cli`
- `vercel/vercel` (CLI package)
- `Aider-AI/aider`

We borrowed patterns around command discoverability, non-interactive execution, stable JSON outputs, and strong automation ergonomics.

## Quality Gate

CI enforces **100% unit test coverage**.

```bash
go test ./... -covermode=atomic -coverprofile=coverage.out
```

## Roadmap

- broader Grafana Cloud product coverage (alerting, access control, reporting, synthetic monitoring, OnCall, k6)
- richer agent execution plans and remediation actions
- **Graph RAG for past incidents** to reuse historical context during incident triage and diagnosis

## Architecture

See [docs/architecture.md](docs/architecture.md).
