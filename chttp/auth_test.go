package chttp

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"testing"

	"github.com/flimzy/diff"
	"github.com/flimzy/kivik"
)

func TestDefaultAuth(t *testing.T) {
	dsn, err := url.Parse(dsn(t))
	if err != nil {
		t.Fatalf("Failed to parse DSN '%s': %s", dsn, err)
	}
	user := dsn.User.Username()
	client := getClient(t)

	if name := getAuthName(client, t); name != user {
		t.Errorf("Unexpected authentication name. Expected '%s', got '%s'", user, name)
	}

	if err = client.Logout(context.Background()); err != nil {
		t.Errorf("Failed to de-authenticate: %s", err)
	}

	if name := getAuthName(client, t); name != "" {
		t.Errorf("Unexpected authentication name after logout '%s'", name)
	}
}

func TestBasicAuth(t *testing.T) {
	dsn, err := url.Parse(dsn(t))
	if err != nil {
		t.Fatalf("Failed to parse DSN '%s': %s", dsn, err)
	}
	user := dsn.User
	dsn.User = nil
	client, err := New(context.Background(), dsn.String())
	if err != nil {
		t.Fatalf("Failed to connect: %s", err)
	}
	if name := getAuthName(client, t); name != "" {
		t.Errorf("Unexpected authentication name '%s'", name)
	}

	if err = client.Logout(context.Background()); err == nil {
		t.Errorf("Logout should have failed prior to login")
	}

	password, _ := user.Password()
	ba := &BasicAuth{
		Username: user.Username(),
		Password: password,
	}
	if err = client.Auth(context.Background(), ba); err != nil {
		t.Errorf("Failed to authenticate: %s", err)
	}
	if err = client.Auth(context.Background(), ba); err == nil {
		t.Errorf("Expected error trying to double-auth")
	}
	if name := getAuthName(client, t); name != user.Username() {
		t.Errorf("Unexpected auth name. Expected '%s', got '%s'", user.Username(), name)
	}

	if err = client.Logout(context.Background()); err != nil {
		t.Errorf("Failed to de-authenticate: %s", err)
	}

	if name := getAuthName(client, t); name != "" {
		t.Errorf("Unexpected authentication name after logout '%s'", name)
	}
}

func getAuthName(client *Client, t *testing.T) string {
	result := struct {
		Ctx struct {
			Name string `json:"name"`
		} `json:"userCtx"`
	}{}
	if _, err := client.DoJSON(context.Background(), "GET", "/_session", nil, &result); err != nil {
		t.Errorf("Failed to check session info: %s", err)
	}
	return result.Ctx.Name
}

type mockRT struct {
	resp *http.Response
	err  error
}

var _ http.RoundTripper = &mockRT{}

func (rt *mockRT) RoundTrip(_ *http.Request) (*http.Response, error) {
	return rt.resp, rt.err
}

func TestCookieAuthAuthenticate(t *testing.T) {
	tests := []struct {
		name           string
		auth           *CookieAuth
		client         *Client
		err            string
		expectedCookie *http.Cookie
	}{
		{
			name: "standard request",
			auth: &CookieAuth{
				Username: "foo",
				Password: "bar",
			},
			client: &Client{
				Client: &http.Client{
					Transport: &mockRT{
						resp: &http.Response{
							Header: http.Header{
								"Set-Cookie": []string{
									"AuthSession=cm9vdDo1MEJCRkYwMjq0LO0ylOIwShrgt8y-UkhI-c6BGw; Version=1; Path=/; HttpOnly",
								},
							},
							Body: ioutil.NopCloser(strings.NewReader(`{"userCtx":{"name":"foo"}}`)),
						},
					},
				},
				dsn: &url.URL{Scheme: "http", Host: "foo.com"},
			},
			expectedCookie: &http.Cookie{
				Name:  kivik.SessionCookieName,
				Value: "cm9vdDo1MEJCRkYwMjq0LO0ylOIwShrgt8y-UkhI-c6BGw",
			},
		},
		{
			name: "Invalid JSON response",
			auth: &CookieAuth{
				Username: "foo",
				Password: "bar",
			},
			client: &Client{
				Client: &http.Client{
					Jar: &cookiejar.Jar{},
					Transport: &mockRT{
						resp: &http.Response{
							Body: ioutil.NopCloser(strings.NewReader(`{"asdf"}`)),
						},
					},
				},
				dsn: &url.URL{Scheme: "http", Host: "foo.com"},
			},
			err: "invalid character '}' after object key",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.auth.Authenticate(context.Background(), test.client)
			var errMsg string
			if err != nil {
				errMsg = err.Error()
			}
			if errMsg != test.err {
				t.Errorf("Unexpected error: %s", errMsg)
			}
			if err != nil {
				return
			}
			cookie, ok := test.auth.Cookie()
			if !ok {
				t.Errorf("Expected cookie")
				return
			}
			if d := diff.Interface(test.expectedCookie, cookie); d != nil {
				t.Error(d)
			}
		})
	}
}
