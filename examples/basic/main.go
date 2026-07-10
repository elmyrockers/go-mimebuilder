package main

import (
	"github.com/elmyrockers/go-mimebuilder"
	"os"
	"fmt"
)

func main() {
	favoritepng, _ := os.ReadFile( "favorite.png" )
	idolpng, _ := os.ReadFile( "idol.png" )
	moorishpng, _ := os.ReadFile( "moorish-idol.png" )
	builder := mimebuilder.New()

	mime, _ := builder.
					// SetFrom( "elmyrockers@gmail.com", "Helmi Aziz" ).
					SetFrom( "elmyrockers@gmail.com", "" ).
					// AddTo( "test@yahoo.com", "test" ).
					// AddTo( "test2@yahoo.com", "test2" ).
					AddTo( "test@yahoo.com", "" ).
					AddTo( "test2@yahoo.com", "To2" ).

					AddCC( "cc1@yahoo.com", "" ).
					AddCC( "cc2@yahoo.com", "CC2" ).

					AddBCC( "bcc1@yahoo.com", "" ).
					AddBCC( "bcc2@yahoo.com", "BCC2" ).

					AddReplyTo( "replyto1@yahoo.com", "" ).
					AddReplyTo( "replyto2@yahoo.com", "ReplyTo2" ).

					SetSubject( "Test Sahaja 😭" ).
					SetBody( "<html><body><h1>Ini adalah yeay..........	â.....💁👌🎍😍......hjkghgjhf.................hgjhggjhghj......ÿ.........jhgjhgjhiuyuiu..............iuyiuyiuy.............Ç............hehhehe.............................ehehehe.................... html <img src=\"cid:cid1\"></h1></body></html>" ).
					AsHTML().
					// SetBody( "Ini adalah plain 100%" ).
					SetAltBody( "Ini adalâh plain 100%" ).
					Embed( "favorite.png", favoritepng, "cid1" ).
					Embed( "idol.png", idolpng, "cid2" ).
					
					Attach( "moorish-idol.png", moorishpng ).
					Build()
	defer builder.Release(mime)

	fmt.Print( mime.String() )
	// fmt.Println( data )
}

/*******************************************
 *  Plain-text only
 * 	HTML only
 * 	Alternative
 *  	- Plain-text and html
 * 		- text, html and embed
 * 		
 * 
 * ****************************************/