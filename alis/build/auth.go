package build

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	AuthServerURL = "https://identity.alisx.com"
)

type AuthClient struct {
	ID          string
	Secret      string
	RedirectURL string
}

func (c *AuthClient) AuthorizeURL(state string) string {
	return fmt.Sprintf("%s/authorize?client_id=%s&redirect_uri=%s&state=%s", AuthServerURL, c.ID, c.RedirectURL, state)
}

type ExchangeCodeResponse struct {
	*Tokens
	Email string `json:"email"`
	Sub   string `json:"sub"`
}

// ExchangeCode exchanges an authorization code for access and refresh tokens
func (c *AuthClient) ExchangeCode(code string) (*ExchangeCodeResponse, error) {
	// exchange code
	tokens := &Tokens{}
	err := c.postToken(tokens, authorizationCode, code)
	if err != nil {
		return nil, fmt.Errorf("posting token: %w", err)
	}
	resp := &ExchangeCodeResponse{Tokens: tokens}

	// extract sub and email from jwt
	accessTokenParts := strings.Split(tokens.AccessToken, ".")
	if len(accessTokenParts) != 3 {
		return nil, fmt.Errorf("invalid access token returned from authorization server")
	}
	buffer := bytes.NewBuffer([]byte(accessTokenParts[1]))
	decodedB64Reader := base64.NewDecoder(base64.RawURLEncoding, buffer)
	decoder := json.NewDecoder(decodedB64Reader)
	if err := decoder.Decode(resp); err != nil {
		return nil, fmt.Errorf("decoding access token body %s: %w", accessTokenParts[1], err)
	}

	// return
	return resp, nil
}

type AuthenticateResponse struct {
	Refreshed bool
}

// Authenticate refreshes the user's access token if its expired.
func (c *AuthClient) Authenticate(tokens *Tokens, now time.Time) (*AuthenticateResponse, error) {
	// fail if no correctly formatted access token is stored
	if tokens.AccessToken == "" {
		return nil, fmt.Errorf("missing access token for user")
	}
	accessTokenParts := strings.Split(tokens.AccessToken, ".")
	if len(accessTokenParts) != 3 {
		return nil, fmt.Errorf("invalid access token stored for user")
	}

	// decode access token
	buffer := bytes.NewBuffer([]byte(accessTokenParts[1]))
	decodedB64Reader := base64.NewDecoder(base64.RawURLEncoding, buffer)
	decoder := json.NewDecoder(decodedB64Reader)
	type jwtType struct {
		Exp int64 `json:"exp"`
	}
	jwt := &jwtType{}
	if err := decoder.Decode(jwt); err != nil {
		return nil, fmt.Errorf("decoding access token: %w", err)
	}

	// try to refresh token if it's expired
	expTime := time.Unix(jwt.Exp, 0)
	if expTime.Before(now) {
		if err := c.refresh(tokens); err != nil {
			return nil, err
		}
		return &AuthenticateResponse{Refreshed: true}, nil
	}

	// no refresh needed
	return &AuthenticateResponse{Refreshed: false}, nil
}

func (c *AuthClient) refresh(tokens *Tokens) error {
	err := c.postToken(tokens, refreshToken, tokens.RefreshToken)
	if err != nil {
		return err
	}
	return nil
}

type grantType string

const (
	refreshToken      grantType = "refresh_token"
	authorizationCode grantType = "authorization_code"
)

type Tokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func (c *AuthClient) postToken(tokens *Tokens, grantType grantType, grant string) error {
	// build request body
	type bodyType struct {
		GrantType    string `json:"grant_type"`
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
		Code         string `json:"code,omitempty"`
		RedirectURI  string `json:"redirect_uri"`
		RefreshToken string `json:"refresh_token,omitempty"`
	}
	body := &bodyType{
		GrantType:    string(grantType),
		ClientID:     c.ID,
		ClientSecret: c.Secret,
	}
	switch grantType {
	case refreshToken:
		body.RefreshToken = grant
	case authorizationCode:
		body.Code = grant
		body.RedirectURI = c.RedirectURL
	}
	bytesBuffer := bytes.NewBuffer(nil)
	jsonEncoder := json.NewEncoder(bytesBuffer)
	if err := jsonEncoder.Encode(body); err != nil {
		return fmt.Errorf("encoding body: %w", err)
	}

	// make request
	req, err := http.NewRequest(http.MethodPost, AuthServerURL+"/token", bytesBuffer)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("sending request: %w", err)
	}

	// handle error response
	if resp.StatusCode != 200 {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("reading response body: %w", err)
		}
		return fmt.Errorf("%d: %s", resp.StatusCode, bodyBytes)
	}

	// handle success response
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(tokens); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}
	return nil
}
