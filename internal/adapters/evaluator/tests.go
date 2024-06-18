package evaluator

import (
	"testing"

	tt "github.com/vingarcia/insights/internal/testtools"
)

func Test(t *testing.T, factory func(expr string) (Expression, error)) {

	tests := []struct {
		expr               string
		vars               map[string]any
		expectedResult     bool
		expectErrToContain []string
	}{
		{
			expr: "a == 1",
			vars: map[string]any{
				"a": 1,
			},
			expectedResult: true,
		},
		{
			expr: "a != 0",
			vars: map[string]any{
				"a": 1,
			},
			expectedResult: true,
		},
	}

	for _, test := range tests {
		t.Run(test.expr, func(t *testing.T) {
			evaluator, err := factory(test.expr)
			tt.AssertNoErr(t, err)

			result, err := evaluator.Evaluate(test.vars)
			if test.expectErrToContain != nil {
				tt.AssertErrContains(t, err, test.expectErrToContain...)
				t.Skip()
			}
			tt.AssertNoErr(t, err)

			tt.AssertEqual(t, result, test.expectedResult)
		})
	}
}
