package transformer

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/miroir-framework/miroir/go/jzod"
)

type handler func(a ApplyArgs, xf map[string]any) (any, error)

var handlers = map[string]handler{}

func init() {
	handlers["handleTransformer_constant"] = handleReturnValue
	handlers["returnValue"] = handleReturnValue
	handlers["handleTransformer_constantAsExtractor"] = handleConstantAsExtractor
	handlers["constantAsExtractor"] = handleConstantAsExtractor
	handlers["handleTransformer_ifThenElse"] = handleIfThenElse
	handlers["ifThenElse"] = handleIfThenElse
	handlers["handleTransformer_boolExpr"] = handleBoolExpr
	handlers["boolExpr"] = handleBoolExpr
	handlers["handleTransformer_plus"] = handlePlus
	handlers["+"] = handlePlus
	handlers["handleTransformer_case"] = handleCase
	handlers["case"] = handleCase
	handlers["handleTransformer_getFromContext"] = handleGetFromContext
	handlers["getFromContext"] = handleGetFromContext
	handlers["handleTransformer_getFromParameters"] = handleGetFromParameters
	handlers["getFromParameters"] = handleGetFromParameters
	handlers["handleTransformer_generateUuid"] = handleGenerateUuid
	handlers["generateUuid"] = handleGenerateUuid
	handlers["handleTransformer_dataflowObject"] = handleDataflowObject
	handlers["dataflowObject"] = handleDataflowObject
	handlers["handleTransformer_FreeObjectTemplate"] = handleCreateObject
	handlers["createObject"] = handleCreateObject
	handlers["transformer_mustacheStringTemplate_apply"] = handleMustache
	handlers["mustacheStringTemplate"] = handleMustache
	handlers["handleCountTransformer"] = handleAggregate
	handlers["aggregate"] = handleAggregate
	handlers["handleListPickElementTransformer"] = handlePickFromList
	handlers["pickFromList"] = handlePickFromList
	handlers["handleUniqueTransformer"] = handleGetUniqueValues
	handlers["getUniqueValues"] = handleGetUniqueValues
	handlers["transformerForBuild_list_listMapperToList_apply"] = handleMapList
	handlers["mapList"] = handleMapList
	handlers["transformer_object_indexListBy_apply"] = handleIndexListBy
	handlers["indexListBy"] = handleIndexListBy
	handlers["transformer_object_listReducerToSpreadObject_apply"] = handleListReducerToSpreadObject
	handlers["listReducerToSpreadObject"] = handleListReducerToSpreadObject
	handlers["handleTransformer_getObjectEntries"] = handleGetObjectEntries
	handlers["getObjectEntries"] = handleGetObjectEntries
	handlers["handleTransformer_getObjectValues"] = handleGetObjectValues
	handlers["getObjectValues"] = handleGetObjectValues
	handlers["handleTransformer_createObjectFromPairs"] = handleCreateObjectFromPairs
	handlers["createObjectFromPairs"] = handleCreateObjectFromPairs
	handlers["handleTransformer_mergeIntoObject"] = handleMergeIntoObject
	handlers["mergeIntoObject"] = handleMergeIntoObject
	handlers["transformer_dynamicObjectAccess_apply"] = handleAccessDynamicPath
	handlers["accessDynamicPath"] = handleAccessDynamicPath
	handlers["handleTransformer_concatLists"] = handleConcatLists
	handlers["concatLists"] = handleConcatLists
	handlers["handleTransformer_filterList"] = handleFilterList
	handlers["filterList"] = handleFilterList
	handlers["handleTransformer_find"] = handleFind
	handlers["find"] = handleFind
	handlers["handleTransformer_object_fromEntries"] = handleObjectFromEntries
	handlers["object_fromEntries"] = handleObjectFromEntries
	handlers["handleTransformer_sortList"] = handleSortList
	handlers["sortList"] = handleSortList
	handlers["handleTransformer_listLength"] = handleListLength
	handlers["listLength"] = handleListLength
	handlers["handleTransformer_stringOp"] = handleStringOp
	handlers["stringOp"] = handleStringOp
	handlers["handleTransformer_numericOp"] = handleNumericOp
	handlers["numericOp"] = handleNumericOp
	handlers["handleTransformer_currentDate"] = handleCurrentDate
	handlers["currentDate"] = handleCurrentDate
	handlers["handleTransformer_currentTimestamp"] = handleCurrentTimestamp
	handlers["currentTimestamp"] = handleCurrentTimestamp
	handlers["transformer_jzodTypeCheck"] = handleJzodTypeCheck
	handlers["jzodTypeCheck"] = handleJzodTypeCheck
	handlers["transformer_resolveTransformerResultSchema"] = handleResolveResultSchema
	handlers["resolveTransformerResultSchema"] = handleResolveResultSchema
	handlers["handleTransformer_ansiColumnsToJzodSchema"] = handleAnsiColumns
	handlers["ansiColumnsToJzodSchema"] = handleAnsiColumns
	handlers["transformer_defaultValueForMLSchema"] = handleDefaultValueForMLSchema
	handlers["defaultValueForMLSchema"] = handleDefaultValueForMLSchema
	handlers["transformer_unfoldSchemaOnce"] = handleUnfoldSchemaOnce
	handlers["unfoldSchemaOnce"] = handleUnfoldSchemaOnce
	handlers["transformer_resolveConditionalSchema"] = handleResolveConditionalSchema
	handlers["resolveConditionalSchema"] = handleResolveConditionalSchema
	handlers["transformer_resolveSchemaReferenceInContext"] = handleResolveSchemaReferenceInContext
	handlers["resolveSchemaReferenceInContext"] = handleResolveSchemaReferenceInContext
	handlers["handleTransformer_menu_AddItem"] = handleMenuAddItem
	handlers["transformer_menu_addItem"] = handleMenuAddItem
	handlers["handleTransformer_duplicateApplicationModel"] = handleDuplicateApplicationModel
	handlers["duplicateApplicationModel"] = handleDuplicateApplicationModel
}

