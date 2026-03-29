package mimebuilder



import (
	"io"
	"fmt"
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
	from 		string
	to 			[]string
	cc 			[]string
	bcc 		[]string
	replyTo 	[]string

	subject 	string
	body 		string
	altBody 	string
	contentType string

	attachments 	[]Attachment
	inlineImages 	[]InlineImage

	errorList 	[]error
}


func New() *MimeBuilder {
	return &MimeBuilder{contentType: "text/plain"}
}

func (m *MimeBuilder) SetFrom(name string, email string) *MimeBuilder {
	m.from = fmt.Sprintf("%s <%s>", name, email)
	return m
}

func (m *MimeBuilder) AddTo(name string, email string) *MimeBuilder {
	m.to = append(m.to, fmt.Sprintf("%s <%s>", name, email))
	return m
}

func (m *MimeBuilder) AddCC( email string, name string ) *MimeBuilder {
	m.cc = append(m.cc, fmt.Sprintf("%s <%s>", name, email))
	return m
}

func (m *MimeBuilder) AddBCC( email string, name string ) *MimeBuilder {
	m.bcc = append(m.bcc, fmt.Sprintf("%s <%s>", name, email))
	return m
}

func (m *MimeBuilder) AddReplyTo( email string, name string ) *MimeBuilder {
	m.replyTo = append(m.replyTo, fmt.Sprintf("%s <%s>", name, email))
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
	m.contentType = "text/html"
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

func (m *MimeBuilder) Build() *MimeBuilder {
	return m
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