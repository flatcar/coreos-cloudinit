package config

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"testing"
)

func gzipString(t *testing.T, content string) string {
	t.Helper()

	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	if _, err := w.Write([]byte(content)); err != nil {
		t.Fatalf("compressing test content: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("closing gzip writer: %v", err)
	}

	return buf.String()
}

func TestDecodeContent(t *testing.T) {
	content := "#!/bin/bash\necho hello\n"
	gzipped := gzipString(t, content)

	tests := []struct {
		encoding string
		content  string

		decoded string
		isValid bool
	}{
		{encoding: "", content: content, decoded: content, isValid: true},
		{encoding: "base64", content: base64.StdEncoding.EncodeToString([]byte(content)), decoded: content, isValid: true},
		{encoding: "b64", content: base64.StdEncoding.EncodeToString([]byte(content)), decoded: content, isValid: true},
		{encoding: "gzip", content: gzipped, decoded: content, isValid: true},
		{encoding: "gz", content: gzipped, decoded: content, isValid: true},
		{encoding: "gz+base64", content: base64.StdEncoding.EncodeToString([]byte(gzipped)), decoded: content, isValid: true},
		{encoding: "gzip+b64", content: base64.StdEncoding.EncodeToString([]byte(gzipped)), decoded: content, isValid: true},
		{encoding: "base64", content: "not base64", isValid: false},
		{encoding: "gzip", content: "not gzip", isValid: false},
		{encoding: "unknown", content: content, isValid: false},
	}

	for _, tt := range tests {
		decoded, err := DecodeContent(tt.content, tt.encoding)
		if tt.isValid != (err == nil) {
			t.Errorf("bad validity (%q): want %t, got %q", tt.encoding, tt.isValid, err)
			continue
		}
		if tt.isValid && string(decoded) != tt.decoded {
			t.Errorf("bad content (%q): want %q, got %q", tt.encoding, tt.decoded, string(decoded))
		}
	}
}

// A gzip stream that is cut short still has a valid header, so the failure only
// shows up while reading the body. Decoding it must not return partial content.
func TestDecodeGzipContentIncomplete(t *testing.T) {
	content := bytes.Repeat([]byte("some config content\n"), 200)
	gzipped := gzipString(t, string(content))

	truncated := gzipped[:len(gzipped)/2]

	corrupted := []byte(gzipped)
	corrupted[len(corrupted)-1] ^= 0xff

	tests := []struct {
		name    string
		content string
	}{
		{name: "truncated stream", content: truncated},
		{name: "corrupted checksum", content: string(corrupted)},
	}

	for _, tt := range tests {
		decoded, err := DecodeGzipContent(tt.content)
		if err == nil {
			t.Errorf("%s: expected an error, got %d bytes of content", tt.name, len(decoded))
		}
		if len(decoded) != 0 {
			t.Errorf("%s: expected no content on error, got %d bytes", tt.name, len(decoded))
		}
	}
}
