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
		"/%20/baz/":     " .bar",       // Escaped path
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
					"Input (%s) mapped to unexpected value (%s) instead of the desired topic (%s).",
					input, result, expectation))
		}
	}
}
