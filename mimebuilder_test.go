package mimebuilder

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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