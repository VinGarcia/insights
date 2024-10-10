package eparser

import (
	"testing"

	"github.com/vingarcia/insights/internal/adapters/evaluator"
)

func TestParse(t *testing.T) {
	// This Test function runs all interface tests at once:
	evaluator.Test(t, func(expr string) (evaluator.Expression, error) {
		return Parse(expr)
	})
}
