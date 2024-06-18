package bexpr

import (
	"testing"

	"github.com/vingarcia/insights/internal/adapters/evaluator"
)

func TestBexprEvaluatorInterface(t *testing.T) {
	// This Test function runs all interface tests at once:
	evaluator.Test(t, func(expr string) (evaluator.Expression, error) {
		return New(expr)
	})
}
