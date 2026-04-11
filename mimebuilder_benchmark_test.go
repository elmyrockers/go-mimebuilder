package mimebuilder

import "testing"

func BenchmarkMimeBuilder(b *testing.B) {
    builder := New()
    b.ReportAllocs()
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        buf, _ := builder.
            SetFrom("Alice", "alice@example.com").
            AddTo("Bob", "bob@example.com").
            SetSubject("Hello World").
            SetBody("Recruiter‑ready benchmark email").
            AsHTML().
            Build()
        if len(buf.B) == 0 {
            b.Fatalf("empty buffer")
        }
        builder.Release(buf)
    }
}
