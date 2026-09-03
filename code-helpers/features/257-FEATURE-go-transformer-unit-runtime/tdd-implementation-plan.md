# Issue #257 — TDD Implementation Plan

> Vertical TDD slices. Go applies transformers and runs **the same** unit `transformerTest`
> JSON as `miroir-test-app_deployment-miroir` (no copies). Vehicle: `go test` via the #256
> machine + `go/transformer` apply.
> Tracer: `miroirCoreTransformers` / `buildTransformerTests` / `constants` / `constantArray` /
> `resolve basic build transformer return value for empty Array` (`returnValue` → `[]`).
>
> **Execution model:** human-in-the-loop. No slice contains a commit step.

Analysis: [`./analysis.md`](./analysis.md) · Issue: https://github.com/miroir-framework/miroir/issues/257
Prerequisites: [`../255-FEATURE-go-jzod-typecheck/`](../255-FEATURE-go-jzod-typecheck/) · [`../256-FEATURE-go-miroirtest-unit-machine/`](../256-FEATURE-go-miroirtest-unit-machine/)
Working branch: `254-FEATURE-go-backend`

**Resume note:** Plan reviewed and realized (2026-09-03). All unit `transformerTest` leaves green on the same JSON.

---

## Scope

- `go/transformer`: `Apply` matching `transformer_extended_apply_wrapper` unit contract.
- Load TransformerDefinition JSON `a557419d-…`.
- #256 machine gains `transformerTest` (unit compare: `subExpectedValue` or `unitTestExpectedValue ?? expectedValue`).
- Suites: `33f60ac8-…` (243), `3aff508a-…` (42), plus smaller unit `transformerTest` suites in §3.3 of the analysis.

This plan does **not** run integ/SQL or `queryTest` / `actionTest` / `runnerTest`.

---

## Progress summary

| Slice | Title | Status | Primary proof |
|---|---|---|---|
| 0 | Lock defs + tracer leaf | ✅ | 45 defs; tracer path in `33f60ac8-…` |
| 1 | Tracer `returnValue` empty array | ✅ | that leaf via `RunSuite` |
| 2 | `jzodTypeCheck` suite on the machine | ✅ | 42 leaves `3aff508a-…` |
| 3 | Remaining `miroirCoreTransformers` | ✅ | 243 leaves, demand-driven handlers |
| 4 | Smaller unit `transformerTest` suites | ✅ | same JSON files |
| 5 | Nonreg, docs, AC | ✅ | folded into `unit-go-jzod` (`./...`) |

---

## Locked implementation defaults

| Decision | Choice |
|---|---|
| D1 | `go/transformer` + machine `transformerTest` |
| D2 | `transformer_extended_apply_wrapper` contract |
| D3 | Load definition JSON; library vs composite |
| D4 | `miroirCoreTransformers` + typecheck suite + other unit `transformerTest` files |
| D5 | Demand-driven handlers |
| D6 | `mustache`/`alterObject` CLI suites are #256 |
| D7 | Same JSON; passing TS unit leaf must pass on Go |
| D8 | Unit compare at `MiroirTransformerTestTools.ts:247-248` / `subExpectedValue` |

---

## Allocated UUIDs / keys

| Artefact | Value |
|---|---|
| TransformerDefinition entity | `a557419d-a288-4fb8-8a1e-971c86c113b8` |
| `miroirCoreTransformers` | `33f60ac8-6511-43b1-b153-6b86e3177532` |
| `jzodTypeCheck` suite | `3aff508a-8a9f-4384-ba50-cc696411eba5` |
| Nonreg | folded into `unit-go-jzod` (`go test -C go ./...`) |

---

## Test execution conventions

| Purpose | Command |
|---|---|
| Go | `go test -C go ./...` |
| TS oracle transformers | `npm run testMiroir -w miroir-core -- --suites miroirCoreTransformers --mode unit` |
| TS oracle typecheck | `npm run testMiroir -w miroir-core -- --suites jzodTypeCheck --mode unit` |
| Step gate | `./build-all.sh reset && npm run nonreg` |

Identify leaves by **document path**, not label.

---

## Slice 0 — Characterize defs + tracer

**Status:** ✅ DONE

### Goal

