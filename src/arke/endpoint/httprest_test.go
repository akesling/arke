package endpoint

import (
	"fmt"
	"testing"
	"time"
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
			t.Error("decodeTopicURLPath unexpectedly returned error: %s", err)
		}
		if result != expectation {
			t.Error(
				fmt.Sprintf(
					"Input (%q) mapped to unexpected value (%q) instead of the desired topic (%q).",
					input, result, expectation))
		}
	}
}

func TestConstrainLease(t *testing.T) {
	input_output_table := map[time.Duration]time.Duration{
		time.Duration(-1):                   time.Duration(0),
		time.Duration(0):                    time.Duration(0),
		time.Duration(1):                    time.Duration(1),
		maxLeaseDuration - time.Duration(1): maxLeaseDuration - time.Duration(1),
		maxLeaseDuration:                    maxLeaseDuration,
		maxLeaseDuration + time.Duration(1): maxLeaseDuration,
	}

	for input := range input_output_table {
		expectation := input_output_table[input]
		result := constrainLease(input)
		if result != expectation {
			t.Error(
				fmt.Sprintf(
					"Input (%q) mapped to unexpected value (%q) instead of the desired topic (%q).",
					input, result, expectation))
		}
	}
}