func mustApply(a ApplyArgs, name string, xf any) any {
	v, err := applyValue(childArgs(a, name, xf))
	if err != nil {
		if f, ok := AsFailure(err); ok {
			throw(f)
		}
		throw(Failure{QueryFailure: "FailedTransformer", FailureMessage: err.Error(), TransformerPath: pathAny(a.Path)})
	}
	if f, ok := AsFailure(v); ok {
		throw(f)
	}
	return v
}

func handleReturnValue(_ ApplyArgs, xf map[string]any) (any, error) {
	return xf["value"], nil
}

func handleConstantAsExtractor(_ ApplyArgs, xf map[string]any) (any, error) {
	return xf["value"], nil
}

func handleIfThenElse(a ApplyArgs, xf map[string]any) (any, error) {
	cond := jsTruthy(mustApply(a, "if", xf["if"]))
	if cond {
		if xf["then"] == nil {
			return true, nil
		}
		return mustApply(a, "then", xf["then"]), nil
	}
	if xf["else"] == nil {
		return false, nil
	}
	return mustApply(a, "else", xf["else"]), nil
}

func handleBoolExpr(a ApplyArgs, xf map[string]any) (any, error) {
	left := mustApply(a, "left", xf["left"])
	op, _ := xf["operator"].(string)
	unary := op == "isNull" || op == "isNotNull" || op == "!"
	var right any
	if !unary && xf["right"] != nil {
		right = mustApply(a, "right", xf["right"])
	}
	switch op {
	case "==":
		return jsLooseEqual(left, right), nil
	case "!=":
		return !jsLooseEqual(left, right), nil
	case "===":
		return jsStrictEqual(left, right), nil
	case "!==":
		return !jsStrictEqual(left, right), nil
	case "deepEqual":
		return deepEqualJSON(left, right), nil
	case "notDeepEqual":
		return !deepEqualJSON(left, right), nil
	case "<":
		return jsLess(left, right), nil
	case "<=":
		return jsLess(left, right) || jsLooseEqual(left, right), nil
	case ">":
		return jsLess(right, left), nil
	case ">=":
		return jsLess(right, left) || jsLooseEqual(left, right), nil
	case "&&":
		return jsTruthy(left) && jsTruthy(right), nil
	case "||":
		return jsTruthy(left) || jsTruthy(right), nil
	case "isNull":
		return left == nil, nil
	case "isNotNull":
		return left != nil, nil
	case "!":
		return !jsTruthy(left), nil
	default:
		return false, nil
	}
}

func handlePlus(a ApplyArgs, xf map[string]any) (any, error) {
	args, _ := xf["args"].([]any)
	if len(args) == 0 {
		throw(Failure{QueryFailure: "FailedTransformer", TransformerPath: pathAny(a.Path), FailureOrigin: []any{"handleTransformer_plus"}, FailureMessage: "Cannot apply + to empty args array"})
	}
	evaluated := make([]any, len(args))
	for i, arg := range args {
		evaluated[i] = mustApply(a, "args."+strconv.Itoa(i), arg)
	}
	if len(evaluated) == 1 {
		return evaluated[0], nil
	}
	result := evaluated[0]
	resultIsBigint := schemaTypeOfArg(args[0]) == "bigint"
	for i := 1; i < len(evaluated); i++ {
		next := evaluated[i]
		nextIsBigint := schemaTypeOfArg(args[i]) == "bigint"
		if result == nil || next == nil {
			throw(Failure{QueryFailure: "FailedTransformer", TransformerPath: pathAny(a.Path), FailureOrigin: []any{"handleTransformer_plus"}, FailureMessage: fmt.Sprintf("Cannot apply + to null/undefined at index %d: result=%v, next=%v", i, result, next)})
		}
		rf, rok := toFloat(result)
		nf, nok := toFloat(next)
		rs, rstr := result.(string)
		ns, nstr := next.(string)
		if resultIsBigint && nextIsBigint {
			rb, err1 := strconv.ParseInt(jsString(result), 10, 64)
			nb, err2 := strconv.ParseInt(jsString(next), 10, 64)
			if err1 != nil || err2 != nil {
				throw(Failure{QueryFailure: "FailedTransformer", TransformerPath: pathAny(a.Path), FailureOrigin: []any{"handleTransformer_plus"}, FailureMessage: fmt.Sprintf("Failed to perform bigint addition at index %d", i)})
			}
			result = strconv.FormatInt(rb+nb, 10)
			resultIsBigint = nextIsBigint
			continue
		}
		if rok && nok && !rstr && !nstr {
			result = rf + nf
		} else if rstr && nstr {
			if resultIsBigint != nextIsBigint {
				throw(Failure{QueryFailure: "FailedTransformer", TransformerPath: pathAny(a.Path), FailureOrigin: []any{"handleTransformer_plus"}, FailureMessage: fmt.Sprintf("Type mismatch at index %d: cannot apply + between bigint and string.", i)})
			}
			result = rs + ns
		} else {
			throw(Failure{QueryFailure: "FailedTransformer", TransformerPath: pathAny(a.Path), FailureOrigin: []any{"handleTransformer_plus"}, FailureMessage: fmt.Sprintf("Type mismatch at index %d: cannot apply + to %T and %T. All operands must be of the same type (number, string, or bigint).", i, result, next), QueryContext: map[string]any{"result": result, "nextValue": next}})
		}
		resultIsBigint = nextIsBigint
	}
	return result, nil
}

func schemaTypeOfArg(arg any) string {
	m, _ := asMap(arg)
	ml, _ := asMap(m["mlSchema"])
	t, _ := ml["type"].(string)
	return t
}

