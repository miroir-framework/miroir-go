package miroirtest

func registerUnique() {
	registry["miroir-core/1_core/tools"] = map[string]Fn{
		"pushIfUnique":           wrapPushIfUnique,
		"mergeIfUnique":          wrapMergeIfUnique,
		"pushIfUniqueReturning":  wrapPushIfUniqueReturning,
		"mergeIfUniqueReturning": wrapMergeIfUniqueReturning,
	}
}

func wrapPushIfUnique(args []any) (any, error) {
	pushIfUnique(asAnySlicePtr(args, 0), argAt(args, 1))
	return nil, nil
}

func wrapMergeIfUnique(args []any) (any, error) {
	items, _ := argAt(args, 1).([]any)
	mergeIfUnique(asAnySlicePtr(args, 0), items)
	return nil, nil
}

func wrapPushIfUniqueReturning(args []any) (any, error) {
	arr := asAnySlicePtr(args, 0)
	pushIfUnique(arr, argAt(args, 1))
	return *arr, nil
}

func wrapMergeIfUniqueReturning(args []any) (any, error) {
	arr := asAnySlicePtr(args, 0)
	items, _ := argAt(args, 1).([]any)
	mergeIfUnique(arr, items)
	return *arr, nil
}

func asAnySlicePtr(args []any, i int) *[]any {
	if i >= len(args) {
		empty := []any{}
		return &empty
	}
	if list, ok := args[i].([]any); ok {
		return &list
	}
	empty := []any{}
	return &empty
}

func pushIfUnique(array *[]any, item any) {
	for _, existing := range *array {
		if valuesEqual(existing, item, nil) {
			return
		}
	}
	*array = append(*array, item)
}

func mergeIfUnique(array *[]any, items []any) {
	for _, item := range items {
		pushIfUnique(array, item)
	}
}

func init() {
	registerUnique()
}
