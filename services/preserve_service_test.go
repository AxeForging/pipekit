package services

import (
	"strings"
	"testing"
)

// ---- YAML preserving set ----

func TestSetYAMLPreserving_KeepsCommentsOrderAndSiblings(t *testing.T) {
	in := []byte(`# Application config
image:
  repository: myapp # the repo
  tag: v1.0.0
replicas: 3
`)
	out, err := setYAMLPreserving(in, ".image.tag", "v2.0.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := string(out)
	for _, want := range []string{"# Application config", "# the repo", "tag: v2.0.0", "repository: myapp", "replicas: 3"} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in output:\n%s", want, got)
		}
	}
	if strings.Contains(got, "v1.0.0") {
		t.Errorf("old value should be gone:\n%s", got)
	}
	// Order preserved: image block before replicas.
	if strings.Index(got, "image:") > strings.Index(got, "replicas:") {
		t.Errorf("key order changed:\n%s", got)
	}
}

func TestSetYAMLPreserving_KeepsDoubleQuoteStyle(t *testing.T) {
	in := []byte("tag: \"v1.0.0\"\n")
	out, err := setYAMLPreserving(in, ".tag", "v2.0.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := string(out); !strings.Contains(got, `tag: "v2.0.0"`) {
		t.Errorf("expected quoting preserved, got: %s", got)
	}
}