func handleCase(a ApplyArgs, xf map[string]any) (any, error) {
	disc := mustApply(a, "discriminator", xf["discriminator"])
	whens, _ := xf["whens"].([]any)
	for i, raw := range whens {
		when, _ := asMap(raw)
		whenVal := mustApply(a, fmt.Sprintf("whens.%d.when", i), when["when"])
		if jsLooseEqual(disc, whenVal) {
			return mustApply(a, fmt.Sprintf("whens.%d.then", i), when["then"]), nil
		}
	}
	if xf["else"] != nil {
		return mustApply(a, "else", xf["else"]), nil
	}
	return nil, nil
}

func handleGetFromContext(a ApplyArgs, xf map[string]any) (any, error) {
	return resolveInnerReference(a, xf), nil
}

func handleGetFromParameters(a ApplyArgs, xf map[string]any) (any, error) {
	return resolveInnerReference(a, xf), nil
}

func handleGenerateUuid(a ApplyArgs, xf map[string]any) (any, error) {
	return resolveInnerReference(a, xf), nil
}

func handleDataflowObject(a ApplyArgs, xf map[string]any) (any, error) {
	def, _ := asMap(xf["definition"])
	result := map[string]any{}
	keys := KeysOfOrSorted(def)
	for _, key := range keys {
		ctx := copyMap(a.Context)
		for k, v := range result {
			ctx[k] = v
		}
		child := a
		child.Context = ctx
		result[key] = mustApply(child, key, def[key])
	}
	jzod.RememberKeys(result, keys)
	if target, _ := xf["target"].(string); target != "" {
		return result[target], nil
	}
	return result, nil
}

func handleCreateObject(a ApplyArgs, xf map[string]any) (any, error) {
	def, _ := asMap(xf["definition"])
	result := map[string]any{}
	keys := KeysOfOrSorted(def)
	for _, key := range keys {
		result[key] = mustApply(a, key, def[key])
	}
	jzod.RememberKeys(result, keys)
	return result, nil
}

func handleMustache(a ApplyArgs, xf map[string]any) (any, error) {
	def, _ := xf["definition"].(string)
	if strings.Count(def, "{{") != strings.Count(def, "}}") {
		throw(Failure{QueryFailure: "FailedTransformer", TransformerPath: pathAny(a.Path), FailureOrigin: []any{"transformer_mustacheStringTemplate_apply"}, QueryContext: "error in transformer_mustacheStringTemplate_apply, could not render template."})
	}
	interp, _ := xf["interpolation"].(string)
	bank := a.Params
	if interp == "runtime" {
		bank = a.Context
	}
	return renderMustache(def, bank), nil
}

func handleJzodTypeCheck(_ ApplyArgs, xf map[string]any) (any, error) {
	valuePath, _ := xf["currentValuePath"].([]any)
	typePath, _ := xf["currentTypePath"].([]any)
	rel, _ := asMap(xf["relativeReferenceJzodContext"])
	env := jzod.EnvironmentFromMap(jzod.DefaultMiroirModelEnvironment())
	result := jzod.TypeCheck(xf["mlSchema"], xf["valueObject"], valuePath, typePath, env, rel)
	raw, _ := json.Marshal(result)
	var out map[string]any
	_ = json.Unmarshal(raw, &out)
	return out, nil
}

func handleResolveResultSchema(a ApplyArgs, xf map[string]any) (any, error) {
	ctx, _ := asMap(xf["context"])
	return ResolveResultSchema(xf["transformer"], ctx, Definitions()), nil
}

func handleAnsiColumns(a ApplyArgs, xf map[string]any) (any, error) {
	cols := xf["value"]
	if cols == nil {
		cols = resolveApplyTo(a, xf)
	} else if isTypedTransformer(cols) {
		cols = mustApply(a, "value", cols)
	}
	out, err := jzod.AnsiColumnsToJzodSchema(cols)
	if err != nil {
		throw(Failure{QueryFailure: "FailedTransformer", TransformerPath: pathAny(a.Path), FailureOrigin: []any{"handleTransformer_ansiColumnsToJzodSchema"}, FailureMessage: err.Error()})
	}
	return out, nil
}

func handleCurrentDate(ApplyArgs, map[string]any) (any, error) {
	return time.Now().UTC().Format("2006-01-02"), nil
}

func handleCurrentTimestamp(ApplyArgs, map[string]any) (any, error) {
	return time.Now().UTC().Format("2006-01-02T15:04:05.000Z"), nil
}

func handleGenerateUuidRaw() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

func resolveApplyTo(a ApplyArgs, xf map[string]any) any {
	if xf["applyTo"] == nil {
		tt := "getFromContext"
		if a.Step == "build" {
			tt = "getFromParameters"
		}
		return mustApply(a, "applyTo", map[string]any{"transformerType": tt, "referenceName": defaultTransformerInput})
	}
	applyTo := xf["applyTo"]
	switch applyTo.(type) {
	case string, float64, float32, int, bool:
		return applyTo
	}
	if applyTo == nil {
		return nil
	}
	if _, ok := applyTo.([]any); ok {
		return applyTo
	}
	if m, ok := asMap(applyTo); ok {
		if m["transformerType"] == nil {
			return applyTo
		}
		return mustApply(a, "applyTo", applyTo)
	}
	throw(Failure{QueryFailure: "FailedTransformer", TransformerPath: append(pathAny(a.Path), "applyTo"), FailureOrigin: []any{"resolveApplyTo_legacy"}, FailureMessage: "resolveApplyTo failed, unknown type for transformer.applyTo=" + fmt.Sprint(applyTo)})
	return nil
}

func requireList(v any, origin, msg string, path []string) []any {
	list, ok := asList(v)
	if !ok {
		throw(Failure{QueryFailure: "FailedTransformer", TransformerPath: pathAny(path), FailureOrigin: []any{origin}, FailureMessage: msg, QueryContext: fmt.Sprintf("%s: %T", msg, v), QueryParameters: v})
	}
	return list
}

