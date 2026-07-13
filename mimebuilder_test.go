//go:build unit
// +build unit

package mimebuilder

import (
	"encoding/base64"
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

func TestGetMimeType(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     string
	}{
		{"pdf", "report.pdf", "application/pdf"},
		{"docx", "file.docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document"},
		{"xlsx", "sheet.xlsx", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"},
		{"png", "logo.png", "image/png"},
		{"jpg", "photo.jpg", "image/jpeg"},
		{"jpeg", "photo.jpeg", "image/jpeg"},
		{"svg", "icon.svg", "image/svg+xml"},
		{"txt", "notes.txt", "text/plain"},
		{"csv", "data.csv", "text/csv"},
		{"zip", "archive.zip", "application/zip"},
		{"mp4", "video.mp4", "video/mp4"},
		{"unknown extension", "file.xyz", "application/octet-stream"},
		{"no extension", "filenoext", "application/octet-stream"},
		{"trailing dot", "file.", "application/octet-stream"},
		{"dot in directory but no ext", "/path.dir/file", "application/octet-stream"},
		{"empty filename", "", "application/octet-stream"},
		{"uppercase extension not matched", "FILE.PDF", "application/octet-stream"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getMimeType([]byte(tt.filename))
			assert.Equal(t, tt.want, string(got))
		})
	}
}

func TestEncodeBase64(t *testing.T) {
	tests := []struct {
		name string
		in   []byte
	}{
		{"empty data", []byte{}},
		{"small data", []byte("hello")},
		{"exactly one chunk (57 bytes)", []byte(strings.Repeat("a", 57))},
		{"multiple chunks (150 bytes)", []byte(strings.Repeat("b", 150))},
		{"binary data", []byte{0x00, 0xFF, 0x10, 0x20, 0xAB, 0xCD}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := bytebufferpool.Get()
			defer bytebufferpool.Put(buf)

			encodeBase64(buf, tt.in)

			got := string(buf.B)

			if len(tt.in) == 0 {
				assert.Empty(t, got, "expected no output for empty input")
				return
			}

			// Strip the CRLF line breaks encodeBase64 inserts every 76 chars,
			// then confirm the decoded result matches the original input.
			cleaned := strings.ReplaceAll(got, "\r\n", "")
			decoded, err := base64.StdEncoding.DecodeString(cleaned)
			assert.NoError(t, err, "expected valid base64 output")
			assert.Equal(t, tt.in, decoded, "expected decoded output to match original input")
		})
	}
}

func TestEncodeBase64_LineWrapping(t *testing.T) {
	// MIME base64 requires a line break every 76 encoded characters.
	// 57 input bytes -> 76 encoded chars is the standard chunk size used here.
	data := []byte(strings.Repeat("x", 57*2)) // two full chunks

	buf := bytebufferpool.Get()
	defer bytebufferpool.Put(buf)

	encodeBase64(buf, data)

	got := string(buf.B)
	lines := strings.Split(strings.TrimRight(got, "\r\n"), "\r\n")

	for i, line := range lines {
		if i == len(lines)-1 {
			continue // last line may be shorter
		}
		assert.LessOrEqual(t, len(line), 76, "expected each base64 line to be at most 76 characters")
	}
}

func TestEncodeBase64_AppendsToExistingBuffer(t *testing.T) {
	buf := bytebufferpool.Get()
	defer bytebufferpool.Put(buf)

	buf.Write([]byte("prefix:"))
	encodeBase64(buf, []byte("data"))

	got := string(buf.B)
	assert.True(t, strings.HasPrefix(got, "prefix:"), "expected encodeBase64 to append, not overwrite existing buffer content")
}

func TestSetFrom(t *testing.T) {
	tests := []struct {
		name  string
		email string
		label string
		want  string
	}{
		{"with display name", "john@example.com", "John Doe", "John Doe <john@example.com>"},
		{"email only, no name", "john@example.com", "", "john@example.com"},
		{"name with injection attempt", "john@example.com", "John\r\nBcc: evil@x.com", "JohnBcc: evil@x.com <john@example.com>"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New()
			result := m.SetFrom(tt.email, tt.label)

			assert.Same(t, m, result, "SetFrom should return the same *MimeBuilder for chaining")
			assert.Equal(t, tt.want, string(m.from))
		})
	}
}

func TestSetFrom_ResetsOnSecondCall(t *testing.T) {
	m := New()
	m.SetFrom("first@example.com", "First")
	m.SetFrom("second@example.com", "Second")

	assert.Equal(t, "Second <second@example.com>", string(m.from), "expected SetFrom to overwrite, not append")
}

