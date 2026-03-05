# Agent-First Grafana CLI Architecture

## Goal

Build an open source Go CLI that can interact with Grafana Cloud comprehensively while being optimized for AI/automation agents.

## Core product decisions

1. Dual interface model:
- High-level commands (`cloud`, `runtime`, `aggregate`, `agent`) for common workflows.
- Raw API escape hatch (`api`) for full surface access and forward compatibility.

2. Agent-first defaults:
- Deterministic machine-readable output (`--output json`).
- Explicit plan step (`agent plan`) before execution (`agent run`).
- Structured responses suitable for autonomous orchestration.

3. Runtime + aggregator coverage from day one:
- Metrics queries (PromQL range).
- Logs queries (LogQL range).
- Trace search.
- Cross-signal aggregate snapshot.
- Grafana Assistant chat/status/skills for agent investigation loops.

## Main Grafana use cases to optimize

1. Incident triage
- Correlate errors and latency across metrics, logs, and traces.
- Fast command: `agent run --goal "Investigate elevated error rate"`.

2. Fleet and stack inventory
- Discover cloud stacks and service endpoints.
- Fast command: `cloud stacks list`.

3. Programmatic observability data extraction
- Pull runtime telemetry into automation workflows.
- Commands: `runtime metrics query`, `runtime logs query`, `runtime traces search`.

4. Assistant-driven investigation
- Delegate scoped investigations to Grafana Assistant and fetch compact status/results.
- Commands: `assistant chat`, `assistant status`, `assistant skills`.

5. Infrastructure automation
- Use command JSON outputs in CI agents and autonomous remediation workflows.

## Inspiration from DeepWiki research

### `cli/cli` (gh)
- Cobra command tree + factory/dependency injection pattern.
- Dedicated `gh api` raw command with formatting/pagination.
- Agent-task lifecycle (`create/list/view`) demonstrates explicit agent job UX.

### `stripe/stripe-cli` and `twilio/twilio-cli`
- Efficient broad API coverage through generated or metadata-driven commands.
- Strong separation between generated endpoint wrappers and manual high-value commands.

### `supabase/cli` and `wrangler`
- Configuration layering and non-interactive execution patterns.
- Environment/context support and machine-readable outputs.

### Grafana ecosystem repos
- `grafana/grafanactl`: resource-centric CLI patterns and context-based config.
- `grafana/terraform-provider-grafana`: practical Grafana Cloud domain map (stacks, access policies, service-specific APIs).
- `grafana/grafana-openapi-client-go`: transport/auth/retry model for API clients.
- `grafana/mimir`, `grafana/tempo`, `grafana/synthetic-monitoring-agent`: runtime query and telemetry patterns.

## Initial architecture

1. `internal/config`
- File-backed profile (token + endpoints + org ID).
- Deterministic defaults and explicit auth status.

2. `internal/grafana`
- HTTP client wrapper with auth and org headers.
- Domain methods for cloud stacks, metrics/logs/traces, raw API, and aggregate snapshot.

3. `internal/agent`
- Deterministic playbook planner that maps goals to query actions.
- Request template generation for execution.

4. `internal/cli`
- Command dispatcher with explicit subcommand boundaries.
- Shared output mode (`text` or `json`) for every command.

## API domain map for full product coverage (roadmap)

1. Core Grafana API
- Dashboards, folders, datasources, users, teams, access control, alerting, reporting.

2. Cloud control plane
- Stacks, access policies/tokens, service accounts.

3. Runtime services
- Metrics (Mimir/Prometheus API), logs (Loki API), traces (Tempo API), alerting endpoints.

4. Specialized cloud products
- Synthetic monitoring, OnCall, k6, frontend observability, fleet management.

## CI and quality policy

- Mandatory `go test` with `-covermode=atomic`.
- Hard fail when total coverage `< 100.0%`.
- Keep command contracts stable to avoid breaking agent integrations.

## DeepWiki sources

- https://deepwiki.com/search/summarize-the-command-architec_e80e8ec2-35f5-4695-b3ce-132c4411a946
- https://deepwiki.com/search/explain-the-agent-task-system_6b2b58e2-17e2-4d16-ba75-f4ff44682041
- https://deepwiki.com/search/how-does-gh-implement-the-dire_bbb0e13c-0281-45f7-9e6f-dc4ffa0b5a84
- https://deepwiki.com/search/explain-the-architecture-patte_673fa262-24f2-45fa-977f-5c4baecedac2
- https://deepwiki.com/search/what-patterns-should-a-new-go_db7d19e6-a83c-4107-931d-c4ac5f348a67
- https://deepwiki.com/search/from-wrangler-cli-what-pattern_b156516d-3a1e-4805-bd5b-2c8a9271890d
- https://deepwiki.com/search/how-does-twilio-cli-dynamic-co_cc148e77-97c2-4e47-a47a-85f280735527
- https://deepwiki.com/search/list-the-grafana-cloud-resourc_fa8305ac-5c4b-46cb-980c-7e8fbbcffedb
- https://deepwiki.com/search/explain-how-this-generated-cli_a93cf132-026f-4c45-af12-68f13a5de7a3
- https://deepwiki.com/search/for-a-cli-that-retrieves-runti_5a521719-d680-437c-92a7-944b1fe0cabf
- https://deepwiki.com/search/for-agent-automation-over-trac_de1e61b3-24f4-47b2-b41c-c761edb7f161
- https://deepwiki.com/search/what-api-and-data-model-patter_784b4890-95c2-484e-aacf-bada9baa24dd