func handleAggregate(a ApplyArgs, xf map[string]any) (any, error) {
	resolved := resolveApplyTo(a, xf)
	list := requireList(resolved, "handleCountTransformer", "count can not apply to resolvedReference of wrong type: "+fmt.Sprintf("%T", resolved), a.Path)
	fn, _ := xf["function"].(string)
	resultKey := fn
	if resultKey == "" {
		resultKey = "aggregate"
	}
	attr, _ := xf["attribute"].(string)
	distinct, _ := xf["distinct"].(bool)
	attrObj, _ := asMap(xf["attributeObject"])
	if xf["groupBy"] != nil {
		var groupBy []string
		if s, ok := xf["groupBy"].(string); ok {
			groupBy = []string{s}
		} else if arr, ok := xf["groupBy"].([]any); ok {
			for _, x := range arr {
				groupBy = append(groupBy, fmt.Sprint(x))
			}
		}
		type row struct {
			attrs map[string]any
			acc   *aggAcc
		}
		groups := map[string]*row{}
		var order []string
		for _, entry := range list {
			em := mustMap(entry)
			attrs := map[string]any{}
			for _, g := range groupBy {
				attrs[g] = em[g]
			}
			key := stringify(attrs)
			if existing, ok := groups[key]; ok {
				accumulateAgg(existing.acc, em, fn, attr, distinct, attrObj)
			} else {
				r := &row{attrs: attrs, acc: initAgg(em, fn, attr, distinct, attrObj)}
				groups[key] = r
				order = append(order, key)
			}
		}
		var result []any
		for _, key := range order {
			r := groups[key]
			item := copyMap(r.attrs)
			item[resultKey] = finalizeAgg(r.acc, fn)
			result = append(result, item)
		}
		sort.SliceStable(result, func(i, j int) bool {
			im, _ := asMap(result[i])
			jm, _ := asMap(result[j])
			for _, g := range groupBy {
				if jsLess(im[g], jm[g]) {
					return true
				}
				if jsLess(jm[g], im[g]) {
					return false
				}
			}
			return false
		})
		if xf["having"] != nil {
			var filtered []any
			for _, rowv := range result {
				rm := mustMap(rowv)
				ctx := copyMap(a.Context)
				ctx["aggregateValue"] = rm[resultKey]
				child := a
				child.Context = ctx
				ok := false
				func() {
					defer func() { _ = recover() }()
					ok = jsTruthy(mustApply(child, "having", xf["having"]))
				}()
				if ok {
					filtered = append(filtered, rowv)
				}
			}
			result = filtered
		}
		return result, nil
	}
	acc := &aggAcc{count: 0, sum: 0, values: []any{}}
	if distinct {
		acc.distinct = map[string]bool{}
	}
	for _, entry := range list {
		accumulateAgg(acc, mustMap(entry), fn, attr, distinct, attrObj)
	}
	result := []any{map[string]any{resultKey: finalizeAgg(acc, fn)}}
	if xf["having"] != nil {
		var filtered []any
		for _, rowv := range result {
			rm := mustMap(rowv)
			ctx := copyMap(a.Context)
			ctx["aggregateValue"] = rm[resultKey]
			child := a
			child.Context = ctx
			ok := false
			func() {
				defer func() { _ = recover() }()
				ok = jsTruthy(mustApply(child, "having", xf["having"]))
			}()
			if ok {
				filtered = append(filtered, rowv)
			}
		}
		result = filtered
	}
	return result, nil
}

type aggAcc struct {
	count    int
	sum      float64
	min      *float64
	max      *float64
	values   []any
	distinct map[string]bool
}

func entryVal(entry map[string]any, attr string, attrObj map[string]any) any {
	if attrObj != nil {
		out := map[string]any{}
		for k, v := range attrObj {
			out[k] = entry[fmt.Sprint(v)]
		}
		return out
	}
	if attr != "" {
		return entry[attr]
	}
	return nil
}

func isNullOrAllNull(val any) bool {
	if val == nil {
		return true
	}
	m, ok := asMap(val)
	if !ok {
		return false
	}
	for _, v := range m {
		if v != nil {
			return false
		}
	}
	return true
}

func initAgg(entry map[string]any, fn, attr string, distinct bool, attrObj map[string]any) *aggAcc {
	acc := &aggAcc{count: 0}
	if distinct {
		acc.distinct = map[string]bool{}
	}
	accumulateAgg(acc, entry, fn, attr, distinct, attrObj)
	return acc
}

func accumulateAgg(acc *aggAcc, entry map[string]any, fn, attr string, distinct bool, attrObj map[string]any) {
	val := entryVal(entry, attr, attrObj)
	switch fn {
	case "sum", "avg":
		if n, ok := toFloat(val); ok {
			acc.sum += n
		}
		acc.count++
	case "min":
		if n, ok := toFloat(val); ok {
			if acc.min == nil || n < *acc.min {
				acc.min = &n
			}
		}
		acc.count++
	case "max":
		if n, ok := toFloat(val); ok {
			if acc.max == nil || n > *acc.max {
				acc.max = &n
			}
		}
		acc.count++
	case "json_agg":
		acc.values = append(acc.values, val)
		acc.count++
	case "json_agg_strict":
		if !isNullOrAllNull(val) {
			acc.values = append(acc.values, val)
		}
		acc.count++
	case "count":
		if distinct && attr != "" {
			acc.distinct[stringify(val)] = true
		}
		acc.count++
	default:
		acc.count++
	}
}

func finalizeAgg(acc *aggAcc, fn string) any {
	switch fn {
	case "sum":
		return acc.sum
	case "avg":
		if acc.count == 0 {
			return 0
		}
		return acc.sum / float64(acc.count)
	case "min":
		if acc.min == nil {
			return nil
		}
		return *acc.min
	case "max":
		if acc.max == nil {
			return nil
		}
		return *acc.max
	case "json_agg", "json_agg_strict":
		return acc.values
	case "count":
		if acc.distinct != nil {
			return len(acc.distinct)
		}
		return acc.count
	default:
		return acc.count
	}
}

