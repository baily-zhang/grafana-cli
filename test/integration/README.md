# Integration Tests

This suite runs the CLI against a local Grafana stack started with Docker Compose.

## Stack

- Grafana `11.6.13`
- Prometheus `v3.10.0`
- Pushgateway `v1.11.2`
- Loki pinned by digest
- Tempo `v2.10.1` pinned by digest and started in single-binary mode
- Grafana image renderer `v5.6.3`

The bootstrap script creates a service-account token and seeds telemetry into Prometheus, Loki, and Tempo.

## Run locally

Start the stack:

```bash
docker compose -f test/integration/docker-compose.yml up -d
```

Seed Grafana and write the integration environment file:

```bash
env_file="${TMPDIR:-/tmp}/grafana-cli-integration.env"
./test/integration/bootstrap.sh "$env_file"
```

Run all integration shards:

```bash
set -a
source "$env_file"
set +a
go test -tags=integration ./cmd/grafana -count=1
```

Run one shard:

```bash
set -a
source "$env_file"
set +a
go test -tags=integration ./cmd/grafana -run '^TestRuntimeObservability$' -count=1
```

Stop and remove the stack:

```bash
docker compose -f test/integration/docker-compose.yml down -v
```

## Shards

The workflow follows the command-coverage groups used by the CLI tests:

- `TestSchemaGlobalFlags`
- `TestAuthConfig`
- `TestDashboardsDatasources`
- `TestFoldersAnnotationsAlerting`
- `TestRuntimeObservability`
- `TestInvestigationIncidents`
- `TestAssistantAccessCloud`
- `TestAgentWorkflows`

## Notes

Trace ingestion is real, but the Go integration harness serves `/api/search` through a local proxy. The minimal Tempo fixture accepts the spans, but its recent-search API is not stable enough in this setup to rely on directly for deterministic CI.

The shard-to-command mapping lives in [command-coverage.json](/home/marctc/workspace/grafana-cli/test/integration/command-coverage.json). A unit test fails if the discovery schema adds a new leaf command that is not assigned to an integration shard.

If you do want a repository-local env file for debugging, pass an explicit output path such as `./test/integration/bootstrap.sh test/integration/integration.env`. That file is gitignored.
