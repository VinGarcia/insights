package bexpr

import (
	"encoding/json"
	"fmt"

	"github.com/hashicorp/go-bexpr"
)

type Evaluator struct {
	evaluator *bexpr.Evaluator
}

func New(expr string) (Evaluator, error) {
	evaluator, err := bexpr.CreateEvaluator(expr)
	if err != nil {
		return Evaluator{}, fmt.Errorf("error parsing expression '%s': %w", expr, err)
	}

	return Evaluator{
		evaluator: evaluator,
	}, nil
}

func (e Evaluator) Evaluate(vars map[string]any) (bool, error) {
	result, err := e.evaluator.Evaluate(vars)
	if err != nil {
		return false, fmt.Errorf(
			"error evaluating expression '%v': %w, input values: %s",
			e.evaluator, err, stringify(vars),
		)
	}

	return result, nil
}

func stringify(obj any) string {
	b, err := json.Marshal(obj)
	if err != nil {
		b = []byte(fmt.Sprintf("%+v", obj))
	}

	return string(b)
}