func handlePickFromList(a ApplyArgs, xf map[string]any) (any, error) {
	list := requireList(resolveApplyTo(a, xf), "innerTransformer_apply", "pickFromList can not apply to resolvedReference, wrong type", a.Path)
	orderBy, _ := xf["orderBy"].(string)
	copied := append([]any{}, list...)
	if orderBy != "" {
		sort.SliceStable(copied, func(i, j int) bool {
			im, _ := asMap(copied[i])
			jm, _ := asMap(copied[j])
			return localeCompare(fmt.Sprint(im[orderBy]), fmt.Sprint(jm[orderBy])) < 0
		})
	}
	idx := 0
	switch n := xf["index"].(type) {
	case float64:
		idx = int(n)
	case int:
		idx = n
	case string:
		idx, _ = strconv.Atoi(n)
	}
	if idx < 0 {
		idx = len(copied) + idx
	}
	if idx < 0 || idx >= len(copied) {
		return nil, nil
	}
	return copied[idx], nil
}

func handleGetUniqueValues(a ApplyArgs, xf map[string]any) (any, error) {
	list := requireList(resolveApplyTo(a, xf), "handleUniqueTransformer", "getUniqueValues applyTo is not an array", a.Path)
	attr, _ := xf["attribute"].(string)
	seen := map[string]bool{}
	var out []any
	for _, item := range list {
		var val any = item
		if attr != "" {
			if m, ok := asMap(item); ok {
				val = m[attr]
			}
		}
		key := stringify(val)
		if !seen[key] {
			seen[key] = true
			if attr != "" {
				out = append(out, map[string]any{attr: val})
			} else {
				out = append(out, val)
			}
		}
	}
	orderBy, _ := xf["orderBy"].(string)
	if orderBy != "" {
		sort.SliceStable(out, func(i, j int) bool {
			im, _ := asMap(out[i])
			jm, _ := asMap(out[j])
			return localeCompare(fmt.Sprint(im[orderBy]), fmt.Sprint(jm[orderBy])) < 0
		})
	}
	return out, nil
}

func handleMapList(a ApplyArgs, xf map[string]any) (any, error) {
	resolved := resolveApplyTo(a, xf)
	var out []any
	if list, ok := asList(resolved); ok {
		for _, element := range list {
			ctx := copyMap(a.Context)
			ctx[outerName(xf)] = element
			child := a
			child.Context = ctx
			out = append(out, mustApply(child, "elementTransformer", xf["elementTransformer"]))
		}
		return out, nil
	}
	if m, ok := asMap(resolved); ok {
		for _, k := range KeysOfOrSorted(m) {
			ctx := copyMap(a.Context)
			ctx[outerName(xf)] = m[k]
			child := a
			child.Context = ctx
			out = append(out, mustApply(child, "elementTransformer", xf["elementTransformer"]))
		}
		return out, nil
	}
	throw(Failure{QueryFailure: "FailedTransformer", TransformerPath: pathAny(a.Path), FailureOrigin: []any{"transformerForBuild_list_listMapperToList_apply"}, FailureMessage: "mapList can not work on resolvedReference"})
	return nil, nil
}

func handleIndexListBy(a ApplyArgs, xf map[string]any) (any, error) {
	list := requireList(resolveApplyTo(a, xf), "transformer_object_indexListBy_apply", "transformer_object_indexListBy_apply can not apply to resolvedReference of wrong type", a.Path)
	index := xf["indexAttribute"]
	if index == nil {
		index = xf["indexBy"]
	}
	out := map[string]any{}
	for _, item := range list {
		m := mustMap(item)
		var key string
		if s, ok := index.(string); ok {
			key = fmt.Sprint(m[s])
		} else if arr, ok := index.([]any); ok {
			parts := make([]string, len(arr))
			for i, k := range arr {
				parts[i] = fmt.Sprint(m[fmt.Sprint(k)])
			}
			key = strings.Join(parts, "|")
		}
		out[key] = item
	}
	return out, nil
}

func handleListReducerToSpreadObject(a ApplyArgs, xf map[string]any) (any, error) {
	list := requireList(resolveApplyTo(a, xf), "transformer_object_listReducerToSpreadObject_apply", "transformer_object_listReducerToSpreadObject_apply can not apply to resolvedReference of wrong type", a.Path)
	out := map[string]any{}
	for _, item := range list {
		m, ok := asMap(item)
		if !ok {
			throw(Failure{QueryFailure: "FailedTransformer", TransformerPath: pathAny(a.Path), FailureOrigin: []any{"transformer_object_listReducerToSpreadObject_apply"}, FailureMessage: "listReducerToSpreadObject fails when non-objects are included in the list"})
		}
		for k, v := range m {
			out[k] = v
		}
	}
	return out, nil
}

func handleGetObjectEntries(a ApplyArgs, xf map[string]any) (any, error) {
	resolved := resolveApplyTo(a, xf)
	m, ok := asMap(resolved)
	if !ok || resolved == nil {
		throw(Failure{QueryFailure: "FailedTransformer", TransformerPath: pathAny(a.Path), FailureOrigin: []any{"handleTransformer_getObjectEntries"}, FailureMessage: "handleTransformer_getObjectEntries called on something that is not an object: " + fmt.Sprintf("%T", resolved)})
	}
	if _, isList := asList(resolved); isList {
		throw(Failure{QueryFailure: "FailedTransformer", TransformerPath: pathAny(a.Path), FailureOrigin: []any{"handleTransformer_getObjectEntries"}, FailureMessage: "handleTransformer_getObjectEntries called on something that is not an object: array"})
	}
	var out []any
	for _, k := range KeysOfOrSorted(m) {
		out = append(out, []any{k, m[k]})
	}
	return out, nil
}

