# Product Research Snapshot (2026-03-06)

This document tracks the current Grafana Cloud product areas that matter most
for an agent-first CLI.

## Most Important Grafana Cloud Features For Agents

1. Core Grafana workspace APIs
- Dashboards, folders, annotations, alerting, query history, sharing, short URLs,
  and server-side rendering are the highest-value read surfaces for agents.
- Docs:
  - https://grafana.com/docs/grafana/latest/developers/http_api/
  - https://grafana.com/docs/grafana/latest/developers/http_api/dashboard/
  - https://grafana.com/docs/grafana/latest/developers/http_api/folder/
  - https://grafana.com/docs/grafana/latest/developers/http_api/annotations/
  - https://grafana.com/docs/grafana/latest/developers/http_api/query_history/
  - https://grafana.com/docs/grafana/latest/developers/http_api/alerting_provisioning/
  - https://grafana.com/docs/grafana/latest/dashboards/share-dashboards-panels/

2. LGTM + Profiles data plane
- Metrics, logs, traces, and continuous profiling are still the main incident and
  performance investigation loop for agents.
- Docs:
  - https://grafana.com/products/cloud/features/
  - https://grafana.com/docs/grafana-cloud/profiles/continuous-profiling/

3. Application Observability and Frontend Observability
- These features add service-level and end-user context that is difficult to infer
  from raw telemetry alone.
- Docs:
  - https://grafana.com/docs/grafana-cloud/monitor-applications/application-observability/
  - https://grafana.com/docs/grafana-cloud/monitor-applications/frontend-observability/

4. Alerting, IRM, OnCall, and SLO
- Agents need the operational envelope around an incident: active alert rules,
  escalation paths, incidents, and error budget state.
- Docs:
  - https://grafana.com/docs/grafana-cloud/alerting-and-irm/irm/
  - https://grafana.com/docs/grafana-cloud/alerting-and-irm/slo/

5. Synthetic Monitoring and k6
- These are important when an agent must reproduce failures, validate fixes, or
  confirm regressions from outside the cluster.
- Docs:
  - https://grafana.com/docs/grafana-cloud/testing/synthetic-monitoring/
  - https://grafana.com/docs/grafana-cloud/testing/k6/

6. Knowledge Graph
- Knowledge Graph is a high-value agent surface because it ties telemetry to
  entities, services, and root-cause context.
- Docs:
  - https://grafana.com/docs/grafana-cloud/knowledge-graph/

7. Grafana Assistant and remote MCP
- Assistant is the closest official Grafana feature to agent-native workflows.
  Remote MCP makes Grafana’s context available to external coding/debugging agents.
- Docs:
  - https://grafana.com/docs/plugins/grafana-assistant-app/latest/
  - https://grafana.com/docs/grafana-cloud/machine-learning/assistant/use-assistant/use-remote-mcp-servers/

8. Grafana Cloud control plane
- Access policies, service accounts, plugins, billed usage, and stack inventory
  are necessary for reliable automation and multi-stack agent workflows.
- Docs:
  - https://grafana.com/docs/grafana-cloud/developer-resources/api-reference/cloud-api/

## What This Means For `grafana-cli`

The highest-priority command areas for agents are:

1. Dashboard context
- Read dashboard JSON, folder context, versions, permissions, annotations,
  shared links, and rendered screenshots.

2. Investigation context
- Read alert rules, contact points, policies, query history, incidents, SLOs,
  and synthetic checks in a compact JSON shape.

3. Product-aware runtime access
- Keep metrics/logs/traces fast, but add profiles, application observability,
  frontend observability, and knowledge graph surfaces.

4. Agent-native delegation
- Support Assistant chat/status/skills, streaming/status events, feedback, and
  remote-MCP-aware workflows.

5. Cloud administration
- Expose access policies, service accounts, plugins, billed usage, and richer
  stack inventory without forcing agents into raw API mode.

## DeepWiki References

- `grafana/grafanactl` patterns:
  - https://deepwiki.com/search/what-capabilities-and-resource_8c19831b-85d8-4ffb-b6e5-131fd43c67db
- `grafana/terraform-provider-grafana` domain map:
  - https://deepwiki.com/search/what-grafana-cloud-and-grafana_ddaf2bc8-6ca0-4eeb-930b-d19cbac82d1e
- `grafana-openapi-client-go` as typed coverage foundation:
  - https://deepwiki.com/search/what-is-this-client-generated_af82cfba-fed0-4e86-a335-3348772ec4b8
