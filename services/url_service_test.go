package services

import "testing"

func TestParseURL(t *testing.T) {
	kvs, err := ParseURL("postgres://app:secret@db.internal:5432/prod?sslmode=require", "DB_")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	got := map[string]string{}
	for _, kv := range kvs {
		got[kv.Key] = kv.Value
	}
	want := map[string]string{
		"DB_SCHEME":   "postgres",
		"DB_HOST":     "db.internal",
		"DB_PORT":     "5432",
		"DB_USER":     "app",
		"DB_PASSWORD": "secret",
		"DB_PATH":     "/prod",
		"DB_QUERY":    "sslmode=require",
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %q, want %q", k, got[k], v)
		}
	}
}

func TestParseURL_NoCredentials(t *testing.T) {
	kvs, err := ParseURL("https://example.com/api", "")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	for _, kv := range kvs {
		if kv.Key == "USER" || kv.Key == "PASSWORD" {
			t.Errorf("unexpected credential key %s=%s", kv.Key, kv.Value)
		}
	}
}

func TestParseURL_EmptyError(t *testing.T) {
	if _, err := ParseURL("not a url", ""); err == nil {
		t.Error("expected error for malformed URL without scheme")
	}
}
