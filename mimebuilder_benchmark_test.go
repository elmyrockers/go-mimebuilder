//go:build benchmark
// +build benchmark

package mimebuilder

import (
    "testing"
    "time"
)

// BenchmarkMimeBuilder_PlainText exercises the minimal hot path:
// From, To, Subject, Body (plain text), Build, Release.
func BenchmarkMimeBuilder_PlainText(b *testing.B) {
    builder := New()
    b.ReportAllocs()
    b.ResetTimer()

    for i := 0; i < b.N; i++ {
        buf, _ := builder.
            SetFrom("noreply@xeno.com.my", "Xeno System").
            AddTo("helmi@xeno.com.my", "Helmi Aziz").
            SetSubject("Benchmark Test").
            SetBody("This is email body for our benchmark test").
            Build()

        if len(buf.B) == 0 {
            b.Fatalf("empty buffer")
        }
        builder.Release(buf)
    }
}

// BenchmarkMimeBuilder_HTML exercises the HTML-only hot path:
// From, To, Subject, Body (HTML via AsHTML), Build, Release.
func BenchmarkMimeBuilder_HTML(b *testing.B) {
    builder := New()
    b.ReportAllocs()
    b.ResetTimer()

    for i := 0; i < b.N; i++ {
        buf, _ := builder.
            SetFrom("noreply@xeno.com.my", "Xeno System").
            AddTo("helmi@xeno.com.my", "Helmi Aziz").
            SetSubject("Benchmark Test").
            SetBody("<h1>Hello!</h1><p>This is an HTML benchmark test</p>").AsHTML().
            Build()

        if len(buf.B) == 0 {
            b.Fatalf("empty buffer")
        }
        builder.Release(buf)
    }
}

// BenchmarkMimeBuilder_Alternative exercises the multipart/alternative
// hot path: From, To, Subject, HTML Body, AltBody, Build, Release.
func BenchmarkMimeBuilder_Alternative(b *testing.B) {
    builder := New()
    b.ReportAllocs()
    b.ResetTimer()

    for i := 0; i < b.N; i++ {
        buf, _ := builder.
            SetFrom("noreply@xeno.com.my", "Xeno System").
            AddTo("helmi@xeno.com.my", "Helmi Aziz").
            SetSubject("Benchmark Test").
            SetBody("<h1>Hello!</h1><p>HTML version</p>").AsHTML().
            SetAltBody("Hello! Plain text version").
            Build()

        if len(buf.B) == 0 {
            b.Fatalf("empty buffer")
        }
        builder.Release(buf)
    }
}

// BenchmarkMimeBuilder_FullHeaders exercises every hot-path header
// method in a single call: From, To, CC, BCC, ReplyTo, Subject, Body.
func BenchmarkMimeBuilder_FullHeaders(b *testing.B) {
    builder := New()
    b.ReportAllocs()
    b.ResetTimer()

    for i := 0; i < b.N; i++ {
        buf, _ := builder.
            SetFrom("noreply@xeno.com.my", "Xeno System").
            AddTo("helmi@xeno.com.my", "Helmi Aziz").
            AddCC("cc@xeno.com.my", "CC Person").
            AddBCC("bcc@xeno.com.my", "BCC Person").
            AddReplyTo("info@xeno.com.my", "Xeno Admin").
            SetSubject("Benchmark Test - Full Headers").
            SetBody("This is email body for our full-header benchmark test").
            Build()

        if len(buf.B) == 0 {
            b.Fatalf("empty buffer")
        }
        builder.Release(buf)
    }
}

// BenchmarkMimeBuilder_MultipleRecipients exercises repeated appends to
// To/CC/BCC/ReplyTo, which is the realistic "multiple recipients" case.
func BenchmarkMimeBuilder_MultipleRecipients(b *testing.B) {
    builder := New()
    b.ReportAllocs()
    b.ResetTimer()

    for i := 0; i < b.N; i++ {
        buf, _ := builder.
            SetFrom("noreply@xeno.com.my", "Xeno System").
            AddTo("a@xeno.com.my", "Ali").
            AddTo("b@xeno.com.my", "Ahmad").
            AddTo("c@xeno.com.my", "Razman").
            AddCC("cc1@xeno.com.my", "CC One").
            AddCC("cc2@xeno.com.my", "CC Two").
            SetSubject("Benchmark Test - Multiple Recipients").
            SetBody("This is email body for our multi-recipient benchmark test").
            Build()

        if len(buf.B) == 0 {
            b.Fatalf("empty buffer")
        }
        builder.Release(buf)
    }
}

// TestStressMillion runs 1,000,000 iterations of the HTML hot path to
// measure sustained throughput under load.
func TestStressMillion(t *testing.T) {
    builder := New()
    start := time.Now()

    for i := 0; i < 1_000_000; i++ {
        buf, _ := builder.
            SetFrom("noreply@xeno.com.my", "Xeno System").
            AddTo("helmi@xeno.com.my", "Helmi Aziz").
            SetSubject("Stress Test").
            SetBody("This is a 1M iteration stress test.").AsHTML().
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