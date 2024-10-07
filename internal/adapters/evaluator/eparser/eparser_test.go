package eparser

import (
	"testing"

	tt "github.com/vingarcia/insights/internal/testtools"
)

func TestParse(t *testing.T) {
	t.Run("should return errors for empty strings", func(t *testing.T) {
		_, err := Parse("")
		tt.AssertErrContains(t, err, "empty string")
	})
}
