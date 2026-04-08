package main

import (
	"github.com/elmyrockers/go-mimebuilder"
	"os"
	// "fmt"
)

func main() {
	data, _ := os.ReadFile( "../email.jpg" )
	mimebuilder.New().
				SetFrom( "elmyrockers@gmail.com", "Helmi Aziz" ).
				AddTo( "test@yahoo.com", "test" ).
				AddTo( "test2@yahoo.com", "test2" ).
				SetSubject( "Test Sahaja" ).
				SetBody( "Ini adalah html" ).AsHTML().
				SetAltBody( "Ini adalah plain 100%" ).
				Attach( "test.jpg", data ).
				Build()

	// fmt.Println( data )
}