func TestAddTo(t *testing.T) {
	tests := []struct {
		name  string
		email string
		label string
		want  string
	}{
		{"with display name", "jane@example.com", "Jane Doe", "Jane Doe <jane@example.com>"},
		{"email only, no name", "jane@example.com", "", "jane@example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New()
			result := m.AddTo(tt.email, tt.label)

			assert.Same(t, m, result, "AddTo should return the same *MimeBuilder for chaining")
			assert.Equal(t, tt.want, string(m.to))
		})
	}
}

func TestAddTo_MultipleAppendsWithComma(t *testing.T) {
	m := New()
	m.AddTo("first@example.com", "First").
		AddTo("second@example.com", "Second").
		AddTo("third@example.com", "")

	want := "First <first@example.com>, Second <second@example.com>, third@example.com"
	assert.Equal(t, want, string(m.to))
}

func TestAddTo_DoesNotResetBetweenCalls(t *testing.T) {
	m := New()
	m.AddTo("a@example.com", "")
	m.AddTo("b@example.com", "")

	assert.Contains(t, string(m.to), "a@example.com", "expected first AddTo call to persist")
	assert.Contains(t, string(m.to), "b@example.com", "expected second AddTo call to be appended, not overwrite")
}

func TestAddCC(t *testing.T) {
	tests := []struct {
		name  string
		email string
		label string
		want  string
	}{
		{"with display name", "cc@example.com", "CC Person", "CC Person <cc@example.com>"},
		{"email only, no name", "cc@example.com", "", "cc@example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New()
			result := m.AddCC(tt.email, tt.label)

			assert.Same(t, m, result, "AddCC should return the same *MimeBuilder for chaining")
			assert.Equal(t, tt.want, string(m.cc))
		})
	}
}

func TestAddCC_MultipleAppendsWithComma(t *testing.T) {
	m := New()
	m.AddCC("cc1@example.com", "CC One").
		AddCC("cc2@example.com", "")

	want := "CC One <cc1@example.com>, cc2@example.com"
	assert.Equal(t, want, string(m.cc))
}

func TestAddBCC(t *testing.T) {
	tests := []struct {
		name  string
		email string
		label string
		want  string
	}{
		{"with display name", "bcc@example.com", "BCC Person", "BCC Person <bcc@example.com>"},
		{"email only, no name", "bcc@example.com", "", "bcc@example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New()
			result := m.AddBCC(tt.email, tt.label)

			assert.Same(t, m, result, "AddBCC should return the same *MimeBuilder for chaining")
			assert.Equal(t, tt.want, string(m.bcc))
		})
	}
}

func TestAddBCC_MultipleAppendsWithComma(t *testing.T) {
	m := New()
	m.AddBCC("bcc1@example.com", "BCC One").
		AddBCC("bcc2@example.com", "")

	want := "BCC One <bcc1@example.com>, bcc2@example.com"
	assert.Equal(t, want, string(m.bcc))
}

func TestAddReplyTo(t *testing.T) {
	tests := []struct {
		name  string
		email string
		label string
		want  string
	}{
		{"with display name", "reply@example.com", "Reply Person", "Reply Person <reply@example.com>"},
		{"email only, no name", "reply@example.com", "", "reply@example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New()
			result := m.AddReplyTo(tt.email, tt.label)

			assert.Same(t, m, result, "AddReplyTo should return the same *MimeBuilder for chaining")
			assert.Equal(t, tt.want, string(m.replyTo))
		})
	}
}

func TestAddReplyTo_MultipleAppendsWithComma(t *testing.T) {
	m := New()
	m.AddReplyTo("reply1@example.com", "Reply One").
		AddReplyTo("reply2@example.com", "")

	want := "Reply One <reply1@example.com>, reply2@example.com"
	assert.Equal(t, want, string(m.replyTo))
}

func TestSetSubject(t *testing.T) {
	tests := []struct {
		name    string
		subject string
		want    string
	}{
		{"normal subject", "Hello World", "Hello World"},
		{"empty subject", "", ""},
		{"subject with injection attempt", "Subject\r\nBcc: evil@x.com", "SubjectBcc: evil@x.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New()
			result := m.SetSubject(tt.subject)

			assert.Same(t, m, result, "SetSubject should return the same *MimeBuilder for chaining")
			assert.Equal(t, tt.want, string(m.subject))
		})
	}
}

func TestSetSubject_ResetsOnSecondCall(t *testing.T) {
	m := New()
	m.SetSubject("First Subject")
	m.SetSubject("Second Subject")

	assert.Equal(t, "Second Subject", string(m.subject), "expected SetSubject to overwrite, not append")
}

func TestSetBody(t *testing.T) {
	m := New()
	result := m.SetBody("Hello Body")

	assert.Same(t, m, result, "SetBody should return the same *MimeBuilder for chaining")
	assert.Equal(t, "Hello Body", string(m.body))
}

func TestSetBody_OverwritesOnSecondCall(t *testing.T) {
	m := New()
	m.SetBody("First")
	m.SetBody("Second")

	assert.Equal(t, "Second", string(m.body))
}