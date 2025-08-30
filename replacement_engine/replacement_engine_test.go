package replacement_engine

import "testing"

func TestReplaceInStringEmpty(t *testing.T) {
	t.Run("Valid run", func(t *testing.T) {
		expected := "none"
		source := "none"
		replacements := map[string]string{"STRING_REPLACEMENT_1": "one"}
		received, err := ReplaceInString(source, replacements)
		if err != nil {
			t.Errorf("%v", err)
		}
		if received != expected {
			t.Errorf("expected: %s, received %s", expected, received)
		}
	})
}

func TestReplaceInStringBasic(t *testing.T) {
	t.Run("Valid run", func(t *testing.T) {
		expected := "number one"
		source := "number STRING_REPLACEMENT_1"
		replacements := map[string]string{"STRING_REPLACEMENT_1": "one"}
		received, err := ReplaceInString(source, replacements)
		if err != nil {
			t.Errorf("%v", err)
			return
		}
		if received != expected {
			t.Errorf("expected: %s, received %s", expected, received)
			return
		}
	})
}
