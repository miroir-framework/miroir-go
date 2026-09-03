# 257 — Go transformer runtime and existing unit tests

> Analysis: port transformer **apply** and `transformerTest` execution so existing **unit**
> MiroirTest assets (same JSON as TS) pass on Go. Validity proof for #254. Not integ / SQL.

Related issue: https://github.com/miroir-framework/miroir/issues/257
Parent: [#254](https://github.com/miroir-framework/miroir/issues/254) · Prerequisites: [#255](https://github.com/miroir-framework/miroir/issues/255) (blocked until done), [#256](https://github.com/miroir-framework/miroir/issues/256) (blocked until done)
Related analyses: [`../254-FEATURE-go-backend/analysis.md`](../254-FEATURE-go-backend/analysis.md) · [`../255-FEATURE-go-jzod-typecheck/analysis.md`](../255-FEATURE-go-jzod-typecheck/analysis.md) · [`../256-FEATURE-go-miroirtest-unit-machine/analysis.md`](../256-FEATURE-go-miroirtest-unit-machine/analysis.md)

Key sources:
- TransformerDefinition Entity `a557419d-a288-4fb8-8a1e-971c86c113b8` · instances [`…/miroir_data/a557419d-…/`](../../../packages/miroir-test-app_deployment-miroir/assets/miroir_data/a557419d-a288-4fb8-8a1e-971c86c113b8/)
- [`packages/miroir-core/src/2_domain/Transformers.ts`](../../../packages/miroir-core/src/2_domain/Transformers.ts)
- [`packages/miroir-core/src/2_domain/TransformersForRuntime.ts`](../../../packages/miroir-core/src/2_domain/TransformersForRuntime.ts) (`transformer_extended_apply_wrapper` at line 4032)
- [`packages/miroir-core/src/5_tests/MiroirTransformerTestTools.ts`](../../../packages/miroir-core/src/5_tests/MiroirTransformerTestTools.ts)
- Suite `miroirCoreTransformers` `33f60ac8-6511-43b1-b153-6b86e3177532`
- Suite `jzodTypeCheck_TransformerTestSuite` `3aff508a-8a9f-4384-ba50-cc696411eba5` (42 leaves; #255 already used payloads)

**Document role:** analysis and architectural decision record.
**Status:** implemented and reviewed (2026-09-03). `go/transformer` apply + #256 `transformerTest` runner execute the same 317 unit `transformerTest` leaves as TS.

---

## Decision record

| Decision | Choice |
|---|---|
| D1 — Package | **`go/transformer`** apply + handlers; #256 machine gains a `transformerTest` leaf |
| D2 — Apply entry | **Port `transformer_extended_apply_wrapper` contract** (step `build`/`runtime`, params, context). Not `applyTransformerDEFUNCT`. |
| D3 — Definitions | **Load JSON** TransformerDefinition instances. Composite (`transformerImplementationType: "transformer"`) = apply the inner graph. Library = Go handler map keyed by `transformerType`. |
| D4 — In-scope unit suites | **`miroirCoreTransformers`** (243 leaves) is the **validity** suite. **`jzodTypeCheck_TransformerTestSuite`** (42) re-runs #255 via the machine. Smaller unit `transformerTest` suites (table §3.3) are in scope if they stay unit-only. |
| D5 — Handler order | **Demand-driven by failing leaves** of D4 suites. Unsupported `transformerType` fails closed. |
| D6 — `mustache` / `alterObject` CLI suites | **Out of AC** — they are `functionCallTest` helpers (#256 optional). `mustacheStringTemplate` **transformer** is in `miroirCoreTransformers`. |
| D7 — Parity | A failing TS unit test is not a Go bug. A passing TS unit test in D4 must pass on Go against the **same JSON**. |
| D8 — SQL / integ | **Out.** Unit compare is `unitTestExpectedValue ?? expectedValue` at `MiroirTransformerTestTools.ts:247-248`. Lines 82–84 are the **integration** helper `resolveTransformerIntegrationExpectedValue` only. |

### D4 — Which suite proves the epic

**Status:** Accepted — `miroirCoreTransformers` + typecheck suite via machine.

| Option | Mechanism | Pros | Cons |
|---|---|---|---|
| **D4-a. Core transformer unit corpus** ★ | Run `33f60ac8-…` + `3aff508a-…` | Matches “existing transformer unit tests” | 243+42 leaves; handlers grow over slices |
| D4-b. Only `mustache`+`alterObject` keys | Those 9 `functionCallTest`s | Small | Does **not** execute transformers |
| D4-c. All 51 MiroirTests | Everything | — | Includes `actionTest` / `queryTest` / integ-shaped runners |

**Decision:** D4-a. Tracer leaf (document order in `33f60ac8-…`): `buildTransformerTests` / `constants` / `constantArray` / `resolve basic build transformer return value for empty Array` — `transformerType: "returnValue"`, `runTestStep: "build"`, `expectedValue: []`. `jzodTypeCheck` `test010_literal` is the second machine proof (reuses #255 payloads). `pilot_transformer_plus` `4b18adc6-…` is a one-leaf `resolveConditionalSchema` suite — useful early #257 slice, not the epic validity gate.

---

## 1. Goals

1. **Apply a transformer** — In order to compute ML results as a \<Go runtime\>, I can apply a transformer JSON and get the same success/failure as TS unit apply.
2. **Run `transformerTest`** — In order to reuse assets as a \<test author\>, I can execute existing unit `transformerTest` JSON on the #256 machine.
3. **Validity** — In order to trust the Go approach as a \<platform maintainer\>, I can see the in-scope unit transformer suites pass on Go.

## 2. Non-goals

- `--mode integration`, SQL transformers, `miroirCoreTransformers` integ host (`-w miroir-standalone-app`).
- Query / action / runner tests.
- HTTP `miroir-server` replacement.
- New transformer features (#249, #250, #251) — consume **current** TS unit behavior.
- Porting `functionCallTest` helper suites as the epic proof (D6).

## 3. Current state

### 3.1 Definitions (aligned — enumerated 2026-09-03)

**45** files in `miroir_data/a557419d-a288-4fb8-8a1e-971c86c113b8/`.

| `transformerImplementationType` | Count |
|---|---|
| `libraryImplementation` | 43 |
| `transformer` (composite) | 2 (`entityDefinition_extractAttributes` `1bbed895-7d5a-4541-97bd-4d5cf22b128c`, `spreadSheetToJzodSchema` `e44300e8-ed02-40fb-a9ee-d83d08cb1f25`) |

Library handlers are keyed by `transformerImplementation.inMemoryImplementationFunctionName` (e.g. `mustacheStringTemplate` → `transformer_mustacheStringTemplate_apply`; `returnValue` → `handleTransformer_constant`). `listReducerToSpreadObject` uuid is `0894ed4f-ca11-4b04-878d-471d1d780fac` (not the same as `indexListBy` `8ddb7e2e-…`).

`Transformers.ts` re-exports each definition JSON. Dispatch: `applicationTransformerDefinitions` (`TransformersForRuntime.ts:781`) + `inMemoryTransformerImplementations` (`:722`) + `transformer_extended_apply` (`:3547`).

**No `transformerTest` oracle:** `getActiveDeployment` (`d554c31b-…`) and composite `spreadSheetToJzodSchema`. Out of AC unless a later slice adds tests; fail closed is enough.

`jzodTypeCheck` definition uuid `a3f7b5c2-1e8d-4a9b-9c7e-6f2d3e8a1b5c` — library implementation wrapping #255.

### 3.2 Apply path (aligned)

Unit transformer tests (`MiroirTransformerTestTools.ts:168-194`) call `transformer_extended_apply_wrapper` (`:4032`, default `resolveBuildTransformersTo = "constantTransformer"`), which wraps `transformer_extended_apply` (`:3547`). Unit host environment is `defaultMetaModelEnvironment` (`Model.ts:116`) — not the wrapper’s `defaultMiroirModelEnvironment` used only inside `jzodTypeCheckTransformer`.

`applyTransformerDEFUNCT` (`TransformersForRuntime.ts:4167`) is **not** the unit path.

### 3.3 Unit `transformerTest` suites (enumerated)

| Name | Uuid | `transformerTest` count | Notes |
|---|---|---|---|
| `miroirCoreTransformers` | `33f60ac8-6511-43b1-b153-6b86e3177532` | 243 | Validity corpus |
| `jzodTypeCheck_TransformerTestSuite` | `3aff508a-…` | 42 | Same payloads as #255 |
| `defaultValueForMLSchema` | `3d8570ba-…` | 11 | unit |
| `unfoldSchemaOnce` | `dd06922d-…` | 8 | unit |
| `resolveConditionalSchema` | `10bd8532-…` | 5 | unit; #255 D8 deferred engine |
| `resolveSchemaReferenceInContext` | `02a34783-…` | 3 | unit |
| `adminTransformers` | `8f07f7a2-…` | 1 | |
| `metaModelTransformersTest` | `a9a39db6-…` | 1 | CLI key `metaModelTransformers` |
| `menu_build` | `1a251573-…` | 1 | |
| `pilot_transformer_plus` | `4b18adc6-5cec-4abf-bb60-7a7fa26e4dc4` | 1 | Feature 196 machine pilot; `resolveConditionalSchema` build |
| `transformerResultSchema` | `0d3bd258-…` | 1 (+ 38 `functionCallTest`) | mixed |

`mustache` / `alterObject_atPath` are **not** in this table. Across these suites there are **~317** `transformerTest` leaves. **40 of 45** `applicationTransformerDefinitions` keys appear in at least one `transformerTest`; `getActiveDeployment` and `spreadSheetToJzodSchema` do not.

### 3.4 Comparison fields (aligned)

`MiroirTransformerTestTools.ts`: `expectedValue`, `unitTestExpectedValue`, `integrationTestExpectedValue`, `subExpectedValue`, `ignoreAttributes`, `retainAttributes`, `skip`, `runTestStep`. Go unit runner must implement the **unit** subset (D8).

### 3.5 Misaligned / debt for the plan

- `jzodTypeCheckTransformer` ignores the passed `queryParams` model environment and uses `defaultMiroirModelEnvironment` (`jzodTypeCheck.ts:2328`). Go should **match that bug** for unit parity, not “fix” it in this issue.
- Composite count is **2**, not a third implementation type.
- Label collisions inside `3aff508a-…` (duplicate `test010`… names) — runner must use path, not label, as identity.
- `getActiveDeployment` and `spreadSheetToJzodSchema` have no `transformerTest` — fail closed; not AC.

## 4. Key reuse

| Piece | Location |
|-------|----------|
| Apply | `transformer_extended_apply` `:3547` + wrapper `:4032` |
| Unit runner | `MiroirTransformerTestTools.ts` |
| 45 definitions | `a557419d-…` instances |
| Core suite | `33f60ac8-…` |
| Typecheck suite | `3aff508a-…` |
| #255 `TypeCheck` | `go/jzod` |
| #256 machine | `go/miroirtest` |

## 5. Proposals / options

| # | Proposal | Impact | Effort | Verdict |
|---|---|---|---|---|
| 1 | Demand-driven handlers + shared JSON | High | High | **adopt** |
| 2 | Reimplement only `mustacheStringTemplate` | Low | Low | reject as epic AC (too weak) |
| 3 | Generate handlers from TS | — | — | reject (no such generator) |

---

## Next step

Implemented and reviewed. Unit `transformerTest` JSON is identical to `miroir-test-app_deployment-miroir`.
