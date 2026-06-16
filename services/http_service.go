package services

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// HTTPRequestOptions configures one HTTP request.
type HTTPRequestOptions struct {
	Method         string
	URL            string
	Headers        map[string]string
	Body           []byte
	Timeout        time.Duration
	ExpectedStatus []int
	RetryAttempts  int
	RetryDelay     time.Duration
	Backoff        bool
	InsecureTLS    bool
}

// HTTPResult contains a completed HTTP response.
type HTTPResult struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
}

// ExecuteHTTPRequest runs an HTTP request, optionally retrying failed attempts.
func ExecuteHTTPRequest(opts HTTPRequestOptions) (*HTTPResult, error) {
	if opts.Method == "" {
		opts.Method = http.MethodGet
	}
	if opts.Timeout <= 0 {
		opts.Timeout = 30 * time.Second
	}
	if opts.RetryAttempts <= 0 {
		opts.RetryAttempts = 1
	}
	delay := opts.RetryDelay
	if delay <= 0 {
		delay = time.Second
	}

	var lastErr error
	for attempt := 1; attempt <= opts.RetryAttempts; attempt++ {
		res, err := doHTTPRequest(opts)
		if err == nil {
			return res, nil
		}
		lastErr = err
		if attempt == opts.RetryAttempts {
			break
		}
		time.Sleep(delay)
		if opts.Backoff {
			delay *= 2
		}
	}
	return nil, lastErr
}

func doHTTPRequest(opts HTTPRequestOptions) (*HTTPResult, error) {
	req, err := http.NewRequest(strings.ToUpper(opts.Method), opts.URL, bytes.NewReader(opts.Body))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	for k, v := range opts.Headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{Timeout: opts.Timeout}
	if opts.InsecureTLS {
		client.Transport = &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}} //nolint:gosec
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("requesting %s: %w", opts.URL, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	result := &HTTPResult{StatusCode: resp.StatusCode, Headers: resp.Header.Clone(), Body: body}
	if len(opts.ExpectedStatus) > 0 && !statusAllowed(resp.StatusCode, opts.ExpectedStatus) {
		return result, fmt.Errorf("HTTP %s returned status %d, expected one of %v", opts.URL, resp.StatusCode, opts.ExpectedStatus)
	}
	return result, nil
}

// HTTPChainPlan defines a sequence of dependent HTTP requests.
type HTTPChainPlan struct {
	Steps []HTTPChainStep `json:"steps"`
}

// HTTPChainStep defines one request in an HTTP chain.
type HTTPChainStep struct {
	Name           string            `json:"name"`
	Method         string            `json:"method"`
	URL            string            `json:"url"`
	Headers        map[string]string `json:"headers"`
	Data           string            `json:"data"`
	JSON           string            `json:"json"`
	ExpectStatus   []int             `json:"expectStatus"`
	Capture        map[string]string `json:"capture"`
	TimeoutSeconds int               `json:"timeoutSeconds"`
}

// HTTPChainResult is the structured output from a chain run.
type HTTPChainResult struct {
	Vars  map[string]string     `json:"vars"`
	Steps []HTTPChainStepResult `json:"steps"`
}

// HTTPChainStepResult records one step outcome.
type HTTPChainStepResult struct {
	Name       string `json:"name"`
	StatusCode int    `json:"statusCode"`
}

// DecodeHTTPChainPlan converts decoded JSON/YAML/TOML data into a chain plan.
func DecodeHTTPChainPlan(v interface{}) (HTTPChainPlan, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return HTTPChainPlan{}, err
	}
	var plan HTTPChainPlan
	if err := json.Unmarshal(b, &plan); err != nil {
		return HTTPChainPlan{}, fmt.Errorf("decoding HTTP chain plan: %w", err)
	}
	if len(plan.Steps) == 0 {
		return HTTPChainPlan{}, fmt.Errorf("HTTP chain requires at least one step")
	}
	return plan, nil
}

// ExecuteHTTPChain runs a dependent sequence of HTTP requests.
func ExecuteHTTPChain(plan HTTPChainPlan, base HTTPRequestOptions) (*HTTPChainResult, error) {
	vars := map[string]string{}
	result := &HTTPChainResult{Vars: vars}

	for i, step := range plan.Steps {
		name := step.Name
		if name == "" {
			name = fmt.Sprintf("step%d", i+1)
		}
		method := step.Method
		if method == "" {
			method = http.MethodGet
		}
		headers := make(map[string]string, len(base.Headers)+len(step.Headers))
		for k, v := range base.Headers {
			headers[k] = interpolateHTTPVars(v, vars)
		}
		for k, v := range step.Headers {
			headers[k] = interpolateHTTPVars(v, vars)
		}
		body, contentType, err := chainStepBody(step, vars)
		if err != nil {
			return result, fmt.Errorf("%s: %w", name, err)
		}
		if contentType != "" {
			if _, exists := headers["Content-Type"]; !exists {
				headers["Content-Type"] = contentType
			}
		}
		timeout := base.Timeout
		if step.TimeoutSeconds > 0 {
			timeout = time.Duration(step.TimeoutSeconds) * time.Second
		}
		expected := step.ExpectStatus
		if len(expected) == 0 {
			expected = base.ExpectedStatus
		}

		res, err := ExecuteHTTPRequest(HTTPRequestOptions{
			Method:         method,
			URL:            interpolateHTTPVars(step.URL, vars),
			Headers:        headers,
			Body:           body,
			Timeout:        timeout,
			ExpectedStatus: expected,
			RetryAttempts:  base.RetryAttempts,
			RetryDelay:     base.RetryDelay,
			Backoff:        base.Backoff,
			InsecureTLS:    base.InsecureTLS,
		})
		if err != nil {
			return result, fmt.Errorf("%s: %w", name, err)
		}
		result.Steps = append(result.Steps, HTTPChainStepResult{Name: name, StatusCode: res.StatusCode})
		for key, path := range step.Capture {
			val, err := ExtractHTTPJSON(res.Body, path)
			if err != nil {
				return result, fmt.Errorf("%s capture %s: %w", name, key, err)
			}
			vars[key] = fmt.Sprint(val)
		}
	}
	return result, nil
}

