package transformer

import "fmt"

// Failure is a TransformerFailure / queryFailure value. Map includes
// elementType "failure" so the unit runner does not re-apply it at runtime.
type Failure struct {
	QueryFailure    string
	TransformerPath []any
	FailureOrigin   []any
	QueryContext    any
	QueryParameters any
	FailureMessage  string
	QueryReference  any
	InnerError      any
}

// Error implements error.
func (f Failure) Error() string {
	if f.FailureMessage != "" {
		return f.FailureMessage
	}
	if f.QueryContext != nil {
		return fmt.Sprint(f.QueryContext)
	}
	return f.QueryFailure
}

// Map is the JSON object compared by unit transformerTest (queryFailure,
// elementType, and optional diagnostic fields).
func (f Failure) Map() map[string]any {
	out := map[string]any{"queryFailure": f.QueryFailure, "elementType": "failure"}
	if f.TransformerPath != nil {
		out["transformerPath"] = f.TransformerPath
	}
	if f.FailureOrigin != nil {
		out["failureOrigin"] = f.FailureOrigin
	}
	if f.QueryContext != nil {
		out["queryContext"] = f.QueryContext
	}
	if f.QueryParameters != nil {
		out["queryParameters"] = f.QueryParameters
	}
	if f.FailureMessage != "" {
		out["failureMessage"] = f.FailureMessage
	}
	if f.QueryReference != nil {
		out["queryReference"] = f.QueryReference
	}
	if f.InnerError != nil {
		out["innerError"] = f.InnerError
	}
	return out
}

// AsFailure reports whether v is a [Failure] (value or pointer).
func AsFailure(v any) (Failure, bool) {
	switch t := v.(type) {
	case Failure:
		return t, true
	case *Failure:
		if t == nil {
			return Failure{}, false
		}
		return *t, true
	default:
		return Failure{}, false
	}
}

func throw(f Failure) {
	panic(f)
}

func pathToAny(path []string) []any {
	out := make([]any, len(path))
	for i, p := range path {
		out[i] = p
	}
	return out
}
