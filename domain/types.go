package domain

// KeyValue represents a key-value pair for env var operations.
type KeyValue struct {
	Key   string
	Value string
}

// NotifyMessage represents a structured notification message.
type NotifyMessage struct {
	Status  string            // success, failure, warning
	Title   string
	Message string
	Fields  map[string]string
	URL     string
}

// DiffConfig maps path prefixes to service names for monorepo detection.
type DiffConfig struct {
	Services map[string][]string `yaml:"services"` // service_name -> [path_prefixes...]
}

// WaitResult represents the result of a wait/poll operation.
type WaitResult struct {
	Success    bool
	Attempts   int
	StatusCode int
	Body       string
}