func TestSetYAMLPreserving_KeepsPlainStyle(t *testing.T) {
	in := []byte("tag: v1.0.0\n")
	out, err := setYAMLPreserving(in, ".tag", "v2.0.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := string(out)
	if !strings.Contains(got, "tag: v2.0.0") || strings.Contains(got, `"v2.0.0"`) {
		t.Errorf("expected plain style preserved, got: %s", got)
	}
}

func TestSetYAMLPreserving_Number(t *testing.T) {
	in := []byte("replicas: 3\n")
	out, err := setYAMLPreserving(in, ".replicas", float64(5))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := string(out); !strings.Contains(got, "replicas: 5") {
		t.Errorf("got: %s", got)
	}
}

func TestSetYAMLPreserving_ArrayIndex(t *testing.T) {
	in := []byte("hosts:\n  - a\n  - b\n  - c\n")
	out, err := setYAMLPreserving(in, ".hosts[1]", "z")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := string(out)
	if !strings.Contains(got, "- z") || strings.Contains(got, "- b") {
		t.Errorf("array element not updated: %s", got)
	}
	if !strings.Contains(got, "- a") || !strings.Contains(got, "- c") {
		t.Errorf("siblings lost: %s", got)
	}
}

func TestSetYAMLPreserving_AddsNewKey(t *testing.T) {
	in := []byte("image:\n  tag: v1\n")
	out, err := setYAMLPreserving(in, ".image.pullPolicy", "Always")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := string(out)
	if !strings.Contains(got, "tag: v1") || !strings.Contains(got, "pullPolicy: Always") {
		t.Errorf("new key not added cleanly: %s", got)
	}
}

// TestSetYAMLPreserving_InsertKeepsMultilineFlowStyle reproduces the original
// regression: inserting a new scalar key must NOT collapse multi-line flow-style
// collections ([ ... ] / { ... } spread across lines) elsewhere in the document.
// The whole file outside the single inserted line stays byte-for-byte identical.
func TestSetYAMLPreserving_InsertKeepsMultilineFlowStyle(t *testing.T) {
	in := []byte("deployment:\n" +
		"  args: [\n" +
		"    '--config',\n" +
		"    '/etc/app.yaml',\n" +
		"    '--verbose'\n" +
		"  ]\n" +
		"  annotations: {\n" +
		"    team: platform,\n" +
		"    tier: backend\n" +
		"  }\n" +
		"image:\n" +
		"  repository: myapp\n")
	want := "deployment:\n" +
		"  args: [\n" +
		"    '--config',\n" +
		"    '/etc/app.yaml',\n" +
		"    '--verbose'\n" +
		"  ]\n" +
		"  annotations: {\n" +
		"    team: platform,\n" +
		"    tier: backend\n" +
		"  }\n" +
		"image:\n" +
		"  repository: myapp\n" +
		"  tag: v1.4.0\n"
	out, err := setYAMLPreserving(in, ".image.tag", "v1.4.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(out) != want {
		t.Errorf("multi-line flow style not preserved on insert:\ngot:\n%q\nwant:\n%q", out, want)
	}
}

// TestSetYAMLPreserving_InsertByteForByte covers blank lines, comment column
// alignment and quoting around the insertion point.
func TestSetYAMLPreserving_InsertByteForByte(t *testing.T) {
	in := []byte("# Chart values\n" +
		"image:\n" +
		"  repository: myapp     # registry path\n" +
		"\n" +
		"replicas: 3\n")
	want := "# Chart values\n" +
		"image:\n" +
		"  repository: myapp     # registry path\n" +
		"  tag: v2.0.0\n" +
		"\n" +
		"replicas: 3\n"
	out, err := setYAMLPreserving(in, ".image.tag", "v2.0.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(out) != want {
		t.Errorf("insert not byte-for-byte:\ngot:\n%q\nwant:\n%q", out, want)
	}
}

func TestSetYAMLPreserving_InsertTopLevelKey(t *testing.T) {
	in := []byte("a: 1\nb: 2\n")
	want := "a: 1\nb: 2\nc: 3\n"
	out, err := setYAMLPreserving(in, ".c", float64(3))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(out) != want {
		t.Errorf("got %q want %q", out, want)
	}
}

func TestSetYAMLPreserving_InsertNoTrailingNewline(t *testing.T) {
	in := []byte("image:\n  repository: myapp")
	want := "image:\n  repository: myapp\n  tag: v1\n"
	out, err := setYAMLPreserving(in, ".image.tag", "v1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(out) != want {
		t.Errorf("got %q want %q", out, want)
	}
}

// A string that YAML would otherwise read as a bool/number must be quoted on
// insert so it round-trips as a string.
func TestSetYAMLPreserving_InsertQuotesAmbiguousString(t *testing.T) {
	in := []byte("flags:\n  verbose: true\n")
	out, err := setYAMLPreserving(in, ".flags.enabled", "yes")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := string(out)
	if !strings.Contains(got, "enabled: \"yes\"") {
		t.Errorf("ambiguous string not quoted on insert: %q", got)
	}
	if !yamlPathHasValue(out, ".flags.enabled", "yes") {
		t.Errorf("inserted value does not round-trip as string: %q", got)
	}
}

// Inserting into a deeper block must not disturb a multi-line flow sibling that
// comes after it in the document.
func TestSetYAMLPreserving_InsertBeforeFlowSibling(t *testing.T) {
	in := []byte("image:\n" +
		"  repository: myapp\n" +
		"ports: [\n" +
		"  8080,\n" +
		"  9090\n" +
		"]\n")
	want := "image:\n" +
		"  repository: myapp\n" +
		"  tag: v1\n" +
		"ports: [\n" +
		"  8080,\n" +
		"  9090\n" +
		"]\n"
	out, err := setYAMLPreserving(in, ".image.tag", "v1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(out) != want {
		t.Errorf("got:\n%q\nwant:\n%q", out, want)
	}
}

// When an intermediate map is missing too, the whole missing tail is created as
// a nested block — and unrelated multi-line flow style is still left untouched.
func TestSetYAMLPreserving_InsertCreatesNestedBlock(t *testing.T) {
	in := []byte("service:\n" +
		"  ports: [\n" +
		"    80,\n" +
		"    443\n" +
		"  ]\n")
	want := "service:\n" +
		"  ports: [\n" +
		"    80,\n" +
		"    443\n" +
		"  ]\n" +
		"image:\n" +
		"  tag: v1.4.0\n"
	out, err := setYAMLPreserving(in, ".image.tag", "v1.4.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(out) != want {
		t.Errorf("nested create not surgical:\ngot:\n%q\nwant:\n%q", out, want)
	}
}

func TestSetYAMLPreserving_InsertCreatesDeepNestedBlock(t *testing.T) {
	in := []byte("name: app\n")
	want := "name: app\n" +
		"a:\n" +
		"  b:\n" +
		"    c: 5\n"
	out, err := setYAMLPreserving(in, ".a.b.c", float64(5))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(out) != want {
		t.Errorf("deep nested create not surgical:\ngot:\n%q\nwant:\n%q", out, want)
	}
}

// Indexing through a scalar is impossible; the surgical path must defer to the
// fallback, which surfaces the error rather than producing wrong output.
func TestSetYAMLPreserving_InsertUnderScalarErrors(t *testing.T) {
	in := []byte("image: myapp\n")
	if _, err := setYAMLPreserving(in, ".image.tag", "v1"); err == nil {
		t.Fatal("expected error inserting under a scalar value")
	}
}

func TestSetYAMLPreserving_RejectsIndexIntoScalar(t *testing.T) {
	in := []byte("tag: v1.0.0\n")
	if _, err := setYAMLPreserving(in, ".tag.inner", "x"); err == nil {
		t.Fatal("expected error indexing into scalar")
	}
}

// TestSetYAMLPreserving_ByteForByte proves that a value edit changes ONLY the
// target bytes: comment column alignment, blank lines, and indentation are all
// kept exactly.
func TestSetYAMLPreserving_ByteForByte(t *testing.T) {
	in := []byte("# Chart values\n" +
		"image:\n" +
		"  repository: myapp     # registry path\n" +
		"  tag:        v1.0.0    # bump me\n" +
		"\n" +
		"replicas: 3\n")
	want := "# Chart values\n" +
		"image:\n" +
		"  repository: myapp     # registry path\n" +
		"  tag:        v2.0.0    # bump me\n" +
		"\n" +
		"replicas: 3\n"
	out, err := setYAMLPreserving(in, ".image.tag", "v2.0.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(out) != want {
		t.Errorf("not byte-for-byte preserved:\ngot:\n%q\nwant:\n%q", out, want)
	}
}

func TestSetYAMLPreserving_ByteForByteQuoted(t *testing.T) {
	in := []byte("tag:   \"v1.0.0\"   # keep alignment\n")
	want := "tag:   \"v2.0.0\"   # keep alignment\n"
	out, err := setYAMLPreserving(in, ".tag", "v2.0.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(out) != want {
		t.Errorf("got %q want %q", out, want)
	}
}

// A numeric-looking string set onto a plain field is type-ambiguous; the splice
// validation should reject it and the node-encode fallback keeps it correct.
func TestSetYAMLPreserving_NumericStringFallsBackSafely(t *testing.T) {
	out, err := setYAMLPreserving([]byte("port: 8080\n"), ".port", "9090")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := string(out); !strings.Contains(got, "9090") {
		t.Errorf("got %s", got)
	}
}

// ---- YAML preserving del ----

func TestDelYAMLPreserving_KeepsSiblingsAndComments(t *testing.T) {
	in := []byte(`image:
  repository: myapp # keep me
  tag: v1.0.0
replicas: 3
`)
	out, err := delYAMLPreserving(in, ".image.tag")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := string(out)
	if strings.Contains(got, "tag:") {
		t.Errorf("tag should be deleted: %s", got)
	}
	for _, want := range []string{"repository: myapp", "# keep me", "replicas: 3"} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q preserved: %s", want, got)
		}
	}
}

