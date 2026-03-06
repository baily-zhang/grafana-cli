# Discovery-First Plan

Date: 2026-03-06

## Goal

Make discovery a first-class interface of `grafana-cli`.

An agent or engineer should be able to answer these questions with one bounded CLI call:

- What can this CLI do?
- Which command should I run next?
- Which flags matter?
- What query syntax is expected?
- What shape will the output have?
- How can I stay within a small token budget?

The target is not feature parity with `datadog-pup`.
The target is a Grafana-native CLI that is easier to discover, cheaper to operate in agent loops, and safer to compose.

## What We Keep

These are already strengths and should remain intact:

- Compact JSON as the default contract.
- Output shaping with `--json`, `--jq`, and `--template`.
- Deterministic command behavior.
- A raw `api` escape hatch for uncovered Grafana surfaces.
- Context support and token storage outside the config file.

## Product Principles

### 1. Discovery Is Part Of The Product

Every command must be self-describing.
No new command should ship without:

- a short description
- flag metadata
- example invocations
- expected output shape
- related commands
- read/write classification
- token-cost guidance

### 2. Token Usage Is A Hard Constraint

Discovery must be useful without being verbose.
The CLI should prefer:

- subtree help over global dumps
- compact schema over prose
- summaries over raw payloads
- bounded list defaults
- truncation metadata over silent overfetching

### 3. Human And Agent UX Can Differ

Humans need readable help.
Agents need deterministic schemas.
The CLI should support both without duplicating command behavior.

### 4. Breadth Follows Clarity

Do not add wide product coverage if discovery, safety, and token discipline are weak.
Each new domain should first have a clear discovery story.

## Success Criteria

### Discovery

- `grafana --help` returns a compact, structured overview.
- `grafana <domain> --help` returns only the relevant subtree.
- `grafana schema --compact` returns a bounded machine-readable contract.
- Agents can discover query syntax and examples without reading repo docs.

### Token Usage

- Root compact schema should stay within a small single-call budget.
- Domain schemas should be substantially smaller than root schema.
- Default list and search commands should include count and truncation metadata when useful.
- High-frequency workflows should have summary-first commands before raw data commands.

### UX

- A user can authenticate with fewer required inputs.
- A user can understand missing configuration from CLI output alone.
- A user can perform a common investigation without falling back to raw API calls.

## Workstreams

### 1. Discovery Surface

### Objective

Give the CLI a first-class discovery contract for both humans and agents.

### Deliverables

- Add a command metadata registry as the single source of truth.
- Add `grafana schema` with at least:
  - `--compact`
  - optional domain/subtree targeting
- Make `grafana --help` and `grafana <domain> --help` emit structured discovery output.
- Support human-readable help via `--output pretty` or equivalent rendering.

### Minimum Schema Fields

- `version`
- `description`
- `auth`
- `global_flags`
- `commands`
- `query_syntax`
- `time_formats`
- `workflows`
- `best_practices`
- `anti_patterns`

### Per-Command Metadata

- `name`
- `full_path`
- `description`
- `flags`
- `read_only`
- `examples`
- `output_shape`
- `related_commands`
- `token_cost`

### Notes

- Root help should stay compact by default.
- Domain help should include only domain-relevant query syntax and workflows.
- The schema must be generated from code metadata, not maintained by hand in multiple places.

### 2. Token-Efficient Discovery Contract

### Objective

Make discovery cheap enough for repeated use in agent loops.

### Deliverables

- Define compact and expanded schema modes.
- Introduce subtree-targeted help everywhere.
- Add output metadata for list/search responses:
  - `count`
  - `truncated`
  - `next_action`
- Add token-cost labels to commands and workflows.

### Token Budget Rules

- Compact schema is the default machine-readable help.
- Expanded discovery must be opt-in.
- Large domains should expose focused subtree help instead of one large blob.
- Help text should favor examples and field names over long explanation paragraphs.
- Default responses should omit raw nested payloads unless explicitly requested.
- Summary commands should be preferred for first-pass investigation.

### Example Contract Direction

- `grafana --help` -> compact root schema
- `grafana runtime --help` -> runtime-only schema
- `grafana schema --compact` -> explicit compact contract
- `grafana schema --full runtime` -> richer domain contract when needed

### 3. Authentication And Setup Simplification

### Objective

Reduce setup friction so discovery remains useful after the first command.

### Current Problem

`grafana auth login` can require a token plus multiple URLs for base, cloud, metrics, logs, and traces.
That is too much surface area up front.

### Deliverables

- Allow login from a single stack identifier when possible.
- Resolve Prometheus, Loki, and Tempo endpoints automatically from stack metadata.
- Add an auth diagnostics surface, for example:
  - `grafana auth doctor`
  - richer `grafana auth status`
