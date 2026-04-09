package mimebuilder



import (
	"io"
	"unsafe"
	"fmt"
	"time"
	"os"
	"crypto/rand"
	// "encoding/hex"
	// "mime/quotedprintable"
	"encoding/binary"
	"encoding/base64"

	"github.com/valyala/bytebufferpool"
)

type Attachment struct {
	Filename 	[]byte
	Data 		[]byte
	Stream 		io.Reader
}

type InlineImage struct {
	Filename 	string
	Data 		[]byte
	ContentID 	string
}
type MimeBuilder struct {
	mixedBoundary 	[32]byte
	altBoundary 	[32]byte
	relBoundary 	[32]byte

	from 		[]byte
	to 			[]byte
	cc 			[]byte
	bcc 		[]byte
	replyTo 	[]byte

	subject 	[]byte
	body 		[]byte
	altBody 	[]byte
	isHTML 		bool

	attachments 	[]Attachment
	inlineImages 	[]InlineImage

	errorList 	[]error
}

func New() *MimeBuilder {
	return &MimeBuilder{
		// Headers: Usually short, < 128 bytes
			from:    make([]byte, 0, 64),
			to:      make([]byte, 0, 128),
			cc:      make([]byte, 0, 128),
			bcc:     make([]byte, 0, 128),
			replyTo: make([]byte, 0, 64),
			subject: make([]byte, 0, 128),

		// Content: 4KB is a standard "Page Size" in most OSs
		// Great for text/html bodies
			body:    make([]byte, 0, 4096),
			altBody: make([]byte, 0, 4096),
			isHTML:  false,

		// Slices of Structs: Preallocate space for 4 attachments/images
		// to avoid resizing the slice header itself
			attachments:  make([]Attachment, 0, 4),
			inlineImages: make([]InlineImage, 0, 4),

		// Errors: Hopefully 0, but 2 slots covers typical validation hits
			errorList: make([]error, 0, 2),
	}
}

// str2bytes() converts string to slice of byte without a memory allocation.
func str2bytes(s string) []byte {
	return unsafe.Slice(unsafe.StringData(s), len(s))
}

func qpEncode(buf *bytebufferpool.ByteBuffer, data []byte) {
	const hexTable = "0123456789ABCDEF"
	lineLen := 0
	dataLen := len(data)

	for i := 0; i < dataLen; i++ {
		b := data[i]

		// Soft line break
		if lineLen >= 72 {
			buf.B = append(buf.B, '=', '\r', '\n')
			lineLen = 0
		}

		// Trailing space check (RFC 2045)
		isSpace := b == ' ' || b == '\t'
		// If it's a space and the NEXT byte is a newline or end of data, we MUST encode it
		isTrailing := isSpace && (i+1 == dataLen || data[i+1] == '\r' || data[i+1] == '\n')

		if (b >= '!' && b <= '<') || (b >= '>' && b <= '~') || (isSpace && !isTrailing) {
			buf.B = append(buf.B, b)
			lineLen++
		} else {
			buf.B = append(buf.B, '=', hexTable[b>>4], hexTable[b&0x0f])
			lineLen += 3
		}

		// Reset lineLen on hard newlines to keep lines pretty
		if b == '\n' {
			lineLen = 0
		}
	}
}

func qEncodeSubject(buf *bytebufferpool.ByteBuffer, subject []byte) {
	buf.Write(str2bytes("\r\nSubject: =?UTF-8?Q?"))
	
	lineLen := 19 
	const hexTable = "0123456789ABCDEF"

	for i := 0; i < len(subject); {
		// Determine how many bytes the current UTF-8 character uses
		// This ensures we don't split an emoji across two encoded words.
			charLen := 1
			if subject[i] >= 0x80 {
				// Simple way to find UTF-8 char length without 'unicode/utf8' package
				if subject[i] >= 0xf0 { charLen = 4 } else if subject[i] >= 0xe0 { charLen = 3 } else if subject[i] >= 0xc0 { charLen = 2 }
			}

		// Check if adding this character (and its hex encoding) exceeds the limit
		// A 4-byte char becomes 12 hex chars. We check BEFORE writing.
			if lineLen + (charLen * 3) >= 70 && i > 0 {
				buf.Write(str2bytes("?=\r\n =?UTF-8?Q?"))
				lineLen = 11 
			}

		// Write the character (1 or more bytes)
			for j := 0; j < charLen && i < len(subject); j++ {
				b := subject[i]
				if (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') {
					buf.B = append(buf.B, b)
					lineLen++
				} else if b == ' ' {
					buf.B = append(buf.B, '_')
					lineLen++
				} else {
					buf.B = append(buf.B, '=', hexTable[b>>4], hexTable[b&0x0f])
					lineLen += 3
				}
				i++ // Increment outer loop
			}
	}
	buf.Write(str2bytes("?="))
}

