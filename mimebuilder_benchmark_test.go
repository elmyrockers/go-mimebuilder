package mimebuilder

import (
    "testing"
    "time"
)

func BenchmarkMimeBuilder(b *testing.B) {
    builder := New()
    b.ReportAllocs()
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        buf, _ := builder.
            SetFrom("Your Name", "you@yourcompany.com").
            AddTo("Helmi Aziz", "helmi@xeno.com.my").
            SetSubject("Benchmark Test").
            SetBody("This is email body for our benchmark test").
            Build()
        if len(buf.B) == 0 {
            b.Fatalf("empty buffer")
        }
        builder.Release(buf)
    }
}

func TestStressMillion(t *testing.T) {
    builder := New()
    start := time.Now()
    for i := 0; i < 1_000_000; i++ {
        buf, _ := builder.
            SetFrom("Your Name", "you@yourcompany.com").
            AddTo("Helmi Aziz", "helmi@xeno.com.my").
            SetSubject("Stress Test").
            SetBody("This is a 1M iteration stress test.").
            AsHTML().
            Build()
        if len(buf.B) == 0 {
            t.Fatalf("empty buffer")
        }
        builder.Release(buf)
    }
    elapsed := time.Since(start)
    t.Logf("Processed 1,000,000 requests in %s (%.2f req/sec)",
        elapsed, float64(1_000_000)/elapsed.Seconds())
}