- Report which capabilities are unavailable because of missing URLs or scopes.

### Output Requirements

- Auth status should show resolved endpoints.
- Missing configuration should be explicit and actionable.
- Output should stay compact and omit null-heavy payloads.

### 4. Runtime Investigation Ergonomics

### Objective

Move from thin API wrappers to a better investigation interface.

### Current Problem

The runtime surface is narrow:

- `runtime metrics query`
- `runtime logs query`
- `runtime traces search`
- `aggregate snapshot`

This is usable, but it is not yet optimized for progressive investigation.

### Deliverables

- Add relative time parsing such as `5m`, `1h`, `24h`, `7d`.
- Add clearer runtime verbs such as search, list, and aggregate where the backend supports them.
- Add domain-specific query syntax examples to runtime help.
- Add bounded defaults and truncation metadata.
- Add service-oriented entry points where Grafana data can support them.

### Preferred Investigation Pattern

1. Aggregate to find the hotspot.
2. Narrow by service, route, or label.
3. Fetch a small set of examples.
4. Escalate to raw payloads only when needed.

### Initial Candidate Additions

- `runtime logs aggregate`
- `runtime traces aggregate`
- query examples for PromQL, LogQL, and TraceQL
- relative time support across runtime commands

### 5. Incident Envelope Coverage

### Objective

Prioritize the product areas that make investigation complete, not just possible.

### First Domains To Add

- SLO
- IRM / incidents
- OnCall
- Synthetic Monitoring
- Query history

### Second Wave

- Profiles / continuous profiling
- Application Observability
- Frontend Observability
- Knowledge Graph
- richer cloud administration

### Rule

Do not add a new domain unless it ships with:

- structured help
- query or filter examples
- bounded default outputs
- read/write classification

### 6. Safety And Output Modes

### Objective

Make command execution safer and easier to interpret in both human and agent contexts.

### Deliverables

- Add a global `--read-only` mode.
- Add a human-oriented table renderer for common list commands.
- Consider YAML output only if it adds real value beyond JSON and pretty output.
- Introduce a standard agent envelope for selected commands:
  - `status`
  - `data`
  - `metadata`

### Metadata Fields

- `count`
- `truncated`
- `command`
- `next_action`
- optional warnings

### Guardrails

- Destructive commands should require explicit confirmation or `--yes`.
- Agent mode should avoid interactive hangs.
- Response envelopes should not bloat single-object results unnecessarily.

### 7. Preserve Grafana-Specific Advantages

### Objective

Improve discovery without turning the CLI into a clone of another product.

### Rules

- Keep `api` as the escape hatch.
- Keep `--json`, `--jq`, and `--template` as first-class features.
- Prefer narrow Grafana-native commands over a huge number of thin wrappers.
- Use discovery metadata to guide users toward the narrow command before they fall back to `api`.

## Proposed Phasing

### Phase 1: Discovery Foundation

- Build command metadata registry.
- Implement `grafana schema --compact`.
- Make root and subtree help schema-driven.
- Add discovery tests for stability and coverage.

Exit criteria:

- Root and subtree help are generated from one source of truth.
- Compact schema is usable by agents without external docs.

### Phase 2: Token And Output Contract

- Add compact vs expanded schema modes.
- Add response metadata for list/search style commands.
- Define token-cost guidance and response-size defaults.
- Preserve current projection tools.

Exit criteria:

- Agents can discover and operate the CLI without large help payloads.
- Common list/search results signal truncation and next steps.

### Phase 3: Auth Simplification

- Add endpoint inference from a smaller login surface.
- Improve auth status and diagnostics.
- Document missing-capability reporting.

Exit criteria:

- Login no longer requires manual product endpoint entry in the common case.
- Status output explains what is configured and what is not.

### Phase 4: Runtime UX Upgrade

- Add relative time parsing.
- Add runtime aggregate patterns and better help.
- Introduce progressive investigation examples.

Exit criteria:

- Runtime commands support a clear summary-first investigation loop.

### Phase 5: Incident Envelope

- Add SLO, IRM, OnCall, Synthetic Monitoring, and query history.
- Keep discovery metadata mandatory for every new domain.

Exit criteria:

- An incident workflow can stay inside `grafana-cli` for the common case.

## Metrics To Track

- Median token size of root help.
- Median token size of domain help.
- Median token size of first-pass investigation workflows.
- Percentage of commands with complete metadata coverage.
- Percentage of new domains shipped with query syntax and examples.
- Percentage of user workflows that avoid raw `api`.

## Immediate Next Steps

1. Define the command metadata model in code.
2. Implement `grafana schema --compact` using current commands only.
3. Convert root help and one subtree help path to the new model.
4. Add response metadata to one runtime command and one list command.
5. Prototype auth endpoint inference before broadening product coverage.