func handleGetObjectValues(a ApplyArgs, xf map[string]any) (any, error) {
	resolved := resolveApplyTo(a, xf)
	m, ok := asMap(resolved)
	if !ok {
		throw(Failure{QueryFailure: "FailedTransformer", TransformerPath: pathAny(a.Path), FailureOrigin: []any{"handleTransformer_getObjectValues"}, FailureMessage: "handleTransformer_getObjectValues resolvedReference is not an object: " + fmt.Sprintf("%T", resolved)})
	}
	var out []any
	for _, k := range KeysOfOrSorted(m) {
		out = append(out, m[k])
	}
	return out, nil
}

func handleCreateObjectFromPairs(a ApplyArgs, xf map[string]any) (any, error) {
	if xf["applyTo"] != nil {
		resolved := resolveApplyTo(a, xf)
		ctx := copyMap(a.Context)
		ctx[outerName(xf)] = resolved
		a.Context = ctx
	}
	pairs := xf["definition"]
	list, ok := pairs.([]any)
	if !ok {
		resolved := mustApply(a, "definition", pairs)
		list, ok = asList(resolved)
		if !ok {
			return map[string]any{}, nil
		}
	}
	out := map[string]any{}
	var keys []string
	for i, raw := range list {
		pm, _ := asMap(raw)
		var key any
		if isTypedTransformer(pm["attributeKey"]) {
			key = mustApply(a, fmt.Sprintf("definition.%d.attributeKey", i), pm["attributeKey"])
		} else {
			key = pm["attributeKey"]
		}
		val := mustApply(a, fmt.Sprintf("definition.%d.attributeValue", i), pm["attributeValue"])
		ks := fmt.Sprint(key)
		out[ks] = val
		keys = append(keys, ks)
	}
	jzod.RememberKeys(out, keys)
	return out, nil
}

func handleMergeIntoObject(a ApplyArgs, xf map[string]any) (any, error) {
	var resolved any
	base := map[string]any{}
	if xf["applyTo"] != nil {
		resolved = resolveApplyTo(a, xf)
		if m, ok := asMap(resolved); ok {
			base = copyMap(m)
		}
	}
	ctx := copyMap(a.Context)
	ctx[outerName(xf)] = resolved
	child := a
	child.Context = ctx
	overlay := mustApply(child, "definition", xf["definition"])
	if om, ok := asMap(overlay); ok {
		for k, v := range om {
			base[k] = v
		}
	}
	return base, nil
}

func handleAccessDynamicPath(a ApplyArgs, xf map[string]any) (any, error) {
	path := xf["objectAccessPath"]
	if path == nil {
		path = xf["path"]
	}
	list, ok := path.([]any)
	if !ok || len(list) == 0 {
		return nil, nil
	}
	acc := list[0]
	if isTypedTransformer(acc) {
		acc = mustApply(a, "objectAccessPath.0", acc)
	}
	for i, seg := range list[1:] {
		if isTypedTransformer(seg) {
			seg = mustApply(a, fmt.Sprintf("objectAccessPath.%d", i+1), seg)
		}
		if s, ok := seg.(string); ok {
			acc = walkAccess(acc, s, a.Path)
			continue
		}
		acc = walkAccess(acc, seg, a.Path)
	}
	return acc, nil
}

func walkAccess(acc, seg any, path []string) any {
	if acc == nil {
		throw(Failure{QueryFailure: "FailedTransformer", TransformerPath: pathAny(path), FailureOrigin: []any{"transformer_dynamicObjectAccess_apply"}, FailureMessage: "error in transformer_dynamicObjectAccess_apply, could not find key: " + fmt.Sprint(seg)})
	}
	key := fmt.Sprint(seg)
	if m, ok := asMap(acc); ok {
		v, exists := m[key]
		if !exists {
			throw(Failure{QueryFailure: "FailedTransformer", TransformerPath: pathAny(path), FailureOrigin: []any{"transformer_dynamicObjectAccess_apply"}, FailureMessage: "error in transformer_dynamicObjectAccess_apply, could not find key: " + key})
		}
		return v
	}
	if list, ok := asList(acc); ok {
		idx, err := strconv.Atoi(key)
		if err != nil || idx < 0 || idx >= len(list) {
			throw(Failure{QueryFailure: "FailedTransformer", TransformerPath: pathAny(path), FailureOrigin: []any{"transformer_dynamicObjectAccess_apply"}, FailureMessage: "error in transformer_dynamicObjectAccess_apply, could not find key: " + key})
		}
		return list[idx]
	}
	throw(Failure{QueryFailure: "FailedTransformer", TransformerPath: pathAny(path), FailureOrigin: []any{"transformer_dynamicObjectAccess_apply"}, FailureMessage: "error in transformer_dynamicObjectAccess_apply, could not find key: " + key})
	return nil
}

func handleConcatLists(a ApplyArgs, xf map[string]any) (any, error) {
	lists, _ := xf["lists"].([]any)
	var out []any
	for i, item := range lists {
		resolved := mustApply(a, fmt.Sprintf("lists.%d", i), item)
		list, ok := asList(resolved)
		if !ok {
			throw(Failure{QueryFailure: "FailedTransformer", TransformerPath: pathAny(a.Path), FailureOrigin: []any{"handleTransformer_concatLists"}, FailureMessage: fmt.Sprintf("concatLists: element at index %d is not an array, got: %T", i, resolved)})
		}
		out = append(out, list...)
	}
	if out == nil {
		out = []any{}
	}
	return out, nil
}