func TestDelYAMLPreserving_ByteForByte(t *testing.T) {
	in := []byte("# top\n" +
		"image:\n" +
		"  repository: myapp # keep\n" +
		"  tag: v1.0.0       # drop this line\n" +
		"replicas: 3\n")
	want := "# top\n" +
		"image:\n" +
		"  repository: myapp # keep\n" +
		"replicas: 3\n"
	out, err := delYAMLPreserving(in, ".image.tag")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(out) != want {
		t.Errorf("del not byte-for-byte:\ngot:\n%q\nwant:\n%q", out, want)
	}
}

func TestDelYAMLPreserving_MissingPathNoOp(t *testing.T) {
	in := []byte("a: 1\nb: 2\n")
	out, err := delYAMLPreserving(in, ".nope")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := string(out)
	if !strings.Contains(got, "a: 1") || !strings.Contains(got, "b: 2") {
		t.Errorf("missing-path del should be a no-op: %s", got)
	}
}

// ---- JSON preserving set (exact byte assertions) ----

func TestSetJSONPreserving_OnlyChangesTarget(t *testing.T) {
	in := []byte(`{
  "image": {
    "repo": "old",
    "tag": "v1.0.0"
  },
  "replicas": 3
}
`)
	want := `{
  "image": {
    "repo": "old",
    "tag": "v2.0.0"
  },
  "replicas": 3
}
`
	out, err := setJSONPreserving(in, ".image.tag", "v2.0.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(out) != want {
		t.Errorf("got:\n%s\nwant:\n%s", out, want)
	}
}

func TestSetJSONPreserving_ArrayIndex(t *testing.T) {
	in := []byte(`{"hosts": ["a", "b", "c"]}`)
	want := `{"hosts": ["a", "z", "c"]}`
	out, err := setJSONPreserving(in, ".hosts[1]", "z")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(out) != want {
		t.Errorf("got %s want %s", out, want)
	}
}

