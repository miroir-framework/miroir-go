# 254 — Go backend to replace miroir-server (epic)

> Analysis and sequencing record for a Go port of the Miroir Meta-Language runtime, as the first
> increment toward replacing `miroir-server`. This document is the epic frame; implementation
> decisions live on the step analyses (#255, #256, #257).

Related issue: https://github.com/miroir-framework/miroir/issues/254
Sub-issues: [#255](https://github.com/miroir-framework/miroir/issues/255) · [#256](https://github.com/miroir-framework/miroir/issues/256) · [#257](https://github.com/miroir-framework/miroir/issues/257)
Related analyses: [`../255-FEATURE-go-jzod-typecheck/analysis.md`](../255-FEATURE-go-jzod-typecheck/analysis.md) · [`../256-FEATURE-go-miroirtest-unit-machine/analysis.md`](../256-FEATURE-go-miroirtest-unit-machine/analysis.md) · [`../257-FEATURE-go-transformer-unit-runtime/analysis.md`](../257-FEATURE-go-transformer-unit-runtime/analysis.md)

Key sources:
- [`packages/miroir-server/src/server.ts`](../../../packages/miroir-server/src/server.ts)
- [`packages/miroir-core/src/1_core/jzod/jzodTypeCheck.ts`](../../../packages/miroir-core/src/1_core/jzod/jzodTypeCheck.ts)
- [`packages/miroir-test-app_deployment-miroir/assets/miroir_data/5e81e1b9-38be-487c-b3e5-53796c57fccf/1e8dab4b-65a3-4686-922e-ce89a2d62aa9.json`](../../../packages/miroir-test-app_deployment-miroir/assets/miroir_data/5e81e1b9-38be-487c-b3e5-53796c57fccf/1e8dab4b-65a3-4686-922e-ce89a2d62aa9.json) (`jzodMiroirBootstrapSchema`)
- MiroirTest Entity `a311f363-e238-4203-bdfc-29e8c160c26b` · TransformerDefinition Entity `a557419d-a288-4fb8-8a1e-971c86c113b8`

**Document role:** analysis and sequencing / architectural decision record for the epic.
**Status:** implemented (2026-09-03). Analyses + TDD plans reviewed. #255–#257 done against the same MiroirTest / schema JSON as `miroir-test-app_deployment-miroir`.

---

## Sequencing

| Step | Issue | Title | Status |
|---|---|---|---|
| 1 | #255 | Go Jzod definitions + runtime typecheck | ✅ |
| 2 | #256 | Go unit MiroirTest structure + execution machine | ✅ |
| 3 | #257 | Go transformer runtime + existing **unit** transformer tests | ✅ |
| later | unscheduled | HTTP / DomainController / stores / MCP / SPA serving | after this epic |

Implementation: analysis → Composer review → TDD plan → Composer review → vertical slices. `./build-all.sh reset && npm run nonreg` at the end of each step (TS non-regression). Go tests join `scripts/nonreg-manifest.json` when a plan’s final slice says so.

---

## Decision record

| Decision | Choice |
|---|---|
| D1 — Language / horizon | **Go**, on a path that can replace `miroir-server`. This epic does **not** replace the process. |
| D2 — Module layout | **`go/` at repo root** (outside npm `workspaces: packages/*`) |
| D3 — What “port Jzod + typecheck” means | **Miroir `jzodTypeCheck`**, not `jzodToZod` / Zod |
| D4 — Schema source of truth | **Existing JSON** (`jzodMiroirBootstrapSchema`, MiroirTest instances, TransformerDefinitions). Go interprets; no schema fork. |
| D5 — First MiroirTest | **#255** bootstrap self-parse is `go test` on `jzodMiroirBootstrapSchema`. **#256** first suite is existing `mustache` JSON (`bdf83d4d-…`). No new MiroirTest asset. |
| D6 — Step 3 test scope | **Unit only** (`transformerTest` / `testMiroir --mode unit`). No integ / SQL. |
| D7 — Proof vehicle | **Same JSON assets** as TS. Step 1 may call `TypeCheck` from `go test` against those payloads before the #256 machine exists. |

**Rationale:** the Node server is a thin host (`server.ts` wires `miroir-core`, stores, MCP, CopilotKit). Replacing it without a Go ML runtime is theatre. The three steps are the smallest stack that can declare JSON, run a unit test, and execute transformers — the same assets TS already uses.

### D1 — Language / horizon

**Status:** Accepted — Go, epic stops before HTTP.

| Option | Mechanism | Pros | Cons |
|---|---|---|---|
| **D1-a. Go runtime, Node server stays** ★ | New Go module; `miroir-server` unchanged | Matches the request; reversible | Two runtimes until a later issue |
| D1-b. Replace `miroir-server` in this epic | Port REST + stores + MCP now | Faster “done” narrative | Unbounded; no typecheck/test proof |

**Decision:** D1-a. HTTP replacement is an explicit non-goal (later, unscheduled).

### D2 — Module layout

**Status:** Accepted — `go/` at repo root.

| Option | Mechanism | Pros | Cons |
|---|---|---|---|
| **D2-a. `go/`** ★ | `go.mod` at `go/`; packages `jzod`, `miroirtest`, `transformer` | npm `packages/*` does not swallow it; sibling to `packages/` | New top-level tree |
| D2-b. `packages/miroir-go` | Dir under `packages/` with `go.mod`, no `package.json` | npm ignores dirs without `package.json` (`miroir-designer` precedent) | Name says “package” while this epic is a runtime, not the HTTP host |
| D2-c. `packages/miroir-server-go` | Mirror of `miroir-server` | Good **later** name for the HTTP binary | Misleading for #255–#257 (no REST yet) |
| D2-d. Sibling repo (like `miroir-kotlin`) | Separate checkout | Isolation | User asked to proceed **in this repo** on a branch |

**Decision:** D2-a for this epic’s ML runtime (`go/jzod`, `go/miroirtest`, `go/transformer`). A later HTTP-host issue may add `go/cmd/server` or `packages/miroir-server-go` without moving the libraries. Working branch: `254-FEATURE-go-backend`.

### D3 — Port target for typecheck

**Status:** Accepted — Miroir `jzodTypeCheck` (#255).

Sibling `@miroir-framework/jzod` converts Jzod → Zod and parses with Zod (`jzodToZod` in `jzod/src/facade.ts`). Miroir’s runtime does **not** use that path for instance checking. It uses `jzodTypeCheck` (`packages/miroir-core/src/1_core/jzod/jzodTypeCheck.ts:871`), which returns `ResolvedJzodSchemaReturnType` (`status: "ok" | "error"` plus `resolvedSchema` / `keyMap`). The `jzodTypeCheck` **transformer** (`a3f7b5c2-1e8d-4a9b-9c7e-6f2d3e8a1b5c`) is a thin wrapper (`jzodTypeCheckTransformer` at `jzodTypeCheck.ts:2297`) that always passes `defaultMiroirModelEnvironment`.

A Go Zod-via-CGO or `jzodToZod` clone would not match transformer unit expected values (`status`, `rawSchema`, `resolvedSchema`, `keyMap`).

### D4–D7

Accepted as stated. Detail and rejected alternatives sit on the step analyses.

---

## 1. Goals

1. **Go ML foothold** — In order to eventually host Miroir without Node as a \<platform maintainer\>, I can run Jzod typecheck, unit MiroirTests, and unit transformers in Go against the same JSON as TypeScript.
2. **Shared assets** — In order to trust parity as a \<test author\>, I can keep writing MiroirTest / schema JSON once and have both runtimes execute it.
3. **TS safety** — In order not to regress the product as a \<release manager\>, I can run `./build-all.sh reset && npm run nonreg` after each step and see the existing suite stay green.

## 2. Non-goals

- Replacing or deleting `miroir-server` (later, unscheduled).
- Integration MiroirTests, SQL-compiled transformers, query/extractor/combiner engines (#256 / #257 non-goals, and later issues).
- Porting the full MiroirTest **corpus** in #256 (machine only; first leaf = bootstrap).
- jzod-ts-style generated Go types as a product (#255 non-goal).
- CopilotKit / MCP / persistence backends.

## 3. Current state

### 3.1 `miroir-server` (aligned with “host, not semantics”)

`packages/miroir-server/src/server.ts` starts Express on `rootApiUrl` (default **3080**), loads config, calls `miroirCoreStartup` and store startups, `setupMiroirDomainController`, `setupMcpServer` (`/mcp` on 3080 plus dedicated **4080**), CopilotKit at `/api/copilotkit`, and serves the built client in prod. Other host-only duties (still no semantics): wire REST CRUD via `restServerDefaultHandlers` (`server.ts` ~397–475), open admin + discovered deployment stores (~307–395), mount static `public/` separately from SPA `dist/client/`, optional TLS/HTTPS, and a separate MCP Express app/port. REST is the seven `restServerDefaultHandlers` routes in `packages/miroir-core/src/4_services/RestServer.ts` (`/CRUD/…`, `/action/:actionType`, `/query`, `/queryTemplate`). The client (`RestPersistenceClientAndRestClient`) uses that same contract for `realServer-*`; emulated tests use `RestClientStub`. Semantics live in `miroir-core`. There is no `go.mod` in this repo. npm workspaces are `packages/*` only. A later HTTP-host issue must implement those seven routes + `Action2ReturnType` JSON — **not** this epic.

### 3.2 Three TS surfaces the epic ports

| Surface | Authoritative artefact | TS execution | Count (2026-09-03, enumerated) |
|---|---|---|---|
| Jzod language | `jzodMiroirBootstrapSchema` uuid `1e8dab4b-65a3-4686-922e-ce89a2d62aa9` | `jzodTypeCheck` | 25 context keys; `jzodElement` union of 18 branches |
| Unit MiroirTest | Entity `a311f363-…`; **51** instance files; **39** keys in `MIROIR_TEST_SUITE_REGISTRY_NAMES` | `testMiroir --mode unit` → `runMiroirTests` | All 51 roots are `miroirTestSuite`. The other 12 are runner/action suites routed via standalone-app. |
| Transformers | 45 `TransformerDefinition` instances | `transformer_extended_apply_wrapper` | 43 `libraryImplementation`, 2 `transformer` (composite) |

Sibling `jzod` still owns `jzodBootstrapElementSchema` (`jzod/src/JzodInterface.ts`) and `jzodToZod`. That bootstrap is **not** the Miroir asset (different file, extra TS-only comments / `tag` / `metaSchema` shape). #255 uses the Miroir JSON.

### 3.3 “Bootstrap test” (name collision, resolved)

| Meaning | Where | Role in this epic |
|---|---|---|
| Jzod README self-parse | `jzodToZod(jzodBootstrapElementSchema).parse(...)` | Library demo; not a MiroirTest |
| Miroir bootstrap asset | `jzodMiroirBootstrapSchema` defaultLabel: *“Parses itself.”* | **#255** `go test` self-parse |
| Historical MiroirTest “bootstrap” | Feature 196 empty-suite `cebb6dc8-…` (**absent** today); practical stand-in `pilot_transformer_plus` `4b18adc6-…` (one `transformerTest` of `resolveConditionalSchema`) | **Not** #256’s first leaf — it is a `transformerTest` and belongs to #257 |
| Integ “bootstrap” | `IntegrationTestBootstrap.ts`, playfield reset | Out of epic (integ) |
| `jzodTypeCheck_TransformerTestSuite` | uuid `3aff508a-…`, **42** `transformerTest` leaves | Typecheck **corpus**; needs #257’s runner. #255 reuses the payloads via `go test`. |

#256 does **not** add a bootstrap `functionCallTest`. It runs existing unit `functionCallTest` JSON (tracer: `mustache`). `pilot_transformer_plus` is a `transformerTest` and belongs to #257.

### 3.4 `mustache` / `alterObject` vs transformer unit tests (misaligned with casual naming)

`AGENTS.md` examples `testMiroir --suites mustache,alterObject --mode unit` are **`functionCallTest`** suites (`extractDoubleBracePatterns`, `alterObjectAtPath`) — not `transformerTest`s of `mustacheStringTemplate`.

The large **transformer** unit corpus is `miroirCoreTransformers` uuid `33f60ac8-6511-43b1-b153-6b86e3177532` (**52 suite nodes** = 1 root + 51 nested, **243** `transformerTest` leaves). #257 owns that. #256 may register a few `functionCallTest` helpers; it does not port transformer apply.

## 4. Key reuse

| Piece | Location |
|-------|----------|
| Bootstrap schema JSON | uuid `1e8dab4b-65a3-4686-922e-ce89a2d62aa9` |
| Fundamental schema uuid (absolutePath in generated types) | `fe9b7d99-f216-44de-bb6e-60e1a1ebb739` (`miroirFundamentalJzodSchemaUuid`) |
| `jzodTypeCheck` | `jzodTypeCheck.ts:871` |
| Typecheck suite | `3aff508a-8a9f-4384-ba50-cc696411eba5` (42 leaves; first: `test010_literal`) |
| MiroirTest Entity | `a311f363-e238-4203-bdfc-29e8c160c26b` |
| `FunctionCallTestRegistry` | `packages/miroir-core/src/5_tests/FunctionCallTestRegistry.ts` |
| Transformer apply | `transformer_extended_apply_wrapper` (`TransformersForRuntime.ts:4032`) |
| Transformer unit suite | `miroirCoreTransformers` `33f60ac8-…` |

## 5. Proposals / options

| # | Proposal | Impact | Effort | Verdict |
|---|---|---|---|---|
| 1 | Interpret shared JSON in `go/` | High (enables #256/#257) | High | **adopt** |
| 2 | Generate Go structs from Jzod and stop | Low (cannot run tests) | Med | reject |
| 3 | Wrap Node/Zod from Go | Low (not a replacement) | Low | reject |

---

## Next step

Child plans are realized. Epic gate: `go test -C go ./...` and `./build-all.sh reset && npm run nonreg`.
