package endpoint

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/akesling/arke/codex"
	"github.com/akesling/arke/interchange"
	"golang.org/x/net/context"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
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

func TestHTTPPut(t *testing.T) {
}

func stripError(result interface{}, err error) interface{} {
	return result
}

func TestHandleTopicRequest(t *testing.T) {
	var tests = []struct {
		TestTitle string
		Request   *http.Request
		Response  http.Response
	}{
		{
			TestTitle: "SuccessfulPublication",
			Request: stripError(http.NewRequest(
				"POST",
				"foo.bar/topics/baz",
				ioutil.NopCloser(strings.NewReader("{}")))).(*http.Request),
			Response: http.Response{
				StatusCode:    http.StatusCreated,
				Header:        http.Header{},
				Body:          ioutil.NopCloser(strings.NewReader("")),
				ContentLength: -1,
			},
		},
		{
			TestTitle: "InvalidRequestBody",
			Request: stripError(http.NewRequest(
				"POST",
				"foo.bar/topics/baz",
				ioutil.NopCloser(strings.NewReader("this is not JSON")))).(*http.Request),
			Response: http.Response{
				StatusCode:    http.StatusBadRequest,
				Header:        http.Header{},
				Body:          ioutil.NopCloser(strings.NewReader("")),
				ContentLength: -1,
			},
		},
		{
			TestTitle: "InvalidHTTPMethod",
			Request: stripError(http.NewRequest(
				"GET",
				"foo.bar/topics/",
				ioutil.NopCloser(strings.NewReader("{}")))).(*http.Request),
			Response: http.Response{
				StatusCode:    http.StatusMethodNotAllowed,
				Header:        http.Header{},
				Body:          ioutil.NopCloser(strings.NewReader("")),
				ContentLength: -1,
			},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	hub := interchange.NewHub(ctx, cancel)

	logOutput, logInput := io.Pipe()
	logScanner := bufio.NewScanner(logOutput)
	testLogger := log.New(logInput, "", log.Lshortfile)
	endpoint := &httprest{hub: hub, codex: codex.JSONCodex{}, logger: testLogger}
	topicHandler := handleTopicRequest(endpoint, codex.JSONCodex{})

	for _, test := range tests {
		testRecorder := httptest.NewRecorder()
		test.Response.Request = test.Request
		topicHandler(testRecorder, test.Request)

		expectedStatusCode := test.Response.StatusCode
		actualStatusCode := testRecorder.Code
		if expectedStatusCode != actualStatusCode {
			for logScanner.Scan() {
				t.Log(logScanner.Text())
			}

			t.Error(fmt.Sprintf("Test%s: Expected HTTP status code (%d) did not match actual HTTP status code (%d).", test.TestTitle, expectedStatusCode, actualStatusCode))
		}

		expectedHeader := test.Response.Header
		actualHeader := testRecorder.Header
		if reflect.DeepEqual(expectedHeader, actualHeader) {
			for logScanner.Scan() {
				t.Log(logScanner.Text())
			}

			t.Error(fmt.Sprintf("Test%s: Expected header (%+v) did not match actual header (%+v) of the response.", test.TestTitle, expectedHeader, actualHeader))
		}

		expectedBodyBytes, err := ioutil.ReadAll(test.Response.Body)
		if err != nil {
			for logScanner.Scan() {
				t.Log(logScanner.Text())
			}

			t.Fatal(fmt.Sprintf("Test%s: Expected response body failed to be read.", test.TestTitle))
		}
		expectedBody := bytes.NewBuffer(expectedBodyBytes).String()
		actualBody := testRecorder.Body.String()
		if expectedBody != actualBody {
			t.Error(fmt.Sprintf("Test%s: Expected body (%q) did not match actual body (%q) of the response.", test.TestTitle, expectedBody, actualBody))
		}
	}
}

/*
func TestHandleSubscriptionRequest(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	hub := interchange.NewHub(ctx, cancel)

	endpoint := NewHTTPRestEndpoint(interchange.NewClient(hub), codex.JSONCodex{})
}

func TestSubscribeWorks(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	hub := interchange.NewHub(ctx, cancel)

	endpoint := NewHTTPRestEndpoint(interchange.NewClient(hub), codex.JSONCodex{})
	port := 8080
	err := endpoint.SetPort(port)
	if err != nil {
		t.Fatal(err)
	}

	_, err := endpoint.Start()
	if err != nil {
		t.Fatal(err)
	}
}
*/
