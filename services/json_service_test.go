package services

import (
	"reflect"
	"strings"
	"testing"
)

func TestJSONGet(t *testing.T) {
	doc := map[string]interface{}{
		"image": map[string]interface{}{"tag": "v1.2.3", "repo": "ghcr.io/me/app"},
		"items": []interface{}{
			map[string]interface{}{"name": "a"},
			map[string]interface{}{"name": "b"},
		},
	}
	tests := []struct {
		path string
		want interface{}
	}{
		{".image.tag", "v1.2.3"},
		{".items[0].name", "a"},
		{".items[1].name", "b"},
	}
	for _, tc := range tests {
		got, err := JSONGet(doc, tc.path)
		if err != nil {
			t.Errorf("JSONGet(%q) error: %v", tc.path, err)
			continue
		}
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("JSONGet(%q) = %v, want %v", tc.path, got, tc.want)
		}
	}
}

func TestJSONSet(t *testing.T) {
	doc := map[string]interface{}{
		"image": map[string]interface{}{"tag": "v1.0.0"},
	}
	updated, err := JSONSet(doc, ".image.tag", "v2.0.0")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	got, _ := JSONGet(updated, ".image.tag")
	if got != "v2.0.0" {
		t.Errorf("expected v2.0.0, got %v", got)
	}
	// Original is unchanged.
	if doc["image"].(map[string]interface{})["tag"] != "v1.0.0" {
		t.Error("original doc was mutated")
	}
}

func TestJSONSet_CreatesNestedPath(t *testing.T) {
	updated, err := JSONSet(map[string]interface{}{}, ".a.b.c", "x")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	got, _ := JSONGet(updated, ".a.b.c")
	if got != "x" {
		t.Errorf("expected x, got %v", got)
	}
}

func TestJSONDel(t *testing.T) {
	doc := map[string]interface{}{
		"a": "1",
		"b": "2",
	}
	updated, err := JSONDel(doc, ".a")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if _, ok := updated.(map[string]interface{})["a"]; ok {
		t.Error("a should be deleted")
	}
	if updated.(map[string]interface{})["b"] != "2" {
		t.Error("b should remain")
	}
}

func TestDeepMerge(t *testing.T) {
	base := map[string]interface{}{
		"image": map[string]interface{}{"tag": "v1.0.0", "repo": "old"},
		"keep":  "yes",
	}
	overlay := map[string]interface{}{
		"image": map[string]interface{}{"tag": "v2.0.0"},
		"new":   "added",
	}
	got := DeepMerge(base, overlay).(map[string]interface{})
	if got["keep"] != "yes" {
		t.Error("keep dropped")
	}
	if got["new"] != "added" {
		t.Error("new not added")
	}
	img := got["image"].(map[string]interface{})
	if img["tag"] != "v2.0.0" {
		t.Errorf("tag not overridden: %v", img["tag"])
	}
	if img["repo"] != "old" {
		t.Errorf("repo dropped during merge: %v", img["repo"])
	}
}

func TestEncodeDecode_RoundTrip(t *testing.T) {
	v := map[string]interface{}{
		"name":  "pipekit",
		"port":  float64(8080),
		"tags":  []interface{}{"a", "b"},
		"flags": map[string]interface{}{"x": true},
	}
	for _, format := range []DataFormat{FormatJSON, FormatYAML} {
		data, err := Encode(v, format, false)
		if err != nil {
			t.Fatalf("encode %s: %v", format, err)
		}
		back, err := Decode(data, format)
		if err != nil {
			t.Fatalf("decode %s: %v", format, err)
		}
		// Encode again; expect stable output for the same input.
		again, err := Encode(back, format, false)
		if err != nil {
			t.Fatalf("re-encode %s: %v", format, err)
		}
		if string(data) != string(again) {
			t.Errorf("round-trip not stable for %s\nfirst:\n%s\nsecond:\n%s",
				format, data, again)
		}
	}
}

func TestEncode_TOML(t *testing.T) {
	v := map[string]interface{}{
		"package": map[string]interface{}{"name": "myapp", "version": "1.0.0"},
	}
	data, err := Encode(v, FormatTOML, false)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	out := string(data)
	if !strings.Contains(out, `name = "myapp"`) && !strings.Contains(out, `name = 'myapp'`) {
		t.Errorf("expected toml content, got: %s", out)
	}
	back, err := Decode(data, FormatTOML)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if back.(map[string]interface{})["package"].(map[string]interface{})["name"] != "myapp" {
		t.Error("round-trip lost data")
	}
}

func TestRenderTable(t *testing.T) {
	records := []map[string]interface{}{
		{"name": "api", "port": float64(8080)},
		{"name": "web", "port": float64(3000)},
	}
	out := RenderTable(records, []string{"name", "port"})
	if !strings.Contains(out, "name") || !strings.Contains(out, "port") {
		t.Errorf("missing headers: %s", out)
	}
	if !strings.Contains(out, "api") || !strings.Contains(out, "8080") {
		t.Errorf("missing data: %s", out)
	}
	// Has a separator row.
	if !strings.Contains(out, "----") {
		t.Errorf("missing separator: %s", out)
	}
}

func TestPrettyJSON(t *testing.T) {
	out, err := PrettyJSON([]byte(`{"a":1,"b":[2,3]}`), 2)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	want := `{
  "a": 1,
  "b": [
    2,
    3
  ]
}`
	if string(out) != want {
		t.Errorf("got\n%s\nwant\n%s", out, want)
	}
}

func TestParsePath(t *testing.T) {
	tests := []struct {
		path string
		want []string
		err  bool
	}{
		{".image.tag", []string{"image", "tag"}, false},
		{".items[0].name", []string{"items", "0", "name"}, false},
		{"image.tag", []string{"image", "tag"}, false},
		{"", nil, true},
		{".", nil, true},
	}
	for _, tc := range tests {
		got, err := parsePath(tc.path)
		if (err != nil) != tc.err {
			t.Errorf("parsePath(%q) err=%v, want err=%v", tc.path, err, tc.err)
			continue
		}
		if !tc.err && !reflect.DeepEqual(got, tc.want) {
			t.Errorf("parsePath(%q) = %v, want %v", tc.path, got, tc.want)
		}
	}
}

func TestCSVEncodeDecode(t *testing.T) {
	v := []interface{}{
		map[string]interface{}{"name": "api", "port": float64(8080)},
		map[string]interface{}{"name": "web", "port": float64(3000)},
	}
	data, err := Encode(v, FormatCSV, false)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if !strings.Contains(string(data), "name,port") {
		t.Errorf("expected header row: %s", data)
	}
	back, err := Decode(data, FormatCSV)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	arr := back.([]interface{})
	if len(arr) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(arr))
	}
	if arr[0].(map[string]interface{})["name"] != "api" {
		t.Errorf("decode mismatch: %v", arr[0])
	}
}
