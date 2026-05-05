package services

import (
	"bytes"
	"io"
	"regexp"
	"sync"
)

// maskingWriter wraps an underlying io.Writer and applies regex masking on
// each line written to it. Partial lines are buffered until a newline arrives.
type maskingWriter struct {
	mu       sync.Mutex
	w        io.Writer
	patterns []*regexp.Regexp
	repl     string
	buf      bytes.Buffer
}

func newMaskingWriter(w io.Writer, patterns []*regexp.Regexp, repl string) *maskingWriter {
	return &maskingWriter{w: w, patterns: patterns, repl: repl}
}

func (m *maskingWriter) Write(p []byte) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.buf.Write(p)
	if err := m.flushLines(false); err != nil {
		return 0, err
	}
	return len(p), nil
}

func (m *maskingWriter) Flush() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.flushLines(true)
}

func (m *maskingWriter) flushLines(includePartial bool) error {
	for {
		line, err := m.buf.ReadString('\n')
		if err == io.EOF || (err != nil && err.Error() == "EOF") {
			// No newline yet; put it back and stop unless flushing.
			if line != "" && includePartial {
				if err := m.writeMasked(line); err != nil {
					return err
				}
			} else if line != "" {
				m.buf.WriteString(line)
			}
			return nil
		}
		if err != nil {
			return err
		}
		if err := m.writeMasked(line); err != nil {
			return err
		}
	}
}

func (m *maskingWriter) writeMasked(s string) error {
	for _, re := range m.patterns {
		s = re.ReplaceAllString(s, m.repl)
	}
	_, err := io.WriteString(m.w, s)
	return err
}

// circularBuffer keeps the last N bytes written to it, useful for grabbing
// recent stderr without unbounded growth.
type circularBuffer struct {
	mu   sync.Mutex
	data []byte
	cap  int
}

func newCircularBuffer(capacity int) *circularBuffer {
	return &circularBuffer{cap: capacity}
}

func (c *circularBuffer) Write(p []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data = append(c.data, p...)
	if len(c.data) > c.cap {
		c.data = c.data[len(c.data)-c.cap:]
	}
	return len(p), nil
}

func (c *circularBuffer) Bytes() []byte {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]byte, len(c.data))
	copy(out, c.data)
	return out
}
