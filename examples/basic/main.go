package main

import (
	"github.com/elmyrockers/go-mimebuilder"
	"os"
	// "fmt"
)

func main() {
	// emailjpg, _ := os.ReadFile( "../../email.jpg" )
	checkpng, _ := os.ReadFile( "../../check.png" )
	informationpng, _ := os.ReadFile( "../../Information_icon.svg.png" )
	mimebuilder.New().
				SetFrom( "Helmi Aziz", "elmyrockers@gmail.com" ).
				AddTo( "test", "test@yahoo.com" ).
				AddTo( "test2", "test2@yahoo.com" ).
				SetSubject( "Test Sahaja 😭" ).
				SetBody( "<html><body><h1>Ini adalah yeay..........	â.....💁👌🎍😍......hjkghgjhf.................hgjhggjhghj......ÿ.........jhgjhgjhiuyuiu..............iuyiuyiuy.............Ç............hehhehe.............................ehehehe.................... html</h1></body></html>" ).AsHTML().
				SetAltBody( "Ini adalah plain 100%" ).
				// Attach( "email.jpg", emailjpg ).
				Attach( "check.png", checkpng ).
				Attach( "information.png", informationpng ).
				Build()

	// fmt.Println( data )
}