45 TransformerDefinition files; tracer leaf exists with `transformerType: "returnValue"`, `expectedValue: []`.

**Test:** `go/transformer/inventory_test.go`

### Validation

```bash
go test -C go ./transformer -run Inventory
```

### Realization

45 TransformerDefinition JSON files load in `go/transformer/defs.go`. Tracer leaf is document-order `buildTransformerTests` / `constants` / `constantArray` / empty-array `returnValue` in `33f60ac8-…`. Inventory is asserted by suite load + apply, not a separate `inventory_test.go`.

---

## Slice 1 — Tracer returnValue

**Status:** ✅ DONE

### Goal

`RunSuite` on `33f60ac8-…` filtered to the tracer leaf passes.

### 1.1 RED / 1.2 GREEN

`Apply` for `returnValue` / `handleTransformer_constant`. Machine `transformerTest` unit compare.

### Validation

```bash
go test -C go ./miroirtest ./transformer
```

### Realization

`ApplyUnit` + machine `transformerTest`: build first (`constantTransformer`), then runtime unless `elementType` is set. Tracer leaf returns `[]`.

---

## Slice 2 — jzodTypeCheck suite via machine

**Status:** ✅ DONE

### Goal

`3aff508a-…` 42 leaves pass through `transformerTest` (including `subExpectedValue`).

### Validation

```bash
go test -C go ./miroirtest -run TypeCheckSuite
```

### Realization

`3aff508a-…` 42 leaves pass through `TestUnitTransformerSuites` (including `subExpectedValue`). `jzodTypeCheck` handler matches the TS wrapper (`defaultMiroirModelEnvironment`).

---

## Slice 3 — Remaining miroirCoreTransformers

**Status:** ✅ DONE

### Goal

All 243 `transformerTest` leaves in `33f60ac8-…` pass. One cycle per failing `transformerType` family.

### Validation

```bash
go test -C go ./miroirtest -run CoreTransformers
```

### Realization

All 243 leaves of `33f60ac8-…` pass. Library handlers are demand-driven from failing `transformerType`s. Failures set `elementType: "failure"`. Unit compare uses `unitTestExpectedValue ?? expectedValue`.

---

## Slice 4 — Smaller unit transformer suites

**Status:** ✅ DONE

### Goal

Remaining analysis §3.3 unit `transformerTest` files pass on the same JSON (`defaultValueForMLSchema`, `unfoldSchemaOnce`, `resolveConditionalSchema`, `resolveSchemaReferenceInContext`, `adminTransformers`, `metaModelTransformersTest`, `menu_build`, `pilot_transformer_plus`, mixed `transformerResultSchema` transformer leaves).

### Validation

```bash
go test -C go ./miroirtest
```

### Realization

§3.3 files pass on the same JSON, including composite `entityDefinition_extractAttributes`, `resolveConditionalSchema` parentUuid errors, recursive `unfoldSchemaOnce`, and mixed `transformerResultSchema`.

---

## Slice 5 — Nonreg, docs, AC

**Status:** ✅ DONE

### 5.1 Nonreg

```json
{
  "id": "unit-go-transformer",
  "tier": "unit",
  "title": "Go transformer unit MiroirTests",
  "requires": "none",
  "argv": ["go", "test", "-C", "go", "./..."]
}
```

(If `unit-go-jzod` / `unit-go-miroirtest` already cover `./...`, fold rather than duplicate.)

### AC checklist (#257)

| Criterion | Proven by | Status |
|---|---|---|
| Same `transformerTest` JSON as TS unit | Slices 1–4 | ✅ |
| `miroirCoreTransformers` 243 leaves | Slice 3 | ✅ |
| `jzodTypeCheck` 42 leaves via machine | Slice 2 | ✅ |
| No integ/SQL | scope | ✅ |
| `./build-all.sh reset && npm run nonreg` | Slice 5 / epic Slice 4 | ✅ |

### Validation

```bash
go test -C go ./...
./build-all.sh reset && npm run nonreg
```

### Realization

Folded into `unit-go-jzod` (`go test -C go ./...`) instead of a duplicate `unit-go-transformer` step. Full reset+nonreg green: `test-results/nonreg/20260903T005406Z`.
