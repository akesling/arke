package endpoint

import (
	"fmt"
	"testing"
)

func TestDecodeTopicURLPath(t *testing.T) {
	input_output_table := map[string]string{
		"/":             ".",           // NULL path
		"////":          ".",           // Extra slashes
		"/foo/////":     "foo",         // Extra slashes
		"/foo/bar/baz/": "foo.bar.baz", // Normal path
		"/%20/baz/":     " .baz",       // Escaped path
	}

	for input := range input_output_table {
		expectation := input_output_table[input]
		result, err := decodeTopicURLPath(input)
		if err != nil {
			t.Error("decodeTopicURLPath unexpectedly return error: %s", err)
		}
		if result != expectation {
			t.Error(
				fmt.Sprintf(
					"Input (%q) mapped to unexpected value (%q) instead of the desired topic (%q).",
					input, result, expectation))
		}
	}
}
