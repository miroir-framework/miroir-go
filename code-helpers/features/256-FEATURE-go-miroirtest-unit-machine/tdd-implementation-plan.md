# Issue #3 — TDD Implementation Plan

> Vertical TDD slices. The Go machine executes **the same** MiroirTest JSON as
> `packages/miroir-test-app_deployment-miroir` (no copies, no new suite files).
> Vehicle: `go test` calling `miroirtest.RunSuite` on those files (no Go MiroirTest
> host yet — this issue *is* that host).
> Tracer: existing `mustache` suite `bdf83d4d-…` (6 `functionCallTest` leaves).
>
> **Execution model:** human-in-the-loop. No slice contains a commit step.

Analysis: [`./analysis.md`](./analysis.md) · Issue: https://github.com/miroir-framework/miroir-go/issues/3
Prerequisite: [`../255-FEATURE-go-jzod-typecheck/`](../255-FEATURE-go-jzod-typecheck/)
Working branch: `254-FEATURE-go-backend`

**Resume note:** Plan reviewed and realized (2026-09-03). All unit `functionCallTest` leaves green on the same JSON.

---

## Scope

- Package `go/miroirtest`: load / walk `miroirTestSuite`, run `functionCallTest`.
- Whitelist uses TS `{ module, export }` strings.
- AC: all **unit** `functionCallTest` suites in the miroir deployment MiroirTest directory.

