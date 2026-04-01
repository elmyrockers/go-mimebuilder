package mimebuilder



import (
	"io"
	"unsafe"

	// "github.com/valyala/bytebufferpool"
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
	from 		[]byte
	to 			[]byte
	cc 			[]byte
	bcc 		[]byte
	replyTo 	[]byte

	subject 	string
	body 		string
	altBody 	string
	isHTML 		bool

	attachments 	[]Attachment
	inlineImages 	[]InlineImage

	errorList 	[]error
}

func New() *MimeBuilder {
	return &MimeBuilder{isHTML: false}
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
	m.subject = subject
	return m
}

func (m *MimeBuilder) SetBody(content string) *MimeBuilder {
	m.body = content
	return m
}

func (m *MimeBuilder) AsHTML() *MimeBuilder {
	m.isHTML = true
	return m
}

func (m *MimeBuilder) SetAltBody( content string ) *MimeBuilder {
	m.altBody = content
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
func (m *MimeBuilder) setBoundaries( mixed, alternative, related *[]byte ) {
	// Fetch current entropy (Time ^ Salt ^ PID)
}

func (m *MimeBuilder) Build() ([]byte, error) {
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