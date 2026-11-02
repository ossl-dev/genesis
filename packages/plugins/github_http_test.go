package plugins

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func useHTTPClient(t *testing.T, client *http.Client) {
	t.Helper()
	previous := HTTPClient
	HTTPClient = client
	t.Cleanup(func() { HTTPClient = previous })
}
func downloadResponse(request *http.Request, status int, location string) *http.Response {
	headers := make(http.Header)
	if location != "" {
		headers.Set("Location", location)
	}
	return &http.Response{StatusCode: status, Header: headers, Body: io.NopCloser(strings.NewReader("ok")), Request: request}
}
func TestGitHubAPITokenScope(t *testing.T) {
	for _, test := range []struct{ name, url, github, gh, want string }{
		{"primary", "https://api.github.com/repos/example/release", "primary-token", "fallback-token", "Bearer primary-token"},
		{"fallback", "https://api.github.com/repos/example/release", "", "fallback-token", "Bearer fallback-token"},
		{"empty primary", "https://api.github.com/repos/example/release", "  ", "fallback-token", "Bearer fallback-token"},
		{"unset", "https://api.github.com/repos/example/release", "", "", ""},
		{"host case", "https://API.GITHUB.COM/repos/example/release", "primary-token", "", "Bearer primary-token"},
		{"default port", "https://api.github.com:443/repos/example/release", "primary-token", "", "Bearer primary-token"},
		{"insecure", "http://api.github.com/repos/example/release", "primary-token", "", ""},
		{"subdomain", "https://downloads.api.github.com/archive", "primary-token", "", ""},
		{"different port", "https://api.github.com:8443/repos/example/release", "primary-token", "", ""},
		{"downloads", "https://github.com/example/releases/download/archive.zip", "primary-token", "", ""},
		{"unrelated", "https://example.com/archive.zip", "primary-token", "", ""},
		{"userinfo", "https://user:password@api.github.com/repos/example/release", "primary-token", "", "Basic dXNlcjpwYXNzd29yZA=="},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("GITHUB_TOKEN", test.github)
			t.Setenv("GH_TOKEN", test.gh)
			calls := 0
			client := &http.Client{Transport: transportFunc(func(request *http.Request) (*http.Response, error) {
				calls++
				if got := request.Header.Get("Authorization"); got != test.want {
					t.Fatalf("authorization=%q want=%q", got, test.want)
				}
				return downloadResponse(request, 200, ""), nil
			})}
			useHTTPClient(t, client)
			body, err := fetchBody(context.Background(), test.url)
			if err != nil {
				t.Fatal(err)
			}
			body.Close()
			if calls != 1 {
				t.Fatal(calls)
			}
			if client.CheckRedirect != nil {
				t.Fatal("shared client modified")
			}
		})
	}
}
func TestGitHubTokenRedirectScope(t *testing.T) {
	for _, target := range []string{"https://downloads.example.com/archive", "https://assets.api.github.com/archive", "http://api.github.com/archive", "https://api.github.com:8443/archive", "https://api.github.com:443/archive", "https://api.github.com/archive"} {
		t.Run(target, func(t *testing.T) {
			t.Setenv("GITHUB_TOKEN", "test-secret")
			t.Setenv("GH_TOKEN", "")
			calls := 0
			useHTTPClient(t, &http.Client{Transport: transportFunc(func(request *http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					if request.Header.Get("Authorization") != "Bearer test-secret" {
						t.Fatal("initial API request unauthenticated")
					}
					return downloadResponse(request, 302, target), nil
				}
				want := ""
				if githubAPIOrigin(request.URL) {
					want = "Bearer test-secret"
				}
				if got := request.Header.Get("Authorization"); got != want {
					t.Fatalf("redirect authorization=%q want=%q", got, want)
				}
				return downloadResponse(request, 200, ""), nil
			})})
			body, err := fetchBody(context.Background(), "https://api.github.com/start")
			if err != nil {
				t.Fatal(err)
			}
			body.Close()
			if calls != 2 {
				t.Fatal(calls)
			}
		})
	}
}
func TestGitHubRedirectPolicyAndLimit(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "test-secret")
	t.Setenv("GH_TOKEN", "")
	t.Run("default limit", func(t *testing.T) {
		calls := 0
		useHTTPClient(t, &http.Client{Transport: transportFunc(func(request *http.Request) (*http.Response, error) {
			calls++
			return downloadResponse(request, 302, fmt.Sprintf("/redirect/%d", calls)), nil
		})})
		if _, err := fetchBody(context.Background(), "https://api.github.com/start"); err == nil || !strings.Contains(err.Error(), "stopped after 10 redirects") {
			t.Fatal(err)
		}
		if calls != 10 {
			t.Fatal(calls)
		}
	})
	t.Run("custom policy", func(t *testing.T) {
		calls, redirects := 0, 0
		client := &http.Client{Transport: transportFunc(func(request *http.Request) (*http.Response, error) {
			calls++
			if calls < 12 {
				return downloadResponse(request, 302, fmt.Sprintf("/redirect/%d", calls)), nil
			}
			return downloadResponse(request, 200, ""), nil
		}), CheckRedirect: func(request *http.Request, via []*http.Request) error { redirects++; return nil }}
		useHTTPClient(t, client)
		body, err := fetchBody(context.Background(), "https://api.github.com/start")
		if err != nil {
			t.Fatal(err)
		}
		body.Close()
		if calls != 12 || redirects != 11 {
			t.Fatal(calls, redirects)
		}
	})
	t.Run("callback cannot leak token", func(t *testing.T) {
		calls := 0
		useHTTPClient(t, &http.Client{Transport: transportFunc(func(request *http.Request) (*http.Response, error) {
			calls++
			if calls == 1 {
				return downloadResponse(request, 302, "https://api.github.com/next"), nil
			}
			if request.URL.Scheme != "http" || request.Header.Get("Authorization") != "" || request.Header["authorization"] != nil {
				t.Fatal("callback leaked credentials")
			}
			return downloadResponse(request, 200, ""), nil
		}), CheckRedirect: func(request *http.Request, via []*http.Request) error {
			request.URL.Scheme = "http"
			request.Header.Set("Authorization", "Bearer test-secret")
			request.Header["authorization"] = []string{"Bearer test-secret"}
			return nil
		}})
		body, err := fetchBody(context.Background(), "https://api.github.com/start")
		if err != nil {
			t.Fatal(err)
		}
		body.Close()
	})
	t.Run("callback sees no cross origin token", func(t *testing.T) {
		seen := false
		useHTTPClient(t, &http.Client{Transport: transportFunc(func(request *http.Request) (*http.Response, error) {
			return downloadResponse(request, 302, "https://downloads.example.com/archive"), nil
		}), CheckRedirect: func(request *http.Request, via []*http.Request) error {
			seen = true
			if request.Header.Get("Authorization") != "" {
				t.Fatal("credential visible to cross-origin callback")
			}
			return http.ErrUseLastResponse
		}})
		if _, err := fetchBody(context.Background(), "https://api.github.com/start"); err == nil || !strings.Contains(err.Error(), "302") {
			t.Fatal(err)
		}
		if !seen {
			t.Fatal("callback skipped")
		}
	})
}
func TestGitHubTokenErrorsAreRedacted(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "test-secret")
	t.Setenv("GH_TOKEN", "")
	sentinel := errors.New("transport rejected Bearer test-secret")
	useHTTPClient(t, &http.Client{Transport: transportFunc(func(request *http.Request) (*http.Response, error) { return nil, sentinel })})
	if _, err := fetchBody(context.Background(), "https://api.github.com/start"); err == nil || strings.Contains(err.Error(), "test-secret") || !errors.Is(err, sentinel) {
		t.Fatalf("unredacted or unwrapped error: %v", err)
	}
	t.Setenv("GITHUB_TOKEN", "invalid\ntest-secret")
	if _, err := fetchBody(context.Background(), "https://api.github.com/start"); err == nil || strings.Contains(err.Error(), "test-secret") {
		t.Fatalf("invalid token leaked: %v", err)
	}
}
func TestGitHubClientTimeoutPreserved(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "test-secret")
	t.Setenv("GH_TOKEN", "")
	useHTTPClient(t, &http.Client{Timeout: 20 * time.Millisecond, Transport: transportFunc(func(request *http.Request) (*http.Response, error) {
		<-request.Context().Done()
		return nil, request.Context().Err()
	})})
	if _, err := fetchBody(context.Background(), "https://api.github.com/start"); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("timeout lost: %v", err)
	}
}
func TestGitHubRateLimitDiagnostic(t *testing.T) {
	for _, authenticated := range []bool{false, true} {
		t.Run(fmt.Sprint(authenticated), func(t *testing.T) {
			token := ""
			if authenticated {
				token = "test-secret"
			}
			t.Setenv("GITHUB_TOKEN", token)
			t.Setenv("GH_TOKEN", "")
			body := &trackedBody{Reader: strings.NewReader("response must not be included")}
			useHTTPClient(t, &http.Client{Transport: transportFunc(func(request *http.Request) (*http.Response, error) {
				response := downloadResponse(request, 403, "")
				response.Header.Set("X-RateLimit-Remaining", "0")
				response.Body = body
				return response, nil
			})})
			_, err := fetchBody(context.Background(), "https://api.github.com/repos/example/releases")
			if err == nil || !strings.Contains(err.Error(), "rate limit exceeded") || strings.Contains(err.Error(), "test-secret") || strings.Contains(err.Error(), "response must") {
				t.Fatal(err)
			}
			if !authenticated && !strings.Contains(err.Error(), "GITHUB_TOKEN") {
				t.Fatal("missing authentication guidance", err)
			}
			if !body.closed {
				t.Fatal("response body leaked")
			}
		})
	}
}