func handleFilterList(a ApplyArgs, xf map[string]any) (any, error) {
	list := requireList(resolveApplyTo(a, xf), "handleTransformer_filterList", "filterList: applyTo is not an array", a.Path)
	var out []any
	for _, element := range list {
		func() {
			defer func() { _ = recover() }()
			ctx := copyMap(a.Context)
			ctx[outerName(xf)] = element
			child := a
			child.Context = ctx
			if mustApply(child, "predicate", xf["predicate"]) == true {
				out = append(out, element)
			}
		}()
	}
	orderBy, _ := xf["orderBy"].(string)
	if orderBy != "" {
		sort.SliceStable(out, func(i, j int) bool {
			im, _ := asMap(out[i])
			jm, _ := asMap(out[j])
			return localeCompare(fmt.Sprint(im[orderBy]), fmt.Sprint(jm[orderBy])) < 0
		})
	}
	return out, nil
}

func handleFind(a ApplyArgs, xf map[string]any) (any, error) {
	list := requireList(resolveApplyTo(a, xf), "handleTransformer_find", "find: applyTo is not an array", a.Path)
	for _, element := range list {
		var found any
		ok := false
		func() {
			defer func() { _ = recover() }()
			ctx := copyMap(a.Context)
			ctx[outerName(xf)] = element
			child := a
			child.Context = ctx
			if mustApply(child, "predicate", xf["predicate"]) == true {
				found = element
				ok = true
			}
		}()
		if ok {
			return found, nil
		}
	}
	return nil, nil
}

func handleObjectFromEntries(a ApplyArgs, xf map[string]any) (any, error) {
	list := requireList(resolveApplyTo(a, xf), "handleTransformer_object_fromEntries", "object_fromEntries applyTo is not an array", a.Path)
	out := map[string]any{}
	var keys []string
	for _, raw := range list {
		if arr, ok := raw.([]any); ok && len(arr) >= 2 {
			key := fmt.Sprint(arr[0])
			out[key] = arr[1]
			keys = append(keys, key)
			continue
		}
		if m, ok := asMap(raw); ok {
			key := fmt.Sprint(m["0"])
			if key == "<nil>" {
				key = fmt.Sprint(m["key"])
			}
			out[key] = m["1"]
			if m["1"] == nil {
				out[key] = m["value"]
			}
			keys = append(keys, key)
		}
	}
	jzod.RememberKeys(out, keys)
	return out, nil
}

func handleSortList(a ApplyArgs, xf map[string]any) (any, error) {
	list := requireList(resolveApplyTo(a, xf), "handleTransformer_sortList", "sortList applyTo is not an array", a.Path)
	out := append([]any{}, list...)
	orderBy, _ := xf["orderBy"].(string)
	dir, _ := xf["orderByDirection"].(string)
	mult := 1
	if dir == "desc" {
		mult = -1
	}
	sort.SliceStable(out, func(i, j int) bool {
		var aVal, bVal any
		if orderBy != "" {
			im, _ := asMap(out[i])
			jm, _ := asMap(out[j])
			aVal, bVal = im[orderBy], jm[orderBy]
		} else {
			aVal, bVal = out[i], out[j]
		}
		if sa, ok := aVal.(string); ok {
			if sb, ok := bVal.(string); ok {
				return localeCompare(sa, sb)*mult < 0
			}
		}
		if af, aok := toFloat(aVal); aok {
			if bf, bok := toFloat(bVal); bok {
				return (af-bf)*float64(mult) < 0
			}
		}
		return false
	})
	return out, nil
}

func handleListLength(a ApplyArgs, xf map[string]any) (any, error) {
	list := requireList(resolveApplyTo(a, xf), "handleTransformer_listLength", "listLength applyTo is not an array", a.Path)
	return float64(len(list)), nil
}

func handleStringOp(a ApplyArgs, xf map[string]any) (any, error) {
	resolved := resolveApplyTo(a, xf)
	op, _ := xf["op"].(string)
	asString := func() string {
		s, ok := resolved.(string)
		if !ok {
			throw(Failure{QueryFailure: "FailedTransformer", TransformerPath: pathAny(a.Path), FailureOrigin: []any{"handleTransformer_stringOp"}, FailureMessage: "stringOp " + op + ": applyTo is not a string, got: " + fmt.Sprintf("%T", resolved)})
		}
		return s
	}
	switch op {
	case "toLowerCase":
		return strings.ToLower(asString()), nil
	case "toUpperCase":
		return strings.ToUpper(asString()), nil
	case "trim":
		return strings.TrimSpace(asString()), nil
	case "length":
		return float64(len(asString())), nil
	case "substring":
		s := asString()
		start := 0
		if n, ok := toFloat(xf["start"]); ok {
			start = int(n) - 1
		}
		if start < 0 {
			start = 0
		}
		if xf["length"] != nil {
			if n, ok := toFloat(xf["length"]); ok {
				end := start + int(n)
				if end > len(s) {
					end = len(s)
				}
				if start > len(s) {
					return "", nil
				}
				return s[start:end], nil
			}
		}
		if start > len(s) {
			return "", nil
		}
		return s[start:], nil
	case "replace":
		from, _ := xf["from"].(string)
		to, _ := xf["to"].(string)
		return strings.ReplaceAll(asString(), from, to), nil
	case "split":
		sep, _ := xf["separator"].(string)
		parts := strings.Split(asString(), sep)
		out := make([]any, len(parts))
		for i, p := range parts {
			out[i] = p
		}
		return out, nil
	case "join":
		list, ok := asList(resolved)
		if !ok {
			throw(Failure{QueryFailure: "FailedTransformer", TransformerPath: pathAny(a.Path), FailureOrigin: []any{"handleTransformer_stringOp"}, FailureMessage: "stringOp join: applyTo is not an array, got: " + fmt.Sprintf("%T", resolved)})
		}
		sep, _ := xf["separator"].(string)
		parts := make([]string, len(list))
		for i, p := range list {
			parts[i] = jsString(p)
		}
		return strings.Join(parts, sep), nil
	default:
		throw(Failure{QueryFailure: "FailedTransformer", TransformerPath: pathAny(a.Path), FailureOrigin: []any{"handleTransformer_stringOp"}, FailureMessage: "stringOp: unknown op: " + op})
		return nil, nil
	}
}

