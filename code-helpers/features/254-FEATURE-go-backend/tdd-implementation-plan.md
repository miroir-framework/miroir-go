# Issue #254 — TDD Implementation Plan (epic index)

> Epic sequencing only. Implementation lives on the child plans. Tests use the **same**
> MiroirTest / schema JSON as `miroir-test-app_deployment-miroir` — no forked suites.
>
> **Execution model:** human-in-the-loop. No slice contains a commit step.

Analysis: [`./analysis.md`](./analysis.md) · Issue: https://github.com/miroir-framework/miroir/issues/254
Children: [`../255-FEATURE-go-jzod-typecheck/tdd-implementation-plan.md`](../255-FEATURE-go-jzod-typecheck/tdd-implementation-plan.md) · [`../256-FEATURE-go-miroirtest-unit-machine/tdd-implementation-plan.md`](../256-FEATURE-go-miroirtest-unit-machine/tdd-implementation-plan.md) · [`../257-FEATURE-go-transformer-unit-runtime/tdd-implementation-plan.md`](../257-FEATURE-go-transformer-unit-runtime/tdd-implementation-plan.md)
Working branch: `254-FEATURE-go-backend`

**Resume note:** Index reviewed and realized (2026-09-03). #255–#257 done. Epic nonreg green (`20260903T005406Z`).

---

## Scope

- Sequence #255 → #256 → #257.
- Shared JSON is the source of truth.
- `go/` module layout.

This plan does **not** implement HTTP / `miroir-server` replacement.

---

## Progress summary

| Slice | Title | Status | Primary proof |
|---|---|---|---|
| 1 | #255 Go Jzod TypeCheck | ✅ | `go test -C go ./jzod` |
| 2 | #256 unit MiroirTest machine | ✅ | existing `functionCallTest` JSON |
| 3 | #257 transformer apply + unit `transformerTest` | ✅ | same `transformerTest` JSON as TS |
| 4 | Epic nonreg | ✅ | `./build-all.sh reset && npm run nonreg` (`20260903T005406Z`) |

---

## Locked implementation defaults

Copied from the epic analysis (D1–D7) plus 2026-09-03: **no new / copied MiroirTest assets**.

| Decision | Choice |
|---|---|
| D1 | Go ML runtime; do not replace `miroir-server` |
| D2 | Module at `go/` |
| D3 | Port `jzodTypeCheck`, not Zod |
| D4 | Existing JSON is source of truth |
| D5 | Bootstrap self-parse is #255 `go test`; #256 first suite is existing `mustache` JSON |
| D6 | #257 unit `transformerTest` only |
| D7 | Same files as TS; Go interprets them |

---

## Allocated UUIDs / keys

No new model uuids at epic level. One nonreg step: `unit-go-jzod` (`go test -C go ./...`) covers jzod + machine + transformers.

---

## Test execution conventions

| Purpose | Command |
|---|---|
| Go jzod | `go test -C go ./jzod` |
| Go machine | `go test -C go ./miroirtest` |
| Go transformers | `go test -C go ./transformer` |
| TS oracle | `npm run testMiroir -w miroir-core -- --suites <key> --mode unit` |
| Step gate | `./build-all.sh reset && npm run nonreg` |

---

## Slice 1 — #255

See child plan. Status: ✅

### Realization

`go/jzod` TypeCheck + 42 typecheck leaves + bootstrap self-parse. `unit-go-jzod` in manifest runs `go test -C go ./...`.

---

## Slice 2 — #256

See child plan. Status: ✅. Existing `functionCallTest` JSON only (364 leaves).

### Realization

`go/miroirtest` loads the 51-file MiroirTest directory. `TestUnitFunctionCallSuites` is green. `queryTest` / `actionTest` / `runnerTest` fail closed.

---

## Slice 3 — #257

See child plan. Status: ✅. Existing `transformerTest` JSON only (317 leaves).

### Realization

`go/transformer` apply + composite `entityDefinition_extractAttributes`. `TestUnitTransformerSuites` is green on the same files as TS unit.

---

## Slice 4 — Epic nonreg, docs, AC

**Status:** ✅ DONE

### Validation

```bash
go test -C go ./...
./build-all.sh reset && npm run nonreg
```

### AC checklist (#254)

| Criterion | Proven by | Status |
|---|---|---|
| Go TypeCheck on Miroir bootstrap / suite JSON | #255 | ✅ |
| Go unit MiroirTest machine on **same** JSON | #256 | ✅ |
| Go transformer unit suites on **same** JSON | #257 | ✅ |
| TS non-regression | nonreg after each child | ✅ |

### Realization

`go test -C go ./...` green. `./build-all.sh reset && npm run nonreg` green (`test-results/nonreg/20260903T005406Z`), including `unit-go-jzod` (3.1s).