func TestGitHubRedirectErrorRemainsWrappedAndRedacted(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "test-secret")
	t.Setenv("GH_TOKEN", "")
	sentinel := errors.New("redirect rejected Bearer test-secret")
	useHTTPClient(t, &http.Client{Transport: transportFunc(func(request *http.Request) (*http.Response, error) {
		return downloadResponse(request, 302, "https://api.github.com/next"), nil
	}), CheckRedirect: func(request *http.Request, via []*http.Request) error { return sentinel }})
	if _, err := fetchBody(context.Background(), "https://api.github.com/start"); err == nil || strings.Contains(err.Error(), "test-secret") || !errors.Is(err, sentinel) {
		t.Fatalf("redirect error leaked or lost identity: %v", err)
	}
}
func TestMalformedGitHubTokenDoesNotAffectOtherDownloads(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "invalid\nsecret")
	t.Setenv("GH_TOKEN", "")
	useHTTPClient(t, &http.Client{Transport: transportFunc(func(request *http.Request) (*http.Response, error) {
		if request.Header.Get("Authorization") != "" {
			t.Fatal("credential on download")
		}
		return downloadResponse(request, 200, ""), nil
	})})
	body, err := fetchBody(context.Background(), "https://go.dev/dl/archive.tar.gz")
	if err != nil {
		t.Fatal(err)
	}
	body.Close()
}