func getMimeType(filename []byte) []byte {
	dotIdx := -1
	for i := len(filename) - 1; i >= 0; i-- {
		if filename[i] == '.' {
			dotIdx = i
			break
		}
		if filename[i] == '/' || filename[i] == '\\' {
			break
		}
	}

	if dotIdx == -1 || dotIdx == len(filename)-1 {
		return str2bytes("application/octet-stream")
	}

	ext := filename[dotIdx:]

	switch string(ext) {
	// --- DOCUMENTS ---
	case ".pdf":
		return str2bytes("application/pdf")
	case ".doc":
		return str2bytes("application/msword")
	case ".docx":
		return str2bytes("application/vnd.openxmlformats-officedocument.wordprocessingml.document")
	case ".xls":
		return str2bytes("application/vnd.ms-excel")
	case ".xlsx":
		return str2bytes("application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	case ".ppt":
		return str2bytes("application/vnd.ms-powerpoint")
	case ".pptx":
		return str2bytes("application/vnd.openxmlformats-officedocument.presentationml.presentation")

	// --- IMAGES ---
	case ".jpg", ".jpeg":
		return str2bytes("image/jpeg")
	case ".png":
		return str2bytes("image/png")
	case ".gif":
		return str2bytes("image/gif")
	case ".webp":
		return str2bytes("image/webp")
	case ".svg":
		return str2bytes("image/svg+xml")
	case ".ico":
		return str2bytes("image/x-icon")

	// --- TEXT ---
	case ".txt":
		return str2bytes("text/plain")
	case ".html", ".htm":
		return str2bytes("text/html")
	case ".csv":
		return str2bytes("text/csv")
	case ".css":
		return str2bytes("text/css")
	case ".js":
		return str2bytes("text/javascript")

	// --- ARCHIVES ---
	case ".zip":
		return str2bytes("application/zip")
	case ".rar":
		return str2bytes("application/vnd.rar")
	case ".7z":
		return str2bytes("application/x-7z-compressed")
	case ".tar":
		return str2bytes("application/x-tar")
	case ".gz":
		return str2bytes("application/gzip")

	// --- AUDIO/VIDEO ---
	case ".mp3":
		return str2bytes("audio/mpeg")
	case ".wav":
		return str2bytes("audio/wav")
	case ".mp4":
		return str2bytes("video/mp4")
	case ".mpeg":
		return str2bytes("video/mpeg")
	case ".avi":
		return str2bytes("video/x-msvideo")

	default:
		return str2bytes("application/octet-stream")
	}
}

func encodeBase64(buf *bytebufferpool.ByteBuffer, data []byte) {
	if len(data) == 0 { return }

	// Pre-calculate worst-case space: 
	// Base64 size + CRLF (\r\n) every 76 chars
		maxLen := base64.StdEncoding.EncodedLen(len(data)) + (len(data)/57+1)*2
		currLen := len(buf.B)
	
	// Capacity Guard
	// If the pool's buffer is too small, we grow it manually.
	// This is zero-alloc in the long run as the pool warms up.
		if cap(buf.B) < currLen+maxLen {
			newB := make([]byte, currLen, currLen+maxLen)
			copy(newB, buf.B) // Preserves headers/body already in the buffer
			buf.B = newB
		}

	// The Encoding Loop
		const chunkSize = 57 // 57 bytes in = 76 chars out (MIME standard)
		for i := 0; i < len(data); i += chunkSize {
			end := i + chunkSize
			if end > len(data) {
				end = len(data)
			}

			srcChunk := data[i:end]
			encodedLen := base64.StdEncoding.EncodedLen(len(srcChunk))
			
			// Shift the length forward to "claim" space for this chunk
				startOfChunk := len(buf.B)
				buf.B = buf.B[:startOfChunk+encodedLen]
			
			// Encode directly into the claimed memory
				base64.StdEncoding.Encode(buf.B[startOfChunk:], srcChunk)

			// Append the mandatory MIME line break
				buf.B = append(buf.B, '\r', '\n')
		}
}

func (m *MimeBuilder) SetFrom(name string, email string) *MimeBuilder {
	// Reset the internal buffer (Keep the RAM, set length to 0)
		m.from = m.from[:0]

	// Set name and email
		m.from = append(m.from, str2bytes(name)...)
		m.from = append(m.from, " <"...)
		m.from = append(m.from, str2bytes(email)...)
		m.from = append(m.from, ">"...)

	return m
}

func (m *MimeBuilder) AddTo(name string, email string) *MimeBuilder {
	// Append comma to add new address
		if len(m.to) > 0 {
			m.to = append(m.to, ", "...)
		}

	// Set name and email
		m.to = append(m.to, str2bytes(name)...)
		m.to = append(m.to, " <"...)
		m.to = append(m.to, str2bytes(email)...)
		m.to = append(m.to, ">"...)
		
	 return m
}

func (m *MimeBuilder) AddCC( email string, name string ) *MimeBuilder {
	// Append comma to add new address
		if len(m.cc) > 0 {
			m.cc = append(m.cc, ", "...)
		}

	// Set name and email
		m.cc = append(m.cc, str2bytes(name)...)
		m.cc = append(m.cc, " <"...)
		m.cc = append(m.cc, str2bytes(email)...)
		m.cc = append(m.cc, ">"...)
		
	 return m
}

func (m *MimeBuilder) AddBCC( email string, name string ) *MimeBuilder {
	// Append comma to add new address
		if len(m.bcc) > 0 {
			m.bcc = append(m.bcc, ", "...)
		}

	// Set name and email
		m.bcc = append(m.bcc, str2bytes(name)...)
		m.bcc = append(m.bcc, " <"...)
		m.bcc = append(m.bcc, str2bytes(email)...)
		m.bcc = append(m.bcc, ">"...)
		
	 return m
}

func (m *MimeBuilder) AddReplyTo( email string, name string ) *MimeBuilder {
	// Append comma to add new address
		if len(m.replyTo) > 0 {
			m.replyTo = append(m.replyTo, ", "...)
		}

	// Set name and email
		m.replyTo = append(m.replyTo, str2bytes(name)...)
		m.replyTo = append(m.replyTo, " <"...)
		m.replyTo = append(m.replyTo, str2bytes(email)...)
		m.replyTo = append(m.replyTo, ">"...)
		
	 return m
}

func (m *MimeBuilder) SetSubject(subject string) *MimeBuilder {
	m.subject = m.subject[:0]
	m.subject = append(m.subject, str2bytes(subject)...)
	return m
}

func (m *MimeBuilder) SetBody(content string) *MimeBuilder {
	m.body = m.body[:0]
	m.body = append(m.body, str2bytes(content)...)
	return m
}

func (m *MimeBuilder) AsHTML() *MimeBuilder {
	m.isHTML = true
	return m
}

func (m *MimeBuilder) SetAltBody( content string ) *MimeBuilder {
	m.altBody = m.altBody[:0]
	m.altBody = append(m.altBody, str2bytes(content)...)
	return m
}

// Embed adds an inline image (CID) using a byte slice.
// name: The filename (e.g., "logo.png")
// data: The raw bytes of the image
// cid:  The unique ID used in HTML (e.g., "company_logo")
func (m *MimeBuilder) Embed(name string, data []byte, cid string) *MimeBuilder {
	m.inlineImages = append(m.inlineImages, InlineImage{
		Filename:  name,
		Data:      data,
		ContentID: cid,
	})
	return m
}

func (m *MimeBuilder) Attach(filename string, data []byte) *MimeBuilder {
	m.attachments = append(m.attachments, Attachment{
		Filename: str2bytes(filename),
		Data:     data,
	})
	return m
}

func (m *MimeBuilder) AttachReader(filename string, r io.Reader) *MimeBuilder {
	m.attachments = append(m.attachments, Attachment{
		Filename: str2bytes(filename),
		Stream:   r,
	})
	return m
}

func (m *MimeBuilder) AttachStream(filename string, r io.Reader) *MimeBuilder {
	return m.AttachReader(filename, r)
}

// Generate and set boundaries: mixed, alternative and related
func (m *MimeBuilder) setBoundaries() {
	// Fetch current entropy (Time ^ Salt ^ PID)
		var salt [16]byte
		rand.Read( salt[:] )
		firstHalf := binary.LittleEndian.Uint64(salt[0:8])
		secondHalf := binary.LittleEndian.Uint64(salt[8:16])

		nanoTime := uint64(time.Now().UnixNano())
		pid := uint32(os.Getpid())

		firstHalf ^= nanoTime
		secondHalf ^= (uint64(pid) << 32)

		// fmt.Println( "Salt:",salt, "\nFirsthalf:", firstHalf, "\nSecondHalf:", secondHalf )

	// 32-bytes of hex
		// Encode FirstHalf (0-15)
			const hexTable = "0123456789abcdef"
			var boundary [32]byte
			for i := 0; i < 8; i++ {
				b := byte(firstHalf >> (i * 8))
				boundary[i*2] = hexTable[b>>4]
				boundary[i*2+1] = hexTable[b&0x0f]
			}

		// Encode SecondHalf (16-31)
			for i := 0; i < 8; i++ {
				b := byte(secondHalf >> (i * 8))
				boundary[16+i*2] = hexTable[b>>4]
				boundary[16+i*2+1] = hexTable[b&0x0f]
			}

		// fmt.Println( "\nBoundary: ", boundary, "\nBoundary String: ", string(boundary[:]) )
		
	// Generate mixed, alternative & related boundaries
		boundary[31] = '1'
		copy( m.mixedBoundary[:], boundary[:] )

		boundary[31] = 'a'
		copy( m.altBoundary[:], boundary[:] )

		boundary[31] = 'e'
		copy( m.relBoundary[:], boundary[:] )

	// fmt.Println( "\n\nMixed: ", m.mixedBoundary, "\nAlt: ", m.altBoundary, "\nRelated: ", m.relBoundary )
}

/***************************
		buildMixed()
			- Content-Type: multipart/mixed; boundary="mixedBoundary"
			--<mixedBoundary>
			- call buildPlainText() or buildHtml() or buildAlternative()
			- call buildAttachments()
		buildAlternative()
			- Content-Type: multipart/alternative; boundary="altBoundary"
			--<altBoundary>
			- call buildPlainText()
			--<altBoundary>
			- call buildHtml() or buildRelated()
			--<altBoundary>--
		buildRelated()
			- Content-Type: multipart/related; boundary="relatedBoundary"
			--<relatedBoundary>
			- call buildHtml()
			- call buildInlineImages()

		buildPlainText()
			Content-Type: text/plain; charset=UTF-8
			Content-Transfer-Encoding: quoted-printable
			Hello in plain text.

		buildHtml()
			Content-Type: text/html; charset=UTF-8
			Content-Transfer-Encoding: quoted-printable
			<html><body><p>Hello in HTML</p></body></html>

		buildInlineImages()
			--<relatedBoundary>
			Content-Type: image/png; name="logo.png"
			Content-Transfer-Encoding: base64
			Content-Disposition: inline; filename="logo.png"
			Content-ID: <company_logo>
			<base64-encoded image data>
			--<relatedBoundary>--

		buildAttachments()
			--<mixedBoundary>
			Content-Type: application/pdf; name="report.pdf"
			Content-Transfer-Encoding: base64
			Content-Disposition: attachment; filename="report.pdf"
			<base64-encoded data>
			--<mixedBoundary>--
***************************/



func (m *MimeBuilder) buildMixed( buf *bytebufferpool.ByteBuffer ){
	// Content-Type: multipart/mixed; boundary="mixedBoundary"
			buf.Write(str2bytes( "\r\nContent-Type: multipart/mixed; boundary=\"" ))
			buf.Write( m.mixedBoundary[:] )
			buf.Write(str2bytes( "\"\r\n\r\n" ))
	
	// --<mixedBoundary>
		buf.Write(str2bytes( "--" ))
		buf.Write( m.mixedBoundary[:] )

	// Call buildPlainText() or buildHtml() or buildAlternative()
		if len(m.body)>0 && len(m.altBody)>0 && m.isHTML {
			m.buildAlternative( buf )
		} else if m.isHTML {
			m.buildHtml( buf )
		} else {
			m.buildPlainText( buf )
		}

	// Call buildAttachments()
		m.buildAttachments( buf )
}

func (m *MimeBuilder) buildAlternative( buf *bytebufferpool.ByteBuffer ){
	// Content-Type: multipart/alternative; boundary="altBoundary"
		buf.Write(str2bytes( "\r\nContent-Type: multipart/alternative; boundary=\"" ))
		buf.Write( m.altBoundary[:] )
		buf.Write(str2bytes( "\"\r\n\r\n" ))
	// --<altBoundary>
		buf.Write(str2bytes( "--" ))
		buf.Write( m.altBoundary[:] )

	// Call buildPlainText()
		m.buildPlainText( buf )

	// --<altBoundary>
		buf.Write(str2bytes( "--" ))
		buf.Write( m.altBoundary[:] )

	// Call buildHtml() or buildRelated()
		if len(m.inlineImages)>0 {
			m.buildRelated( buf )
		} else {
			m.buildHtml( buf )
		}

	// --<altBoundary>--
		buf.Write(str2bytes( "--" ))
		buf.Write( m.altBoundary[:] )
		buf.Write(str2bytes( "--" ))
}

func (m *MimeBuilder) buildRelated( buf *bytebufferpool.ByteBuffer ){
	// Content-Type: multipart/related; boundary="relatedBoundary"
		buf.Write(str2bytes( "\r\nContent-Type: multipart/related; boundary=\"" ))
		buf.Write( m.relBoundary[:] )
		buf.Write(str2bytes( "\"\r\n\r\n" ))

	// --<relatedBoundary>
		buf.Write(str2bytes( "--" ))
		buf.Write( m.relBoundary[:] )

	// Call buildHtml()
		m.buildHtml( buf )

	// Call buildInlineImages()
		m.buildInlineImages( buf )
}

func (m *MimeBuilder) buildHtml( buf *bytebufferpool.ByteBuffer ){
	// Content-Type: text/html; charset=UTF-8
		buf.Write(str2bytes( "\r\nContent-Type: text/html; charset=UTF-8" ))
	// Content-Transfer-Encoding: quoted-printable
		buf.Write(str2bytes( "\r\nContent-Transfer-Encoding: quoted-printable\r\n\r\n" ))
	// <html><body><p>Hello in HTML</p></body></html>
		qpEncode( buf, m.body )

		buf.Write(str2bytes("\r\n"))
}

func (m *MimeBuilder) buildPlainText( buf *bytebufferpool.ByteBuffer ){
	// Content-Type: text/plain; charset=UTF-8
		buf.Write(str2bytes( "\r\nContent-Type: text/plain; charset=UTF-8" ))
	// Content-Transfer-Encoding: quoted-printable
		buf.Write(str2bytes( "\r\nContent-Transfer-Encoding: quoted-printable\r\n\r\n" ))
	// Hello in plain text.
		if !m.isHTML {
			qpEncode( buf, m.body )
		} else {
			qpEncode( buf, m.altBody )
		}
		buf.Write(str2bytes("\r\n"))
}

func (m *MimeBuilder) buildInlineImages( buf *bytebufferpool.ByteBuffer ){
	buf.Write(str2bytes( "\r\nIni adalah inlineImages\r\n\r\n" ))
}

func (m *MimeBuilder) buildAttachments( buf *bytebufferpool.ByteBuffer ){
	// buf.Write(str2bytes( "\r\nIni adalah attachments\r\n\r\n" ))
	for _, attach := range m.attachments {
		// --<mixedBoundary>
			buf.Write(str2bytes( "\r\n\r\n--" ))
			buf.Write( m.mixedBoundary[:] )

		// Content-Type: application/pdf; name="report.pdf"
			buf.Write(str2bytes( "\r\nContent-Type: " ))
			buf.Write( getMimeType(attach.Filename) )
			buf.Write(str2bytes( "; name=\"" ))
			buf.Write(attach.Filename)
			buf.Write(str2bytes( "\"" ))

		// Content-Transfer-Encoding: base64
			buf.Write(str2bytes( "\r\nContent-Transfer-Encoding: base64" ))

		// Content-Disposition: attachment; filename="report.pdf"
			buf.Write(str2bytes( "\r\nContent-Disposition: attachment; filename=\"" ))
			buf.Write(attach.Filename)
			buf.Write(str2bytes( "\"\r\n" ))

		// <base64-encoded data>
			encodeBase64( buf, attach.Data )
	}

	// --<mixedBoundary>--
		buf.Write(str2bytes( "--" ))
		buf.Write( m.mixedBoundary[:] )
		buf.Write(str2bytes( "--" ))
}

func (m *MimeBuilder) Build() ([]byte, error) {
	// Borrow a high-performance buffer from the pool
		buf := bytebufferpool.Get()
		defer bytebufferpool.Put(buf)

	// Generate header
		// (from, to, cc, bcc, reply-to)
			buf.Write(str2bytes( "From: " ))
			buf.Write(m.from[:])
			buf.Write(str2bytes( "\r\nTo: " ))
			buf.Write(m.to[:])
			if len(m.cc)>0 {
				buf.Write(str2bytes( "\r\nCc: " ))
				buf.Write(m.cc[:])
			}
			if len(m.bcc)>0 {
				buf.Write(str2bytes( "\r\nBcc: " ))
				buf.Write(m.bcc[:])
			}
			if len(m.replyTo)>0 {
				buf.Write(str2bytes( "\r\nReply-To: " ))
				buf.Write(m.replyTo[:])
			}
		// subject, mime-version
			qEncodeSubject( buf, m.subject[:] )
			buf.Write(str2bytes( "\r\nMIME-Version: 1.0" ))

	// Generate body
	// for content-type: mixed, alt, rel, html, plain
		// mixed - if there is an attachment
			if len(m.attachments)>0 {
				m.setBoundaries()
				m.buildMixed( buf )

		// alt - if there are both altBody & body (html)
			} else if m.isHTML && len(m.body)>0 && len(m.altBody)>0 {
				m.setBoundaries()
				m.buildAlternative( buf )

		// rel - if there are both body(html) & inline-image
			} else if m.isHTML && len(m.body)>0 && len(m.inlineImages)>0 {
				m.setBoundaries()
				m.buildRelated( buf )
		// HTML
			} else if m.isHTML && len(m.body)>0 {
				m.buildHtml( buf )
		// Plain-text
			} else if len(m.altBody)>0 {
				m.buildPlainText( buf )
			}

	fmt.Println("--- DEBUG START ---")
	fmt.Println(buf.String())
	fmt.Println("--- DEBUG END ---")


	// fmt.Println( "\n\nMixed: ", m.mixedBoundary, "\nAlt: ", m.altBoundary, "\nRelated: ", m.relBoundary )
	return nil,nil
}

 

/*
	CONSTRUCT----------
		- New()

	HEADER-------------
		- SetFrom()
		- AddTo()
		- AddCC()
		- AddBCC()
		- AddReplyTo()
		- SetSubject()

	BODY---------------
		- SetBody().AsHTML()
		- SetAltBody()

		- Embed(filename string, data []byte, cid string)
		- Attach(filename string, data []byte)
		- AttachReader(filename string, r io.Reader)
		- AttachStream(filename string, r io.Reader) // alias of AttachReader()

	GENERATE-----------
		- Build() ([]byte, error)
		- Bytes() ([]byte, error) // alias of Build()
		- WriteTo(w io.Writer) error
*/