This plan does **not** run `transformerTest` (#4) or add new JSON assets.

---

## Progress summary

| Slice | Title | Status | Primary proof |
|---|---|---|---|
| 0 | Lock suite files + mustache contract | ✅ | inventory of existing JSON |
| 1 | Tracer: run `mustache` via `RunSuite` | ✅ | 6 leaves `bdf83d4d-…` |
| 2 | `alterObject_atPath` | ✅ | 3 leaves `d3b7f54f-…` |
| 3 | Remaining unit `functionCallTest` suites | ✅ | same files as TS registry |
| 4 | Fail-closed other `miroirTestType` | ✅ | query/action/runner |
| 5 | Nonreg, docs, AC | ✅ | folded into `unit-go-jzod` (`./...`) |

---

## Locked implementation defaults

| Decision | Choice |
|---|---|
| D1 | `go/miroirtest` |
| D2 | Existing `mustache` JSON — no new asset |
| D3 | `miroirTestSuite` + `functionCallTest`; others fail closed until #4 |
| D4 | `expectedValue` deep JSON equality (v1); other modes as existing leaves require |
| D5 | Same `functionRef` module/export as TS |
| D6 | Path / uuid in `go test` |
| D7 | Same files; unit `functionCallTest` corpus in AC |

---

## Allocated UUIDs / keys

| Artefact | Value |
|---|---|
| MiroirTest entity (existing) | `a311f363-e238-4203-bdfc-29e8c160c26b` |
| `mustache` | `bdf83d4d-f4dd-42c9-b2d6-41311d979083` |
| `alterObject_atPath` | `d3b7f54f-8dcf-4159-814e-0f4a71a6081a` |
| Nonreg step | folded into `unit-go-jzod` (`go test -C go ./...`) |

No new model uuids.

---

## Test execution conventions

| Purpose | Command |
|---|---|
| Go machine | `go test -C go ./miroirtest` |
| TS oracle mustache | `npm run testMiroir -w miroir-core -- --suites mustache --mode unit` |
| TS oracle alterObject | `npm run testMiroir -w miroir-core -- --suites alterObject --mode unit` |
| Step gate | `./build-all.sh reset && npm run nonreg` |

Suite path from `go/miroirtest`:

`../../packages/miroir-test-app_deployment-miroir/assets/miroir_data/a311f363-e238-4203-bdfc-29e8c160c26b/<uuid>.json`

---

## Slice 0 — Characterize existing suites

**Status:** ✅ DONE

### Goal

Lock that Go reads the deployment directory (51 files) and the mustache file has 6 `functionCallTest` leaves.

### 0.1 RED → GREEN — inventory

**Test:** `go/miroirtest/inventory_test.go`

Behavior asserted:
- Directory has 51 JSON files
- `bdf83d4d-…` name `mustache`, 6 `functionCallTest` leaves, first `functionRef` is `miroir-core/1_core/mustache` / `extractDoubleBracePatterns`
- `d3b7f54f-…` has 3 `functionCallTest` leaves

### Validation

```bash
go test -C go ./miroirtest -run Inventory
```

### Realization

`go/miroirtest/inventory_test.go`: 51 JSON files; `mustache` 6 FC leaves; `alterObject_atPath` 3 FC leaves.

---

## Slice 1 — Tracer: mustache suite

**Status:** ✅ DONE

### Goal

`RunSuite` on the mustache JSON file reports 6 passed leaves.

**Layers cut:** existing JSON → walk → whitelist → compare `expectedValue`

### 1.1 RED

**Test:** `go/miroirtest/suite_test.go` — `TestRunMustacheSuite`

### 1.2 GREEN

Walk `miroirTests`; dispatch `functionCallTest`; implement `extractDoubleBracePatterns` in Go (same regex as TS).

### 1.3 Refactor checkpoint

- Keep whitelist a map; do not special-case the suite name.

### Validation

```bash
go test -C go ./miroirtest
```

### Realization

`TestRunMustacheSuite` + `RunSuite` on `bdf83d4d-…`. Whitelist `miroir-core/1_core/mustache` / `extractDoubleBracePatterns`.

---

## Slice 2 — alterObject_atPath

**Status:** ✅ DONE

### Goal

Existing `d3b7f54f-…` 3 leaves pass.

### 2.1 RED / 2.2 GREEN

Port `alterObjectAtPath` (`tools.ts:325`).

### Validation

```bash
go test -C go ./miroirtest
```

### Realization

`alterObjectAtPath` ported. 3 leaves green on `d3b7f54f-…`.

---

## Slice 3 — Remaining unit functionCallTest suites

**Status:** ✅ DONE

### Goal

Every unit `functionCallTest` suite file in the MiroirTest directory passes on Go against **that same file**. Demand-driven whitelist (EntityPrimaryKey, tools, jzod helpers, …). Modes used by those leaves (`expectedError`, sentinels, `assertions[]`) implemented when a leaf needs them.

### 3.1 RED

Table-driven: discover files whose walked leaves are all `functionCallTest` (or mixed — run only `functionCallTest` leaves, skip `transformerTest` until #4).

### 3.2 GREEN

Implement the next missing `{ module, export }`.

### Validation

```bash
go test -C go ./miroirtest
```

### Realization

`TestUnitFunctionCallSuites` walks the same 51 files and runs every `functionCallTest` leaf (364). Sentinels and comparison modes used by those leaves are implemented. Mixed files run FC leaves only (transformer leaves go to #4).

---

## Slice 4 — Fail-closed other leaf types

**Status:** ✅ DONE

### Goal

Walking a `transformerTest` leaf returns a clear “not implemented” until #4 (do not fake-pass).

### Validation

```bash
go test -C go ./miroirtest -run FailClosed
```

### Realization

`TestFailClosedQueryTest`: `queryTest` / `actionTest` / `runnerTest` fail closed. `transformerTest` is implemented in #4 (no longer fail-closed).

---

## Slice 5 — Nonreg, docs, AC

**Status:** ✅ DONE

### 5.1 Nonreg

```json
{
  "id": "unit-go-miroirtest",
  "tier": "unit",
  "title": "Go MiroirTest machine (unit functionCallTest)",
  "requires": "none",
  "argv": ["go", "test", "-C", "go", "./miroirtest"]
}
```

### AC checklist (#3)

| Criterion | Proven by | Status |
|---|---|---|
| Load / walk existing suite JSON | Slice 0–1 | ✅ |
| Run `functionCallTest` with TS `functionRef` | Slices 1–3 | ✅ |
| No new / copied MiroirTest assets | same `a311f363-…` files | ✅ |
| `./build-all.sh reset && npm run nonreg` | Slice 5 / epic Slice 4 | ✅ |

### Validation

```bash
go test -C go ./miroirtest
./build-all.sh reset && npm run nonreg
```

### Realization

Did not add a second nonreg step. `unit-go-jzod` already runs `go test -C go ./...`, which includes `./miroirtest`. AGENTS.md / contributing testing note the identical-JSON rule. Full reset+nonreg green: `test-results/nonreg/20260903T005406Z`.
