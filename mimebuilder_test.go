//go:build unit
// +build unit

package mimebuilder

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/bytebufferpool"
)

func TestNew(t *testing.T) {
	m := New()

	require.NotNil(t, m, "New() should never return nil")
	assert.Empty(t, m.from, "expected from to start empty")
	assert.Empty(t, m.to, "expected to to start empty")
	assert.Empty(t, m.cc, "expected cc to start empty")
	assert.Empty(t, m.bcc, "expected bcc to start empty")
	assert.Empty(t, m.replyTo, "expected replyTo to start empty")
	assert.Empty(t, m.subject, "expected subject to start empty")
	assert.Empty(t, m.body, "expected body to start empty")
	assert.Empty(t, m.altBody, "expected altBody to start empty")
	assert.False(t, m.isHTML, "expected isHTML to default to false")
	assert.Empty(t, m.attachments, "expected attachments to start empty")
	assert.Empty(t, m.inlineImages, "expected inlineImages to start empty")

	// Preallocation checks (capacity)
		assert.Equal(t, 64, cap(m.from), "expected from preallocated to 64 bytes")
		assert.Equal(t, 4096, cap(m.body), "expected body preallocated to 4096 bytes")
		assert.Equal(t, 4096, cap(m.altBody), "expected altBody preallocated to 4096 bytes")
		assert.Equal(t, 4, cap(m.attachments), "expected attachments preallocated to 4 slots")
		assert.Equal(t, 4, cap(m.inlineImages), "expected inlineImages preallocated to 4 slots")
}

func TestStr2Bytes(t *testing.T) {
	tests := []struct {
		name string
		in   string
	}{
		{"normal ascii", "hello world"},
		{"empty string", ""},
		{"unicode", "héllo wörld 日本語"},
		{"with spaces and symbols", "a b c !@#$%^&*()"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := str2bytes(tt.in)
			assert.Equal(t, tt.in, string(b), "expected round-trip conversion to preserve content")
			assert.Equal(t, len(tt.in), len(b), "expected byte length to match string length")
		})
	}
}

func TestContainsCRLF(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{"no CRLF", "hello world", false},
		{"empty string", "", false},
		{"contains CR", "hello\rworld", true},
		{"contains LF", "hello\nworld", true},
		{"contains both CRLF", "hello\r\nworld", true},
		{"CRLF at start", "\r\nhello", true},
		{"CRLF at end", "hello\r\n", true},
		{"only CR", "\r", true},
		{"only LF", "\n", true},
		{"unicode with no CRLF", "héllo wörld", false},
		{"tab and space only, no CRLF", "hello\tworld ok", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := containsCRLF(tt.in)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestAppendSanitized(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"clean string, fast path", "hello world", "hello world"},
		{"empty string", "", ""},
		{"strips CR", "hello\rworld", "helloworld"},
		{"strips LF", "hello\nworld", "helloworld"},
		{"strips CRLF", "hello\r\nworld", "helloworld"},
		{"strips multiple CRLF occurrences", "a\r\nb\r\nc", "abc"},
		{"strips leading CRLF", "\r\nhello", "hello"},
		{"strips trailing CRLF", "hello\r\n", "hello"},
		{"header injection attempt", "Legit Name\r\nBcc: attacker@evil.com", "Legit NameBcc: attacker@evil.com"},
		{"unicode preserved, no CRLF", "héllo wörld 日本語", "héllo wörld 日本語"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := appendSanitized(make([]byte, 0, len(tt.in)), tt.in)
			assert.Equal(t, tt.want, string(buf))
		})
	}
}

func TestAppendSanitized_AppendsToExistingBuffer(t *testing.T) {
	buf := []byte("prefix: ")
	buf = appendSanitized(buf, "clean value")

	assert.Equal(t, "prefix: clean value", string(buf))
}

func TestAppendSanitized_AppendsToExistingBufferWithInjection(t *testing.T) {
	buf := []byte("prefix: ")
	buf = appendSanitized(buf, "value\r\nX-Injected: true")

	assert.Equal(t, "prefix: valueX-Injected: true", string(buf))
}