func handleNumericOp(a ApplyArgs, xf map[string]any) (any, error) {
	op, _ := xf["operator"].(string)
	if op == "" {
		op, _ = xf["op"].(string)
	}
	args, _ := xf["args"].([]any)
	if len(args) == 0 {
		return 0, nil
	}
	vals := make([]float64, len(args))
	for i, arg := range args {
		v := mustApply(a, fmt.Sprintf("args.%d", i), arg)
		n, ok := toFloat(v)
		if !ok {
			throw(Failure{QueryFailure: "FailedTransformer", TransformerPath: pathAny(a.Path), FailureOrigin: []any{"handleTransformer_numericOp"}, FailureMessage: fmt.Sprintf("numericOp operand %d is not a number", i)})
		}
		vals[i] = n
	}
	result := vals[0]
	for _, n := range vals[1:] {
		switch op {
		case "-":
			result -= n
		case "*":
			result *= n
		case "/":
			result /= n
		default:
			result -= n
		}
	}
	return result, nil
}

func handleMenuAddItem(a ApplyArgs, xf map[string]any) (any, error) {
	var menu any
	if s, ok := xf["menuReference"].(string); ok {
		menu = mustApply(a, "menuReference", map[string]any{"transformerType": "getFromContext", "interpolation": "runtime", "referenceName": s})
	} else {
		menu = mustApply(a, "menuReference", xf["menuReference"])
	}
	var item any
	if s, ok := xf["menuItemReference"].(string); ok {
		item = mustApply(a, "menuItemReference", map[string]any{"transformerType": "getFromContext", "interpolation": "runtime", "referenceName": s})
	} else {
		item = mustApply(a, "menuItemReference", xf["menuItemReference"])
	}
	section := 0
	if n, ok := toFloat(xf["menuSectionInsertionIndex"]); ok {
		section = int(n)
	}
	itemIndex := 0
	if n, ok := toFloat(xf["menuSectionItemInsertionIndex"]); ok {
		itemIndex = int(n)
	}
	mm := copyMap(mustMap(menu))
	def := copyMap(mustMap(mm["definition"]))
	sections, _ := def["definition"].([]any)
	if section < 0 || section >= len(sections) {
		mm["definition"] = def
		return mm, nil
	}
	sec := copyMap(mustMap(sections[section]))
	items, _ := sec["items"].([]any)
	insert := itemIndex
	if itemIndex < 0 {
		if itemIndex == -1 {
			insert = len(items)
		} else {
			insert = itemIndex - 1
		}
	}
	if insert < 0 {
		insert = 0
	}
	if insert > len(items) {
		insert = len(items)
	}
	newItems := append([]any{}, items[:insert]...)
	newItems = append(newItems, item)
	newItems = append(newItems, items[insert:]...)
	sec["items"] = newItems
	newSections := append([]any{}, sections...)
	newSections[section] = sec
	def["definition"] = newSections
	mm["definition"] = def
	return mm, nil
}

func handleDuplicateApplicationModel(a ApplyArgs, xf map[string]any) (any, error) {
	var newUUID string
	if isTypedTransformer(xf["application"]) {
		newUUID = fmt.Sprint(mustApply(a, "application", xf["application"]))
	} else if s, ok := xf["application"].(string); ok {
		newUUID = s
	}
	if newUUID == "" || newUUID == "<nil>" {
		throw(Failure{QueryFailure: "FailedTransformer", TransformerPath: append(pathAny(a.Path), "application"), FailureOrigin: []any{"handleTransformer_duplicateApplicationModel"}, FailureMessage: "handleTransformer_duplicateApplicationModel failed to resolve application UUID"})
	}
	bundle := mustMap(mustApply(a, "applicationBundle", xf["applicationBundle"]))
	oldUUID := fmt.Sprint(bundle["applicationUuid"])
	out := copyMap(bundle)
	out["applicationUuid"] = newUUID
	replaceUUIDDeep(out, oldUUID, newUUID)
	if apps, ok := out["applications"].([]any); ok {
		for i, app := range apps {
			am := copyMap(mustMap(app))
			am["uuid"] = newUUID
			if s, ok := am["homePageUrl"].(string); ok && oldUUID != "" {
				am["homePageUrl"] = strings.ReplaceAll(s, oldUUID, newUUID)
			}
			apps[i] = am
		}
		out["applications"] = apps
	}
	if ents, ok := out["entities"].([]any); ok {
		for i, e := range ents {
			em := copyMap(mustMap(e))
			em["selfApplication"] = newUUID
			ents[i] = em
		}
		out["entities"] = ents
	}
	return out, nil
}

func replaceUUIDDeep(v any, oldUUID, newUUID string) {
	if oldUUID == "" || oldUUID == "<nil>" {
		return
	}
	switch t := v.(type) {
	case map[string]any:
		for k, item := range t {
			if s, ok := item.(string); ok {
				t[k] = strings.ReplaceAll(s, oldUUID, newUUID)
			} else {
				replaceUUIDDeep(item, oldUUID, newUUID)
			}
		}
	case []any:
		for _, item := range t {
			replaceUUIDDeep(item, oldUUID, newUUID)
		}
	}
}

func handleDefaultValueForMLSchema(a ApplyArgs, xf map[string]any) (any, error) {
	return defaultValueForSchema(xf["mlSchema"], a), nil
}

func handleUnfoldSchemaOnce(a ApplyArgs, xf map[string]any) (any, error) {
	return unfoldSchemaOnce(xf["mlSchema"], a), nil
}

func handleResolveConditionalSchema(a ApplyArgs, xf map[string]any) (any, error) {
	return resolveConditionalSchema(xf, a), nil
}

func handleResolveSchemaReferenceInContext(a ApplyArgs, xf map[string]any) (any, error) {
	return resolveSchemaReferenceInContext(xf, a), nil
}
