package mimebuilder

import "testing"

func BenchmarkMimeBuilder(b *testing.B) {
    builder := New()
    b.ReportAllocs()
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        buf, _ := builder.
            SetFrom("Your Name", "you@yourcompany.com").
            AddTo("Helmi Aziz", "Helmi@xeno.com.my").
            SetSubject("Benchmark Test").
            SetBody("This is email body for our benchmark test").
            Build()
        if len(buf.B) == 0 {
            b.Fatalf("empty buffer")
        }
        builder.Release(buf)
    }
}
