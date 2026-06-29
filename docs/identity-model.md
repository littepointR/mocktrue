# Identity Model

Identity matters in PortWeave because UI tabs, saved workspace files, runtime graph operations, MCP calls, and AI/agent edits all reference graph objects. Prefer stable IDs over display names whenever possible.

## Identity layers

PortWeave has several distinct identity layers:

- Demo workspace ID: catalog entry such as `remote-serial-demo`.
- MCP template ID: backend template such as `serial-remote-raw-tcp`.
- Workspace kind: `portweave.workspace.v1`.
- Graph tab kind: `portweave.graph.v1`.
- Graph document ID: `graph.id` inside a workspace.
- Runtime graph ID: ID passed to backend start/status/send/query/stop tools.
- Node ID: `node.id` inside a graph document.
- Edge ID: `edge.id` inside a graph document.
- Resource identity: OS/network resource claimed by resource-owning nodes.

Do not conflate these layers.

## Graph identity

A `SerialGraphDocument` has `id` and `name`.

- `graph.id` is the stable document identifier inside graph workspace state.
- `graph.name` is user-facing display text.
- `activeGraphId` selects the active graph document in workspace state.
- Graph runtime start uses a runtime graph ID. The frontend defaults to using the document ID, while MCP callers can pass a graph ID explicitly.

Guidance: use `graph.id` for mutations, runtime status, and saved references. Use `name` only for display.

## Node identity

A graph node has:

- `id`: stable within one graph document.
- `type`: provider type, such as `serial.virtual`.
- `name`: optional display name.
- `position`: UI coordinate.
- `config`: provider-specific config.

Guidance:

- Mutate nodes by `node.id`.
- Validate `node.type` against the provider catalog.
- Do not target nodes by display title if an ID is available.
- Runtime `status` and `error` are observations, not authoring identity.

## Edge identity

A graph edge has:

- `id`: stable within one graph document.
- `source`: source node ID.
- `sourceHandle`: source output handle ID.
- `target`: target node ID.
- `targetHandle`: target input handle ID.

Guidance:

- Mutate edges by `edge.id`.
- Use provider catalog handles, not labels.
- Preserve IDs when editing existing edges unless deliberately replacing the edge.

## Resource identity

Resource identity is different from graph object identity. It represents a live external or graph-owned resource claimed at runtime.

### Physical and virtual serial

`serial.physical` and `serial.virtual` use `portName` as the resource key. Two resource-owning nodes in the same graph cannot claim the same non-empty `portName`.

### Remote raw TCP

`serial.remote` uses normalized endpoint + role for uniqueness. The runtime resource ID is shown as a raw TCP URI such as:

```text
raw-tcp://127.0.0.1:3001
```

Validation/resource locking uses endpoint and role, so a server and client can participate in the same localhost loopback demo while two identical client endpoint claims would conflict.

Remote host names are lowercased for endpoint identity, and hosts must not include URL schemes.

## Runtime observations are not identity

These are runtime state, not authoring identity:

- Status (`idle`, `running`, `reconnecting`, `error`).
- Error text.
- RX/TX counters.
- Frame count.
- Endpoint buffer contents.
- Monitor/protocol/script frames.
- Resource ID observations.

Save or compare them as evidence, but do not use them as stable selectors for editing a graph.

## Demo IDs vs template IDs

Frontend demo workspace IDs come from `frontend/src/workspace/demoWorkspaces.ts`, for example:

- `serial-observability-demo`
- `remote-serial-demo`
- `full-workspace-demo`

MCP backend template IDs come from `internal/modules/mcpserver/server.go`, for example:

- `serial-observability-filter-logging`
- `serial-remote-raw-tcp`

These IDs may describe similar graph shapes, but they are not interchangeable. A frontend demo returns a workspace snapshot. An MCP template returns nodes/edges and usage examples for backend runtime tools.

## Guidance for agents

When editing or operating a graph:

1. Inspect provider catalog first.
2. Identify graph/node/edge by ID.
3. Preserve unrelated IDs.
4. Use exact handle IDs from the provider catalog.
5. Validate before start.
6. Query runtime evidence after actions.
7. Stop graph runtime after validation.
8. Avoid writing to physical/remote devices unless operator intent is explicit.

## Examples

### Safe node edit

Good: update node `graph-observability-filter-plain-abc123` config because that exact ID exists in the graph.

Avoid: update “the plain filter” by title without checking ID, because multiple filter nodes may exist.

### Safe resource reasoning

Good: treat `portweave-demo-observability-abc123` as a generated demo virtual resource name.

Avoid: hardcoding a generated demo suffix in docs or automation; each demo load generates fresh values.

### Safe runtime operation

Good: call `serial_graph_status` with the runtime graph ID returned/used at start.

Avoid: assuming a frontend demo ID such as `remote-serial-demo` is the running graph ID.