func TestQpEncode(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"simple ascii", "Hello World", "Hello World"},
		{"empty input", "", ""},
		{"equals sign encoded", "100% = success", "100% =3D success"},
		{"non-ascii byte encoded", "café", "caf=C3=A9"},
		{"tab preserved mid-line", "a\tb", "a\tb"},
		{"newline gets hex-encoded", "line1\nline2", "line1=0Aline2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := bytebufferpool.Get()
			defer bytebufferpool.Put(buf)

			qpEncode(buf, []byte(tt.in))

			assert.Equal(t, tt.want, string(buf.B))
		})
	}
}

func TestQpEncode_TrailingSpaceEncoded(t *testing.T) {
	// RFC 2045: a space immediately before a line break MUST be encoded,
	// otherwise mail transport can silently strip it.
	buf := bytebufferpool.Get()
	defer bytebufferpool.Put(buf)

	qpEncode(buf, []byte("hello \nworld"))

	got := string(buf.B)
	assert.Contains(t, got, "hello=20", "expected trailing space before newline to be encoded as =20")
}

func TestQpEncode_TrailingSpaceAtEndOfData(t *testing.T) {
	buf := bytebufferpool.Get()
	defer bytebufferpool.Put(buf)

	qpEncode(buf, []byte("hello "))

	got := string(buf.B)
	assert.Equal(t, "hello=20", got, "expected trailing space at end of data to be encoded")
}

func TestQpEncode_SoftLineBreakAt72Chars(t *testing.T) {
	// A long line of safe ASCII characters should get a soft line break
	// ("=\r\n") inserted once lineLen reaches 72.
	longLine := strings.Repeat("a", 100)

	buf := bytebufferpool.Get()
	defer bytebufferpool.Put(buf)

	qpEncode(buf, []byte(longLine))

	got := string(buf.B)
	assert.Contains(t, got, "=\r\n", "expected a soft line break to be inserted for lines exceeding 72 chars")
}

func TestQEncodeSubject(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"simple ascii", "Hello World", "\r\nSubject: =?UTF-8?Q?Hello_World?="},
		{"empty subject", "", "\r\nSubject: =?UTF-8?Q??="},
		{"non-ascii", "café", "\r\nSubject: =?UTF-8?Q?caf=C3=A9?="},
		{"alphanumeric only", "Test123", "\r\nSubject: =?UTF-8?Q?Test123?="},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := bytebufferpool.Get()
			defer bytebufferpool.Put(buf)

			qEncodeSubject(buf, []byte(tt.in))

			assert.Equal(t, tt.want, string(buf.B))
		})
	}
}

func TestQEncodeSubject_SpacesBecomeUnderscore(t *testing.T) {
	buf := bytebufferpool.Get()
	defer bytebufferpool.Put(buf)

	qEncodeSubject(buf, []byte("a b c"))

	got := string(buf.B)
	assert.Equal(t, "\r\nSubject: =?UTF-8?Q?a_b_c?=", got)
}

func TestQEncodeSubject_LongSubjectFolds(t *testing.T) {
	// A long subject should fold into multiple encoded-words separated by
	// "?=\r\n =?UTF-8?Q?" once the line length threshold is hit.
	longSubject := strings.Repeat("word ", 30) // 150 chars

	buf := bytebufferpool.Get()
	defer bytebufferpool.Put(buf)

	qEncodeSubject(buf, []byte(longSubject))

	got := string(buf.B)
	assert.Contains(t, got, "?=\r\n =?UTF-8?Q?", "expected long subject to fold onto a continuation line")
}

func TestQEncodeSubject_DoesNotSplitMultiByteUTF8Char(t *testing.T) {
	// Regression check: a multi-byte UTF-8 character (e.g. an emoji, 4 bytes)
	// must never be split across two encoded-word segments.
	subject := strings.Repeat("a", 60) + "😀" + strings.Repeat("b", 60)

	buf := bytebufferpool.Get()
	defer bytebufferpool.Put(buf)

	qEncodeSubject(buf, []byte(subject))

	got := string(buf.B)
	// The emoji's hex-encoded bytes should appear as one contiguous block,
	// never interrupted by a fold sequence in the middle.
	emojiHex := "=F0=9F=98=80" // UTF-8 encoding of 😀
	assert.Contains(t, got, emojiHex, "expected emoji to appear fully encoded and uninterrupted")
}