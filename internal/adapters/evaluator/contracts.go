package evaluator

import "encoding/json"

// Expression represents a compiled boolean expression
// that can evaluate to true or false given a JSON
// input representing a nested set of a variables.
type Expression interface {
	Evaluate(logLine json.RawMessage) (bool, error)
}