func TestSetJSONPreserving_NumberAndObject(t *testing.T) {
	in := []byte(`{"replicas": 3, "image": {"tag": "v1"}}`)
	out, err := setJSONPreserving(in, ".replicas", float64(5))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := string(out); got != `{"replicas": 5, "image": {"tag": "v1"}}` {
		t.Errorf("number set got: %s", got)
	}

	in2 := []byte(`{"replicas": 3, "image": {"tag": "v1"}}`)
	out2, err := setJSONPreserving(in2, ".image", map[string]interface{}{"tag": "v2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := string(out2); got != `{"replicas": 3, "image": {"tag":"v2"}}` {
		t.Errorf("object set got: %s", got)
	}
}

func TestSetJSONPreserving_InsertCompact(t *testing.T) {
	out, err := setJSONPreserving([]byte(`{"a":1}`), ".b", float64(2))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := string(out); got != `{"a":1,"b":2}` { // colon spacing mirrors the sibling
		t.Errorf("got %s", got)
	}
}

func TestSetJSONPreserving_InsertIntoEmptyObject(t *testing.T) {
	out, err := setJSONPreserving([]byte(`{}`), ".x", "y")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := string(out); got != `{"x": "y"}` {
		t.Errorf("got %s", got)
	}
}

func TestSetJSONPreserving_InsertPretty(t *testing.T) {
	in := []byte("{\n  \"a\": 1,\n  \"b\": 2\n}\n")
	want := "{\n  \"a\": 1,\n  \"b\": 2,\n  \"c\": 3\n}\n"
	out, err := setJSONPreserving(in, ".c", float64(3))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(out) != want {
		t.Errorf("got:\n%s\nwant:\n%s", out, want)
	}
}

func TestSetJSONPreserving_InsertNested(t *testing.T) {
	out, err := setJSONPreserving([]byte(`{"image":{"tag":"v1"},"x":1}`), ".image.repo", "r")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := string(out); got != `{"image":{"tag":"v1","repo":"r"},"x":1}` {
		t.Errorf("got %s", got)
	}
}

func TestSetJSONPreserving_InsertMissingParentErrors(t *testing.T) {
	if _, err := setJSONPreserving([]byte(`{"a":1}`), ".nope.deep", "x"); err == nil {
		t.Fatal("expected error when parent object does not exist")
	}
}

// ---- JSON preserving del (exact byte assertions) ----

func TestDelJSONPreserving_MiddleKey(t *testing.T) {
	out, err := delJSONPreserving([]byte(`{"a":1,"b":2,"c":3}`), ".b")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := string(out); got != `{"a":1,"c":3}` {
		t.Errorf("got %s", got)
	}
}

func TestDelJSONPreserving_LastKey(t *testing.T) {
	out, err := delJSONPreserving([]byte(`{"a":1,"b":2,"c":3}`), ".c")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := string(out); got != `{"a":1,"b":2}` {
		t.Errorf("got %s", got)
	}
}

func TestDelJSONPreserving_OnlyKey(t *testing.T) {
	out, err := delJSONPreserving([]byte(`{"a":1}`), ".a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := string(out); got != `{}` {
		t.Errorf("got %s", got)
	}
}

func TestDelJSONPreserving_Pretty(t *testing.T) {
	in := []byte(`{
  "a": 1,
  "b": 2,
  "c": 3
}
`)
	want := `{
  "a": 1,
  "c": 3
}
`
	out, err := delJSONPreserving(in, ".b")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(out) != want {
		t.Errorf("got:\n%s\nwant:\n%s", out, want)
	}
}

func TestDelJSONPreserving_ArrayElement(t *testing.T) {
	out, err := delJSONPreserving([]byte(`[10,20,30]`), ".1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := string(out); got != `[10,30]` {
		t.Errorf("got %s", got)
	}
}

func TestDelJSONPreserving_NestedTarget(t *testing.T) {
	in := []byte(`{"image":{"repo":"old","tag":"v1"},"keep":true}`)
	want := `{"image":{"repo":"old"},"keep":true}`
	out, err := delJSONPreserving(in, ".image.tag")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(out) != want {
		t.Errorf("got %s want %s", out, want)
	}
}

// ---- dispatch / format guards ----

func TestSetPreserving_UnsupportedFormats(t *testing.T) {
	if _, err := SetPreserving([]byte(`a = 1`), FormatTOML, ".a", "x"); err == nil {
		t.Error("expected TOML to be unsupported in preserve mode")
	}
	if _, err := DelPreserving([]byte(``), FormatCSV, ".a"); err == nil {
		t.Error("expected CSV to be unsupported in preserve mode")
	}
}

// ---- backward-compatibility: legacy round-trip path still works ----

func TestLegacyRoundTripStillReformats(t *testing.T) {
	doc, err := Decode([]byte("a: 1\nb: 2\n"), FormatYAML)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	updated, err := JSONSet(doc, ".a", "x")
	if err != nil {
		t.Fatalf("set: %v", err)
	}
	out, err := Encode(updated, FormatYAML, true)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if !strings.Contains(string(out), "a: x") {
		t.Errorf("legacy path broken: %s", out)
	}
}
