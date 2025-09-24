package common_utils

import "testing"

func TestStrPTR(t *testing.T) {
	t.Run("Valid run", func(t *testing.T) {
		src := "src"
		result := StrPTR(src)

		if *result != src {
			t.Errorf("expected: %s, received %p", src, result)
		}
	})
}