func chainStepBody(step HTTPChainStep, vars map[string]string) ([]byte, string, error) {
	if step.Data != "" && step.JSON != "" {
		return nil, "", fmt.Errorf("use only one of data or json")
	}
	if step.Data != "" {
		return []byte(interpolateHTTPVars(step.Data, vars)), "", nil
	}
	if step.JSON != "" {
		body := []byte(interpolateHTTPVars(step.JSON, vars))
		if !json.Valid(body) {
			return nil, "", fmt.Errorf("json body is not valid JSON after interpolation")
		}
		return body, "application/json", nil
	}
	return nil, "", nil
}

func interpolateHTTPVars(value string, vars map[string]string) string {
	out := value
	for k, v := range vars {
		out = strings.ReplaceAll(out, "{{"+k+"}}", v)
	}
	return out
}

func statusAllowed(got int, expected []int) bool {
	for _, code := range expected {
		if got == code {
			return true
		}
	}
	return false
}

// ExtractHTTPJSON parses response JSON and returns the value at a jq-style path.
func ExtractHTTPJSON(body []byte, path string) (interface{}, error) {
	var doc interface{}
	if err := json.Unmarshal(body, &doc); err != nil {
		return nil, fmt.Errorf("decoding response JSON: %w", err)
	}
	return JSONGet(doc, path)
}

// ParseHTTPHeaders parses repeated "Name: value" header flags.
func ParseHTTPHeaders(headers []string) (map[string]string, error) {
	out := make(map[string]string, len(headers))
	for _, h := range headers {
		name, value, ok := strings.Cut(h, ":")
		if !ok || strings.TrimSpace(name) == "" {
			return nil, fmt.Errorf("invalid header %q, expected 'Name: value'", h)
		}
		out[strings.TrimSpace(name)] = strings.TrimSpace(value)
	}
	return out, nil
}

// BuildHTTPBody builds a request body from data, JSON, file, or form inputs.
func BuildHTTPBody(data string, dataFile string, jsonValue string, jsonFile string, formFields []string) ([]byte, string, error) {
	set := 0
	for _, v := range []string{data, dataFile, jsonValue, jsonFile} {
		if v != "" {
			set++
		}
	}
	if len(formFields) > 0 {
		set++
	}
	if set > 1 {
		return nil, "", fmt.Errorf("use only one of --data, --data-file, --json, --json-file, or --form")
	}

	switch {
	case data != "":
		return []byte(data), "", nil
	case dataFile != "":
		b, err := os.ReadFile(dataFile)
		if err != nil {
			return nil, "", fmt.Errorf("reading %s: %w", dataFile, err)
		}
		return b, "", nil
	case jsonValue != "":
		if !json.Valid([]byte(jsonValue)) {
			return nil, "", fmt.Errorf("--json is not valid JSON")
		}
		return []byte(jsonValue), "application/json", nil
	case jsonFile != "":
		b, err := os.ReadFile(jsonFile)
		if err != nil {
			return nil, "", fmt.Errorf("reading %s: %w", jsonFile, err)
		}
		if !json.Valid(b) {
			return nil, "", fmt.Errorf("%s is not valid JSON", jsonFile)
		}
		return b, "application/json", nil
	case len(formFields) > 0:
		values := url.Values{}
		for _, field := range formFields {
			k, v, ok := strings.Cut(field, "=")
			if !ok || k == "" {
				return nil, "", fmt.Errorf("invalid form field %q, expected key=value", field)
			}
			values.Add(k, v)
		}
		return []byte(values.Encode()), "application/x-www-form-urlencoded", nil
	default:
		return nil, "", nil
	}
}

// BuildMultipartBody builds a multipart/form-data request body from key=path pairs.
func BuildMultipartBody(fileFields []string) ([]byte, string, error) {
	if len(fileFields) == 0 {
		return nil, "", nil
	}
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	for _, field := range fileFields {
		name, path, ok := strings.Cut(field, "=")
		if !ok || name == "" || path == "" {
			return nil, "", fmt.Errorf("invalid file field %q, expected name=path", field)
		}
		f, err := os.Open(path)
		if err != nil {
			return nil, "", fmt.Errorf("opening %s: %w", path, err)
		}
		part, err := w.CreateFormFile(name, path)
		if err != nil {
			f.Close()
			return nil, "", err
		}
		if _, err := io.Copy(part, f); err != nil {
			f.Close()
			return nil, "", err
		}
		f.Close()
	}
	if err := w.Close(); err != nil {
		return nil, "", err
	}
	return b.Bytes(), w.FormDataContentType(), nil
}
