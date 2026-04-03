# go-mimebuilder

<div align="center">
	<img src="email.jpg" width="400" />
</div>

**A High-Performance, Zero-Allocation Go library for generating raw MIME messages.** Designed for high-concurrency systems and low-memory environments, it produces standards-compliant output ready for any SMTP client, mail server, or cloud API.

## Why go-mimebuilder?
- **Zero-Allocation Architecture:** Uses `bytebufferpool` to recycle memory, drastically reducing GC overhead on low-RAM (1GB) VPS instances.
- **High-Speed String Handling:** Implements `unsafe` pointer arithmetic for zero-copy string-to-byte conversions, ensuring lightning-fast header construction.
- **Preallocated Buffers:** Slices are pre-sized to `4KB` (OS page size) to eliminate memory fragmentation and "realloc" lag.
- **Smart MIME Nesting:** Automatically manages complex `mixed`, `alternative`, and `related` structures based on your content.
- **RFC 5322 Compliant:** Strictly enforces `\r\n` (CRLF) endings and proper headers to ensure 100% deliverability.
- **Fluent API:** Clean, chainable method syntax for building complex emails in a single, readable block.
- **Inline Image Support (CID):** Full support for embedding images directly into HTML bodies using unique Content-IDs.
- **Dual-Mode Attachments:** Flexible support for attaching raw `[]byte` or streaming via `io.Reader` for large file handling.

## Quickstart

Install the library:
```bash
go get github.com/elmyrockers/go-mimebuilder
```

Basic Example:
```go
package main

import (
	"github.com/elmyrockers/go-mimebuilder"
    "fmt"
    "os"
)

func main() {
    // 1. Initialize and chain your email data
	    m := mimebuilder.New().
	        SetFrom("Sender Name", "sender@example.com").
	        AddTo("Recipient Name", "recipient@example.com").
	        SetSubject("High Performance MIME").
	        SetBody("<h1>Hello!</h1><p>Sent via go-mimebuilder.</p>").AsHTML().
	        SetAltBody("Hello! Sent via go-mimebuilder.").
	        Attach("document.pdf", []byte("%PDF-1.4..."))

    // 2. Build the raw MIME byte slice
    // Uses bytebufferpool for 0 B/op performance
		    raw, err := m.Build()
		    if err != nil {
		        panic(err)
		    }

    // 3. Send via SMTP or save to an .eml file
	    fmt.Printf("Generated %d bytes of MIME data\n", len(raw))
	    os.WriteFile("message.eml", raw, 0644)
}
```