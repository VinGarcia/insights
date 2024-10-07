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

func (e Evaluator) Evaluate(logLine json.RawMessage) (bool, error) {
	var m map[string]any
	err := json.Unmarshal(logLine, &m)
	if err != nil {
		return false, err
	}

	result, err := e.evaluator.Evaluate(m)
	if err != nil {
		return false, fmt.Errorf(
			"error evaluating expression '%v': %w, input logLine: '%s'",
			e.evaluator, err, string(logLine),
		)
	}

	return result, nil
}
