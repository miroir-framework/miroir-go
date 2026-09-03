# 256 — Go unit MiroirTest structure and execution machine

> Analysis: port the **unit** MiroirTest JSON shape and the **execution machine** needed to run
> leaves in Go. First suite: existing `mustache` JSON (`bdf83d4d-…`). Bootstrap self-parse stays
> the #255 `go test`. `transformerTest` execution is #257.

Related issue: https://github.com/miroir-framework/miroir/issues/256
Parent: [#254](https://github.com/miroir-framework/miroir/issues/254) · Prerequisite: [#255](https://github.com/miroir-framework/miroir/issues/255)
Related analyses: [`../254-FEATURE-go-backend/analysis.md`](../254-FEATURE-go-backend/analysis.md) · [`../255-FEATURE-go-jzod-typecheck/analysis.md`](../255-FEATURE-go-jzod-typecheck/analysis.md) · [`../257-FEATURE-go-transformer-unit-runtime/analysis.md`](../257-FEATURE-go-transformer-unit-runtime/analysis.md)

Key sources:
- MiroirTest Entity [`…/a311f363-e238-4203-bdfc-29e8c160c26b.json`](../../../packages/miroir-test-app_deployment-miroir/assets/miroir_model/16dbfe28-e1d7-4f20-9ba4-c1a9873202ad/a311f363-e238-4203-bdfc-29e8c160c26b.json)
- Instances dir [`…/miroir_data/a311f363-e238-4203-bdfc-29e8c160c26b/`](../../../packages/miroir-test-app_deployment-miroir/assets/miroir_data/a311f363-e238-4203-bdfc-29e8c160c26b/) (51 files)
- [`packages/miroir-core/src/5_tests/MiroirTestTools.ts`](../../../packages/miroir-core/src/5_tests/MiroirTestTools.ts) (`runMiroirTest`)
- [`packages/miroir-core/src/5_tests/FunctionCallTestTools.ts`](../../../packages/miroir-core/src/5_tests/FunctionCallTestTools.ts)
- [`packages/miroir-core/src/5_tests/FunctionCallTestRegistry.ts`](../../../packages/miroir-core/src/5_tests/FunctionCallTestRegistry.ts)
- [`packages/miroir-core/src/1_core/testing/test-expect.ts`](../../../packages/miroir-core/src/1_core/testing/test-expect.ts)
- [`docs/contributing/testing.md`](../../../docs/contributing/testing.md) · [`docs/reference/testing.md`](../../../docs/reference/testing.md)

**Document role:** analysis and architectural decision record.
**Status:** implemented and reviewed (2026-09-03). `go/miroirtest` runs all unit `functionCallTest` leaves from the same 51-file MiroirTest directory (364 leaves). No new / copied suite JSON.

---

## Decision record

| Decision | Choice |
|---|---|
| D1 — Package | **`go/miroirtest`** (loads JSON; calls `go/jzod` and a whitelist) |
| D2 — First test | **Existing `mustache` suite** uuid `bdf83d4d-…` (6 `functionCallTest` leaves, same JSON as TS). No new MiroirTest asset. Bootstrap self-parse stays the #255 `go test` proof. |
| D3 — Leaf types in scope | **`miroirTestSuite` + `functionCallTest`**. `queryTest` / `actionTest` / `runnerTest` fail closed. `transformerTest` is #257 (now implemented there). |
| D4 — Comparison | Match TS order: deserialize sentinels → `jsonify` / `unNullify` / `removeUndefinedProperties` → deep equal. Modes **used by existing unit leaves** are in scope (`expectedValue`, `expectedError`, `expectedAction2ErrorType`, `expectUndefinedResult`, `assertions[]` + `resultAccessPath`, environment/fixture refs). `subExpectedValue` is a `transformerTest` field (#257). |
| D5 — Function whitelist | **Same `{ module, export }` strings as TS** `FUNCTION_CALL_REGISTRY`. Go implements those exports as they appear in existing unit `functionCallTest` leaves. |
| D6 — Discovery | **Path / explicit file** in `go test`. Do not port `testMiroir` CLI, vitest `RUN_TEST`, or UI “Miroir Tests”. |
| D7 — Corpus | **Same JSON files** as `miroir-test-app_deployment-miroir`. #256 AC = all **unit** `functionCallTest` suites in that tree (mustache, alterObject, EntityPrimaryKey, tools, jzod helper suites, …). `transformerTest` suites are #257. No forked/copied tests. |

### D2 — First test shape

**Status:** Accepted — existing `mustache` suite `bdf83d4d-…` (D2-b′).

| Option | Mechanism | Pros | Cons |
|---|---|---|---|
| D2-a. New MiroirTest JSON | Suite with one `functionCallTest` over `TypeCheck` | Fast #255 seam | **Withdrawn** — not identical to existing assets |
| **D2-b′. Existing `mustache` JSON** ★ | `bdf83d4d-…` 6 `functionCallTest` leaves | Identical asset; smallest existing `functionCallTest` suite | Needs Go `extractDoubleBracePatterns` |
| D2-c. `pilot_transformer_plus` `4b18adc6-…` | Feature 196 / Phase 3a **practical** TS machine bootstrap (one `resolveConditionalSchema` `transformerTest`) | Already in the 39-key registry | Same problem as D2-b — transformer apply is #257 |
| D2-d. Empty-suite walk only | `miroirTestSuiteWalk.ts:110-120` registers `(empty suite)` when `miroirTests` is empty | Thinnest walk | Does not prove `functionCallTest` or #255 |
| D2-e. Hard-coded `go test` only | No JSON | Fast | No machine proof |

**Decision:** D2-b′ (existing `mustache` JSON). User 2026-09-03: Go MiroirTests must be **identical** to the deployment assets. D2-a (new bootstrap `functionCallTest`) is **withdrawn**.

**Handoff to #257:** `transformerTest` comparison (`subExpectedValue`, `unitTestExpectedValue ?? expectedValue`, ignore-attributes) is **not** this machine’s v1. #257 extends the comparison layer.

There is **no** existing instance named “bootstrap” (51 files, 2026-09-03). Feature 196’s `miroirTest_schema_pilot_empty` `cebb6dc8-…` is **absent**. Today’s TS machine pilot is `pilot_transformer_plus` — rejected as #256 first leaf (D2-c). Integ “bootstrap” (`IntegrationTestBootstrap.ts`) is a third word.

### D3 — Why not `transformerTest` here

`jzodTypeCheck_TransformerTestSuite` (42 leaves) and `miroirCoreTransformers` (243 leaves) are `transformerTest`. Running them is #257 (`transformer_extended_apply_wrapper` semantics). This issue only needs enough structure to recurse suites and invoke a whitelist function.

---

## 1. Goals

1. **Load a suite** — In order to share tests as a \<test author\>, I can point the Go machine at a `MiroirTest` JSON instance and have it see nested `miroirTests`.
2. **Run `functionCallTest`** — In order to prove a Go export as a \<runtime maintainer\>, I can dispatch `functionRef` + `arguments` and compare to `expectedValue`.
3. **Same assets** — In order to trust parity as a \<test author\>, I can run the existing unit `functionCallTest` JSON on Go without copying or rewriting those files.

## 2. Non-goals

- `transformerTest` / `queryTest` / `actionTest` / `runnerTest` execution (#257 or later).
- Integration mode, playfields, `TestConfiguration` (#252), UI launcher (#197).
- New MiroirTest JSON (withdrawn; identical existing assets only).
- `testMiroir` CLI parity.
- Full `FunctionCallTestRegistry` port (dozens of modules).

## 3. Current state

### 3.1 Entity and leaf types (aligned — enumerated)

Entity `MiroirTest` uuid `a311f363-e238-4203-bdfc-29e8c160c26b`. `mlSchema` discriminator `miroirTestType` literals (only these six):

`transformerTest` · `miroirTestSuite` · `functionCallTest` · `queryTest` · `runnerTest` · `actionTest`

All **51** instance files have root `miroirTestType: "miroirTestSuite"`. Walked leaves: **364** `functionCallTest` (25 FC-only files plus FC leaves in mixed files such as `transformerResultSchema` and `virtualAttributes`), **317** `transformerTest`, plus out-of-scope `actionTest` / `queryTest` / `runnerTest`. The **core unit registry** (`miroirCoreTestSuiteRegistry.ts` `MIROIR_TEST_SUITE_REGISTRY_NAMES`) has **39** keys. The other 12 (domain_controller_*, several runner_*, `evolutionTraceWP1`) are routed through standalone-app integ registries. Library runner instances live under `packages/miroir-test-app_deployment-library/assets/library_model/a311f363-…/` (same entity uuid, different deployment).

| Suite (name) | Uuid | Leaves (walk) |
|---|---|---|
| `jzodTypeCheck_TransformerTestSuite` | `3aff508a-…` | 42 `transformerTest` |
| `miroirCoreTransformers` | `33f60ac8-…` | 52 suite nodes, 243 `transformerTest` |
| `mustache` | `bdf83d4d-…` | 6 `functionCallTest` (`miroir-core/1_core/mustache` / `extractDoubleBracePatterns`) |
| `alterObject_atPath` | `d3b7f54f-…` | 3 `functionCallTest` (`miroir-core/tools` / `alterObjectAtPath`) |
| `EntityPrimaryKey` | `7c11632c-…` | 36 `functionCallTest` |
| `tools` | `e5940340-…` | 35 `functionCallTest` |
| domain_controller_* / runner_* / `queries_library` | various | `actionTest` / `runnerTest` / `queryTest` — out of scope |

JSON instance `name` ≠ CLI registry key for some suites (`alterObject` → `alterObject_atPath`, `menu` → `menu_build`, `jzodTypeCheck` → `jzodTypeCheck_TransformerTestSuite`, `metaModelTransformers` → `metaModelTransformersTest`). Go discovery uses file/uuid; CLI examples use registry keys.

### 3.2 `functionCallTest` dispatch (aligned)

Leaf dispatch is `runMiroirTest` in `MiroirTestTools.ts:154` (`switch (leaf.miroirTestType)`). `functionCallTest` in integration mode **throws** (`:226-229`). Runner: `runMiroirFunctionCallTestInMemory` → `FunctionCallTestTools.ts`.

`FunctionCallTestRegistry.ts` `resolveFunctionCallTarget`: whitelist `functionRef: { module, export }` only. Arbitrary paths are rejected.

`FunctionCallTestTools.ts` deserializes JSON sentinels before invoke: `__miroirJsonUndefined`, `__fixtureRef`, `__miroirJsonSet`, `__miroirMatchPattern`, `__miroirEnvironmentRef`. Leaf fields include `expectedError`, `expectedAction2ErrorType`, `expectUndefinedResult`, `assertions[]` + `resultAccessPath`, `ignoreAttributes`. #256 D4 starts with `expectedValue` equality only; sentinels are out unless the bootstrap leaf needs them (it should not).

`jzodTypeCheck.test.ts` is a **`transformerTest`** host (#257), not a `functionCallTest` example.

### 3.3 Execution hosts (aligned)

| Host | Path | Unit vs integ |
|---|---|---|
| Core vitest | `runMiroirCoreTestSuite` (e.g. `jzodTypeCheck.test.ts`) | unit |
| CLI | `npm run testMiroir -w miroir-core -- --suites <key> --mode unit` | unit |
| CLI integ | `testMiroir -w miroir-standalone-app --mode integration` | **out** |
| UI | Miroir Tests menu | unit today; integ #197 — **out** |

Transformer leaves use `MiroirTransformerTestTools.ts` → `transformer_extended_apply_wrapper` (`TransformersForRuntime.ts:4032`). That call is **#257**.

### 3.4 Comparison helpers (aligned)

`packages/miroir-core/src/1_core/testing/test-expect.ts` (`jsonify`, vitest-like `expect`). Transformer tests also use `subExpectedValue`, `ignoreAttributes`, `retainAttributes`, `unitTestExpectedValue` (`MiroirTransformerTestTools.ts`). #256 D4 starts with full `expectedValue` equality only.

### 3.5 Misaligned with “just run bootstrap”

No bootstrap suite file exists. Bootstrap self-parse remains #255 `go test`. #256 uses existing `functionCallTest` JSON (D2-b′).

## 4. Key reuse

| Piece | Location |
|-------|----------|
| MiroirTest Entity / `mlSchema` | `a311f363-e238-4203-bdfc-29e8c160c26b` |
| Bootstrap schema (argument) | `1e8dab4b-…` (#255) |
| Whitelist pattern | `FunctionCallTestRegistry.ts` |
| Expect / jsonify | `test-expect.ts` |

## 5. Proposals / options

| # | Proposal | Impact | Effort | Verdict |
|---|---|---|---|---|
| 1 | `go/miroirtest` + existing `functionCallTest` JSON | Unblocks #257; identical assets | Med | **adopt** |
| 2 | New bootstrap suite JSON | Forks the test corpus | Low | reject (user: identical MiroirTests) |
| 3 | Reuse `3aff508a-…` as first run | Pulls in transformer apply | Med | reject (owned by #257) |

---

## Next step

Implemented and reviewed. `transformerTest` execution is #257.
