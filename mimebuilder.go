package mimebuilder



import (
	"io"
	"unsafe"
	"fmt"
	"time"
	"os"
	"crypto/rand"
	// "encoding/hex"
	"encoding/binary"

	"github.com/valyala/bytebufferpool"
)

type Attachment struct {
	Filename 	string
	Stream 		io.Reader
	Data 		[]byte
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
		Filename: filename,
		Data:     data,
	})
	return m
}

func (m *MimeBuilder) AttachReader(filename string, r io.Reader) *MimeBuilder {
	m.attachments = append(m.attachments, Attachment{
		Filename: filename,
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
func (m *MimeBuilder) Build() ([]byte, error) {
	// Borrow a high-performance buffer from the pool
		buf := bytebufferpool.Get()
		defer bytebufferpool.Put(buf)
		
	// Generate header
		// (from, to, cc, bcc, reply-to)
			buf.Write(str2bytes( "From: " ))
			buf.Write(m.from[:])
			buf.Write(str2bytes( "\nTo: " ))
			buf.Write(m.to[:])
			if len(m.cc)>0 {
				buf.Write(str2bytes( "\nCc: " ))
				buf.Write(m.cc[:])
			}
			if len(m.bcc)>0 {
				buf.Write(str2bytes( "\nBcc: " ))
				buf.Write(m.bcc[:])
			}
			if len(m.replyTo)>0 {
				buf.Write(str2bytes( "\nReply-To: " ))
				buf.Write(m.replyTo[:])
			}
			
		// subject, mime-version
			buf.Write(str2bytes( "\nSubject: " ))
			buf.Write(m.subject[:])
			buf.Write(str2bytes( "\nMIME-Version: 1.0" ))
	// Generate body
		// content-type (mixed, alt, rel, html, plain)
		// Pre-allocate boundaries
			m.setBoundaries()
		// mixed - if there is an attachment
			// Content-Type: multipart/mixed; boundary=
		// alt - if there are both altBody & body (html)
			// Content-Type: multipart/alternative; boundary=
		// rel - if there are both body(html) & inline-image
			// Content-Type: multipart/related; boundary=
		// HTML
			// Content-Type: text/html; charset=UTF-8
		// Plain-text
			// Content-Type: text/plain; charset=UTF-8

	fmt.Println("--- DEBUG START ---")
	fmt.Println(buf.String())
	fmt.Println("--- DEBUG END ---")


	fmt.Println( "\n\nMixed: ", m.mixedBoundary, "\nAlt: ", m.altBoundary, "\nRelated: ", m.relBoundary )
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