package mimebuilder



import (
	"io"
	// "fmt"
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
	return &MimeBuilder{}
}

func (m *MimeBuilder) SetFrom( email string, name string ) *MimeBuilder {
	return m
}

func (m *MimeBuilder) AddTo( email string, name string ) *MimeBuilder {
	return m
}

func (m *MimeBuilder) AddCC( email string, name string ) *MimeBuilder {
	return m
}

func (m *MimeBuilder) AddBCC( email string, name string ) *MimeBuilder {
	return m
}

func (m *MimeBuilder) AddReplyTo( email string, name string ) *MimeBuilder {
	return m
}

func (m *MimeBuilder) SetSubject(  ) *MimeBuilder {
	return m
}

func (m *MimeBuilder) SetBody(  ) *MimeBuilder {
	return m
}

func (m *MimeBuilder) AsHTML(  ) *MimeBuilder {
	return m
}

func (m *MimeBuilder) SetAltBody(  ) *MimeBuilder {
	return m
}

func (m *MimeBuilder) Embed(  ) *MimeBuilder {
	return m
}

func (m *MimeBuilder) Attach(  ) *MimeBuilder {
	return m
}

func (m *MimeBuilder) Build(  ) *MimeBuilder {
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