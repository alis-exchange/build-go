package build_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/alis-exchange/build-go/alis/build"
)

func testAuthClient() *build.AuthClient {
	clientID := os.Getenv("ALIS_CLIENT_ID")
	clientSecret := os.Getenv("ALIS_CLIENT_SECRET")
	if clientID == "" || clientSecret == "" {
		panic("missing ALIS_CLIENT_ID or ALIS_CLIENT_SECRET env")
	}
	return &build.AuthClient{
		ID:          clientID,
		Secret:      clientSecret,
		RedirectURL: "http://localhost:8080/auth/callback",
	}
}

func Test_AuthorizeURL(t *testing.T) {
	println(testAuthClient().AuthorizeURL("mystate"))
}

func Test_ExchangeCode(t *testing.T) {
	code := "PASTE_CODE_HERE"
	tokens, err := testAuthClient().ExchangeCode(code)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("tokens: %+v", tokens.Tokens)
	// write tokens to ".env" file
	file, err := os.Create(".env")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	_, err = fmt.Fprintf(file, "ALIS_ACCESS_TOKEN=%s\n", tokens.AccessToken)
	if err != nil {
		t.Fatal(err)
	}
	_, err = fmt.Fprintf(file, "ALIS_REFRESH_TOKEN=%s\n", tokens.RefreshToken)
	if err != nil {
		t.Fatal(err)
	}
}
