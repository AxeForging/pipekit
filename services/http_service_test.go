package services

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestExecuteHTTPRequestExtractJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-Test"); got != "yes" {
			t.Fatalf("header = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"release":{"tag":"v1.2.3"}}`))
	}))
	defer srv.Close()

	res, err := ExecuteHTTPRequest(HTTPRequestOptions{
		Method:         http.MethodGet,
		URL:            srv.URL,
		Headers:        map[string]string{"X-Test": "yes"},
		Timeout:        2 * time.Second,
		ExpectedStatus: []int{200},
	})
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	val, err := ExtractHTTPJSON(res.Body, ".release.tag")
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if val != "v1.2.3" {
		t.Fatalf("tag = %v", val)
	}
}

func TestExecuteHTTPRequestRetriesStatusFailures(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	res, err := ExecuteHTTPRequest(HTTPRequestOptions{
		Method:         http.MethodPost,
		URL:            srv.URL,
		Timeout:        2 * time.Second,
		ExpectedStatus: []int{201},
		RetryAttempts:  3,
		RetryDelay:     time.Millisecond,
	})
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d", res.StatusCode)
	}
	if calls.Load() != 3 {
		t.Fatalf("calls = %d", calls.Load())
	}
}

func TestBuildHTTPBodyModes(t *testing.T) {
	body, contentType, err := BuildHTTPBody("", "", `{"ok":true}`, "", nil)
	if err != nil {
		t.Fatalf("json body: %v", err)
	}
	if contentType != "application/json" || !json.Valid(body) {
		t.Fatalf("unexpected json body: %s %s", contentType, body)
	}

	body, contentType, err = BuildHTTPBody("", "", "", "", []string{"a=1", "b=two words"})
	if err != nil {
		t.Fatalf("form body: %v", err)
	}
	if contentType != "application/x-www-form-urlencoded" || string(body) != "a=1&b=two+words" {
		t.Fatalf("unexpected form body: %s %s", contentType, body)
	}
}

func TestExecuteHTTPChainCapturesAndInterpolates(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			_, _ = w.Write([]byte(`{"token":"abc123"}`))
		case "/items/abc123":
			if got := r.Header.Get("Authorization"); got != "Bearer abc123" {
				t.Fatalf("Authorization = %q", got)
			}
			_, _ = w.Write([]byte(`{"id":42}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	result, err := ExecuteHTTPChain(HTTPChainPlan{Steps: []HTTPChainStep{
		{
			Name:         "auth",
			Method:       "GET",
			URL:          srv.URL + "/token",
			ExpectStatus: []int{200},
			Capture:      map[string]string{"token": ".token"},
		},
		{
			Name:         "item",
			Method:       "GET",
			URL:          srv.URL + "/items/{{token}}",
			Headers:      map[string]string{"Authorization": "Bearer {{token}}"},
			ExpectStatus: []int{200},
			Capture:      map[string]string{"item_id": ".id"},
		},
	}}, HTTPRequestOptions{Timeout: 2 * time.Second})
	if err != nil {
		t.Fatalf("chain: %v", err)
	}
	if result.Vars["token"] != "abc123" || result.Vars["item_id"] != "42" {
		t.Fatalf("vars = %#v", result.Vars)
	}
	if len(result.Steps) != 2 {
		t.Fatalf("steps = %d", len(result.Steps))
	}
}

func TestParseHTTPHeadersRejectsInvalid(t *testing.T) {
	_, err := ParseHTTPHeaders([]string{"not a header"})
	if err == nil || !strings.Contains(err.Error(), "invalid header") {
		t.Fatalf("expected invalid header error, got %v", err)
	}
}
