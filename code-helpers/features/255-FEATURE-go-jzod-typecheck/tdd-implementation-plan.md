# Issue #255 — TDD Implementation Plan

> Vertical TDD slices (RED → GREEN each). Tests exercise the public Go API `TypeCheck`
> against **real** applicative JSON (`jzodMiroirBootstrapSchema`, `jzodTypeCheck_TransformerTestSuite`
> leaves) — no fixture copies, no mocks.
> Vitest exception: there is no Go MiroirTest machine yet (#256). Vehicle is `go test`.
> Tracer: `test010_literal` payload from suite `3aff508a-…` returns `status: "ok"`.
>
> **Execution model:** human-in-the-loop. No slice contains a commit step — commits happen
> only when the user explicitly asks. Each slice ends with its Validation commands; on
> success its Realization summary is appended and its Status flips to ✅ DONE.

Analysis: [`./analysis.md`](./analysis.md) · Issue: https://github.com/miroir-framework/miroir/issues/255
Prerequisite: none (epic [`../254-FEATURE-go-backend/analysis.md`](../254-FEATURE-go-backend/analysis.md))
Working branch: `254-FEATURE-go-backend`

**Resume note:** Plan reviewed (2026-09-03). Slices 0–7 ✅. Slice 8 docs + `unit-go-jzod` done; full reset+nonreg is #254 Slice 4.

---

## Scope

- Go module `go/` + package `go/jzod`.
- Load Miroir bootstrap JSON; `TypeCheck` with `jzodTypeCheck` result shape.
- Prove using suite `3aff508a-…` leaf payloads and bootstrap self-parse.

This plan does **not** add a MiroirTest runner (#256) or transformer apply (#257).

---

## Progress summary

| Slice | Title | Status | Primary proof |
|---|---|---|---|
| 0 | Lock bootstrap + typecheck-suite contracts | ✅ | `go/jzod` inventory tests |
| 1 | Tracer: literal typecheck from suite JSON | ✅ | `test010_literal` via `TypeCheck` |
| 2 | Primitives used by the suite (string, boolean) | ✅ | `test020_string`, `test022_boolean_true`, `test024_boolean_false` |
| 3 | Local `schemaReference` + simple object | ✅ | `test030_schemaReference`, `test040_simple_object` |
| 4 | Unions | ✅ | `test050_object_with_union`, `test120_simple_union` |
| 5 | `any` + `valueToJzod` sliver | ✅ | `test130_any_*` … `test182_any_empty_object` |
| 6 | Remaining suite leaves (grouped cycles) | ✅ | all 42 leaves `status`+`resolvedSchema` (and `keyMap` where the JSON requires it) |
| 7 | Bootstrap self-parse | ✅ | `TypeCheck(bootstrap, bootstrap)` ok |
| 8 | Nonreg, docs, cleanup, AC | ✅ | `unit-go-jzod` + `./build-all.sh reset && npm run nonreg` |

---

## Locked implementation defaults

| Decision | Choice |
|---|---|
| D1 — Package | `go/jzod` in module `go/` |
| D2 — Semantics | Full `ResolvedJzodSchemaReturnType` (`status`, `rawSchema`, `resolvedSchema`, `valuePath`, `typePath`, `schemaReferenceName`, `keyMap`; errors also `error`, `rawJzodSchemaType`, `value`) |
| D3 — Schema source | JSON `1e8dab4b-65a3-4686-922e-ce89a2d62aa9` |
| D4 — Representation | `encoding/json` / `any` + accessors |
| D5 — Proof | `go test` on real suite JSON |
| D6 — Scope | Implement what proofs need. TS stub-accept types (`undefined`, `never`, `unknown`, `void`, `intersection`, `promise`, `set`, `function`, `map`, `lazy` at `jzodTypeCheck.ts:2241-2270`) stub-accept in Go. Fail closed only for a `type` string TS does not handle. |
| D7 — Environment | Register bootstrap `MlSchema` `1e8dab4b-…` in the absolute-reference list (Slice 7). `defaultMiroirModelEnvironment` cannot resolve that uuid. Suite `3aff508a-…` uses no `absolutePath`. Do not load `fe9b7d99-…` unless a reused leaf’s `absolutePath` is that uuid. |
| D8 — Conditional schema | Skip unless `currentDefaultValue` **and** `reduxDeploymentsState` are present (`jzodTypeCheck.ts:941-955`). Empty `currentValuePath: []` is truthy in JS and does not skip. |
| D9 — `valueToJzod` | Sliver for `any`: `buildAnyObjectEntry` **and** `buildAnySubnodeKeyMap` |

---

## Allocated UUIDs / keys

| Artefact | Value |
|---|---|
| Bootstrap `MlSchema` (existing) | `1e8dab4b-65a3-4686-922e-ce89a2d62aa9` |
| Typecheck suite (existing) | `3aff508a-8a9f-4384-ba50-cc696411eba5` |
| Go module path | `github.com/miroir-framework/miroir/go` |
| Nonreg step | `unit-go-jzod` |

No new model uuids in this issue.

---

## Test execution conventions

| Purpose | Command |
|---|---|
| Go jzod tests | `go test ./...` from `go/` |
| One package | `go test ./jzod` from `go/` |
| TS typecheck suite (parity oracle, not the Go vehicle) | `npm run testMiroir -w miroir-core -- --suites jzodTypeCheck --mode unit` |
| Step gate | `./build-all.sh reset && npm run nonreg` |

**Vitest exception:** `go test` — not reachable through MiroirTest on Go until #256.

Suite JSON path (from `go/jzod`, repo-relative):

`../../packages/miroir-test-app_deployment-miroir/assets/miroir_data/a311f363-e238-4203-bdfc-29e8c160c26b/3aff508a-8a9f-4384-ba50-cc696411eba5.json`

Bootstrap JSON:

`../../packages/miroir-test-app_deployment-miroir/assets/miroir_data/5e81e1b9-38be-487c-b3e5-53796c57fccf/1e8dab4b-65a3-4686-922e-ce89a2d62aa9.json`

Helper: load suite, collect `transformerTest` leaves in document order, pick by **index** (labels prefix-collide after `test182_any_empty_object`; `test010` ≠ `test010_literal`). Identity is document order + parent path, not label.

**42-leaf index → `miroirTestLabel`** (document order; Slice 0 locks this table):

| Index | Label | Notes |
|------:|-------|-------|
| 0 | `test010_literal` | Slice 1 tracer |
| 1 | `test020_string` | Slice 2 |
| 2 | `test022_boolean_true` | Slice 2 |
| 3 | `test024_boolean_false` | Slice 2 |
| 4 | `test030_schemaReference` | Slice 3 |
| 5 | `test040_simple_object` | Slice 3 |
| 6 | `test050_object_with_union` | Slice 4 |
| 7 | `test060_recursive_object` | Slice 6 |
| 8 | `test070` | Slice 6 |
| 9 | `test120_simple_union` | Slice 4 |
| 10–17 | `test130_any_string` … `test182_any_empty_object` | Slice 5 (`test150`/`test160` = indices 12–13) |
| 18–41 | second series `test010`…`test022`, skip `test023`, then `test024`…`test030`, `test100`, `test110`, `test120`, `test130` | Slice 6 |

TS runner prefers `subExpectedValue` path assertions when present (`MiroirTransformerTestTools.ts:224-245`). All 42 leaves have `subExpectedValue`. Indices **12** (`test150_any_object`) and **13** (`test160_any_object_of_object`) have **no** `expectedValue` — Go must assert those dotted paths, not full-object equality. Other leaves: compare `subExpectedValue` paths for TS-oracle parity; `expectedValue` is the deeper Go target (includes `keyMap` / paths).

Required TS callees to port when a leaf needs them: `resolveJzodSchemaReferenceInContext`, `recursiveResolveJzodSchemaReferenceInContext`, `jzodUnion_recursivelyUnfold`, `jzodObjectFlatten` (required — called from `jzodTypeCheck.ts:248`, `:1057`). Discriminated object unions also use `getObjectUnionDiscriminatorValuesFromResolvedSchema` (`jzodTypeCheck.ts:1537`) if a leaf needs them.

TS stub-accept (`jzodTypeCheck.ts:2254-2260`) returns top-level `typePath: []` while `keyMap` entries keep `currentTypePath` — match that quirk on Slice 7.

---

## Slice 0 — Characterize bootstrap + suite contracts

**Status:** ✅ DONE

### Goal

Lock the applicative files #255 interprets so later slices cannot silently retarget sibling `jzod` or the fundamental schema uuid.

### 0.1 RED → GREEN — inventory

**Test:** `go/jzod/inventory_test.go` (justified `go test`)

Behavior asserted:
- Bootstrap file `name == "jzodMiroirBootstrapSchema"`, `uuid == 1e8dab4b-…`
- Context has exactly these **25** keys (analysis §3 / D3): `jzodArray`, `jzodAttributeDateValidations`, `jzodAttributeNumberValidations`, `jzodAttributePlainDateWithValidations`, `jzodAttributePlainNumberWithValidations`, `jzodAttributePlainStringWithValidations`, `jzodAttributeStringValidations`, `jzodBaseObject`, `jzodElement`, `jzodEnum`, `jzodEnumAttributeTypes`, `jzodEnumElementTypes`, `jzodFunction`, `jzodIntersection`, `jzodLazy`, `jzodLiteral`, `jzodMap`, `jzodObject`, `jzodPlainAttribute`, `jzodPromise`, `jzodRecord`, `jzodReference`, `jzodSet`, `jzodTuple`, `jzodUnion`
- `jzodElement` union has **18** `schemaReference` branches in file order: `jzodArray`, `jzodPlainAttribute`, `jzodAttributePlainDateWithValidations`, `jzodAttributePlainNumberWithValidations`, `jzodAttributePlainStringWithValidations`, `jzodEnum`, `jzodFunction`, `jzodLazy`, `jzodLiteral`, `jzodIntersection`, `jzodMap`, `jzodObject`, `jzodPromise`, `jzodRecord`, `jzodReference`, `jzodSet`, `jzodTuple`, `jzodUnion`
- Typecheck suite file has exactly **42** `transformerTest` leaves; the index→label table in Test execution conventions (0 = `test010_literal` … 41 = `test130`)

### Validation

```bash
cd go && go test ./jzod -run Inventory
```

### Realization

`go/go.mod` (`github.com/miroir-framework/miroir/go`) and `go/jzod/inventory_test.go` lock bootstrap `1e8dab4b-…` (25 context keys, 18 `jzodElement` branches) and the 42-leaf index→label table. `go test -C go ./jzod -run Inventory` green. No `TypeCheck` yet.

---

## Slice 1 — Tracer: literal typecheck from suite JSON

**Status:** ✅ DONE

### Goal

A Go caller can typecheck the `test010_literal` payload and get `status: "ok"` with matching `resolvedSchema`.

**Layers cut:** suite JSON → `TypeCheck` public API

### 1.1 RED

**Test:** `go/jzod/typecheck_suite_test.go` — leaf index 0

Behavior asserted:
- `TypeCheck(mlSchema, valueObject, empty paths, empty context)` → `status == "ok"`
- `resolvedSchema` equals `{ "type": "literal", "definition": "myLiteral" }`
- `rawSchema` / `valuePath` / `typePath` match index-0 `expectedValue` (and `subExpectedValue` paths: `status`, `rawSchema`, `resolvedSchema` — the TS oracle)

`expectedValue` also has `keyMap[""]`; Slice 1 may defer full `keyMap` equality (analysis D2-a deepening).

### 1.2 GREEN

`TypeCheck` handles `type: "literal"` only. A `type` string TS does not handle returns `status: "error"` (unsupported). Do **not** fail closed on TS stub-accept types if they appear as the top-level schema — echo `status: "ok"` (D6). Slice 1 need not implement those stubs until a later leaf or Slice 7 hits them.

### 1.3 Refactor checkpoint

- Shared JSON loader for suite leaves (used by later slices).

### Validation

```bash
cd go && go test ./jzod -run 'Inventory|Literal'
```

### Realization

`TypeCheck` + `Result` in `go/jzod/typecheck.go`. Leaf 0 (`test010_literal`) matches `status` / `resolvedSchema` / `rawSchema` / paths. Unsupported `type` errors. `go test -C go ./jzod -run 'Inventory|Literal|Unsupported'` green.

---

## Slice 2 — string and boolean

**Status:** ✅ DONE

### Goal

Suite leaves `test020_string`, `test022_boolean_true`, `test024_boolean_false` pass `status` + `resolvedSchema`.

**Layers cut:** same

### 2.1 RED

**Test:** same file; those three leaves by index (1–3).

### 2.2 GREEN

Plain-attribute cases `string` / `boolean`. A mismatched value (e.g. number vs string) errors — one extra assertion using the string leaf’s schema and a number value (not a new fixture file).

### 2.3 Refactor checkpoint

- Factor primitive dispatch.

### Validation

```bash
cd go && go test ./jzod
```

### Realization

`string` / `boolean` cases; leaf 1–3 green; number vs string schema errors. `go test -C go ./jzod` green.

---

## Slice 3 — local schemaReference + simple object

**Status:** ✅ DONE

### Goal

`test030_schemaReference` and `test040_simple_object` pass.

**Layers cut:** JSON → reference resolver → `TypeCheck`

### 3.1 RED

**Test:** those two leaves (indices 4–5).

Behavior asserted:
- `test030` (index 4): `resolvedSchema.type == "string"`; `typePath` includes `ref:a`; `schemaReferenceName == "a"` (in `expectedValue`)
- `test040` (index 5): object attributes match expected `resolvedSchema`

### 3.2 GREEN

`resolveReference` for `relativePath` + local `context`. Object: each definition attribute vs value key; extra/missing required keys error.

### 3.3 Refactor checkpoint

- Keep resolver a deep module; do not export unless a test needs it.

### Validation

```bash
cd go && go test ./jzod
```

### Realization

Local `schemaReference` (relativePath + schema `context`), `object` (optional attrs, present-keys resolvedSchema), `number`, and first-match `union` unfold. Leaves 4–5 green.

---

## Slice 4 — unions

**Status:** ✅ DONE

### Goal

`test050_object_with_union` and `test120_simple_union` pass.

### 4.1 RED

**Test:** those leaves by index: **6** (`test050_object_with_union`), **9** (`test120_simple_union`). Do not use colliding labels.

### 4.2 GREEN

Unfold union branches; pick the first matching branch (match TS order). Discriminator support only if those leaves need it.

### 4.3 Refactor checkpoint

- Share unfold with later recursive/bootstrap unions.

### Validation

```bash
cd go && go test ./jzod
```

### Realization

Object-valued union branches TypeCheck the chosen object schema (leaf 6). Simple `union(string, number)` (leaf 9) already passed from Slice 3 unfold. `go test -C go ./jzod` green.

---

## Slice 5 — `any` + valueToJzod sliver

**Status:** ✅ DONE

### Goal

`any` leaves indices **10–17** (`test130_any_string` through `test182_any_empty_object`) pass the assertions the JSON actually makes (`status` / `resolvedSchema` / `keyMap` as required per leaf). Indices **12–13** (`test150_any_object`, `test160_any_object_of_object`) have `subExpectedValue` only (dotted paths e.g. `keyMap.a.rawSchema`) — assert those paths, not full-object `expectedValue`.

### 5.1 RED

**Test:** leaves by index 10–17.

### 5.2 GREEN

`type: "any"` accepts values; `resolvedSchema` from a `valueToJzod` sliver (sibling `jzod/src/valueToJzod.ts` behavior for the value kinds those leaves use). Null/`any` follows `jzodTypeCheck.ts:897-936`.

### 5.3 Refactor checkpoint

- Do not import the rest of sibling jzod.

### Validation

```bash
cd go && go test ./jzod
```

### Realization

`type: "any"` via `valueToJzod` + `buildAnySubnodeKeyMap`. Null/`any` early-accept. Leaves 10–17 asserted via `subExpectedValue` (12–13 have no `expectedValue`). `go test -C go ./jzod` green.

---

## Slice 6 — Remaining typecheck suite leaves

**Status:** ✅ DONE

### Goal

All **42** leaves of `3aff508a-…` pass at least `status` + `resolvedSchema` + `rawSchema` + paths (full `expectedValue` / `keyMap` when the leaf asserts them). Suite leaves use **no** `absolutePath`; if a leaf unexpectedly requires `fe9b7d99-…` / `reduxDeploymentsState`, record it in Realization and do **not** generate carry-on.

**Layers cut:** remaining constructors (`enum`, `record`, `array`, `extend`, recursive refs, …) plus required callees (`recursiveResolve…`, `jzodObjectFlatten`).

### 6.1 RED

**Test:** table-driven all 42 leaves **by index**. One cycle per failing constructor family (not one slice per function).

### 6.2 GREEN

Implement only what the next failing leaf needs. A `type` string TS does not handle fails closed. TS stub-accept types must stub-accept (D6), not fail closed.

### 6.3 Refactor checkpoint

- Analysis misalignment: sibling `jzodToZod` stays unused. Dead experimental APIs removed.

### Validation

```bash
cd go && go test ./jzod
```

### Realization

Table-driven all 42 leaves via `subExpectedValue`. Added `record`, `uuid` (v4), `date` (invalid-string error text), `bigint` mismatch, extra-key object error text, stub-accept types. `go test -C go ./jzod` green.

---

## Slice 7 — Bootstrap self-parse

**Status:** ✅ DONE

### Goal

`TypeCheck(jzodMiroirBootstrapSchema.definition, jzodMiroirBootstrapSchema.definition)` with bootstrap `MlSchema` `1e8dab4b-…` **registered in the env lookup chain** (same shape as `absoluteReferences.find(s => s.uuid == absolutePath)?.definition.context[relativePath]`) returns `status: "ok"`. This is a **new** proof — TS has no `jzodTypeCheck` self-parse test.

Do **not** use `defaultMiroirModelEnvironment` (its `jzodSchemas` is `[]`). Do **not** load `fe9b7d99-…`; six occurrences in `jzodBaseObject.tag` editor metadata are annotations (`tag` values simple / nested transformers `type: "any"`).

**Layers cut:** bootstrap JSON → registered `MlSchema` env → `TypeCheck`

### 7.1 RED

**Test:** `go/jzod/bootstrap_self_parse_test.go`

### 7.2 GREEN

Register `{ uuid: 1e8dab4b-…, definition.context }` in the absolute-reference list; resolve `absolutePath` + `relativePath`; object `extend` of `jzodBaseObject`; discriminated `jzodElement` union. Stub-accept (`status: "ok"`, schema echoed) for `undefined`, `never`, `unknown`, `void`, `intersection`, `promise`, `set`, `function`, `map`, `lazy` (`jzodTypeCheck.ts:2241-2270`) when those constructors appear as the schema under check. Schema-only constructors need only to **typecheck as values of the meta-schema**.

### 7.3 Refactor checkpoint

- Deepen resolver; no new public surface.

### Validation

```bash
cd go && go test ./jzod
```

### Realization

Registered `1e8dab4b-…` on `ModelEnvironment.AbsoluteSchemas`. `TypeCheck(definition, definition)` ok. Object `extend` flatten, `enum` / `array` / `tuple`, stub-accept types. `go test -C go ./jzod` green.

---

## Slice 8 — Nonreg, docs, cleanup, AC

**Status:** ✅ DONE

### 8.1 Nonreg

- Add `unit-go-jzod` to `scripts/nonreg-manifest.json`. `run-nonreg.py` always uses repo-root cwd (no `cwd` field). Use:

```json
{
  "id": "unit-go-jzod",
  "tier": "unit",
  "title": "Go jzod TypeCheck unit tests",
  "requires": "none",
  "argv": ["go", "test", "-C", "go", "./..."]
}
```

### 8.2 Docs

- `analysis.md` status → implemented; AGENTS.md / `docs/contributing/testing.md` one line: Go jzod tests live under `go/`.
- Epic analysis sequencing: #255 ✅.

### 8.3 Issue-directory cleanup

- No `tests/**/issues/255-*` vitest dir expected. If any appeared, delete after migrating assertions into `go/jzod`.

### 8.4 Tracer bullet (narrative)

1. Load suite JSON leaf 0 (`test010_literal`).
2. `TypeCheck` in Go → `ok`.
3. Load bootstrap JSON; typecheck definition against itself → `ok`.

Automated equivalent: `go test ./jzod`.

### AC checklist (#255)

| Criterion | Proven by | Status |
|---|---|---|
| Go module in-repo at `go/` + `go/jzod` | `go/go.mod` | ✅ |
| `JzodElement` JSON from Miroir bootstrap loads | Slice 0 | ✅ |
| Typecheck accepts/rejects for in-scope constructors | Slices 1–6 | ✅ |
| Bootstrap self-parse succeeds | Slice 7 | ✅ |
| `schemaReference` + object `extend` for bootstrap | Slice 7 | ✅ |
| Analysis + this plan under `code-helpers/features/255-…/` | files | ✅ |
| `./build-all.sh reset && npm run nonreg` green | Slice 8 / epic Slice 4 | ✅ |

### Validation

```bash
cd go && go test ./...
./build-all.sh reset && npm run nonreg
```

### Realization

`unit-go-jzod` is in `scripts/nonreg-manifest.json` (`go test -C go ./...`). AGENTS.md and `docs/contributing/testing.md` document the Go module. No `tests/**/issues/255-*` vitest dir. Full reset+nonreg green: `test-results/nonreg/20260903T005406Z`.
