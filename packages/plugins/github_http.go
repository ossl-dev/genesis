package plugins

import (
	"errors"
	"net/http"
	"net/url"
	"os"
	"strings"
)

func githubAPIOrigin(target *url.URL) bool {
	return target != nil && target.Scheme == "https" && strings.EqualFold(target.Hostname(), "api.github.com") && (target.Port() == "" || target.Port() == "443") && target.User == nil
}

func githubHTTPClient(request *http.Request) (*http.Client, string, error) {
	client := HTTPClient
	if !githubAPIOrigin(request.URL) {
		return client, "", nil
	}
	token := strings.TrimSpace(os.Getenv("GITHUB_TOKEN"))
	if token == "" {
		token = strings.TrimSpace(os.Getenv("GH_TOKEN"))
	}
	if token == "" {
		return client, "", nil
	}
	for _, character := range token {
		if character <= ' ' || character >= 127 {
			return nil, "", errors.New("invalid GitHub API token format")
		}
	}
	request.Header.Set("Authorization", "Bearer "+token)
	cloned := *client
	redirect := client.CheckRedirect
	cloned.CheckRedirect = func(next *http.Request, via []*http.Request) error {
		if !githubAPIOrigin(next.URL) {
			stripAuthorization(next.Header)
		}
		if redirect != nil {
			err := redirect(next, via)
			if !githubAPIOrigin(next.URL) {
				stripAuthorization(next.Header)
			}
			return err
		}
		if len(via) >= 10 {
			return errors.New("stopped after 10 redirects")
		}
		return nil
	}
	return &cloned, token, nil
}

func stripAuthorization(header http.Header) {
	for name := range header {
		if strings.EqualFold(name, "Authorization") {
			delete(header, name)
		}
	}
}

type redactedHTTPError struct {
	cause   error
	message string
}

func (e *redactedHTTPError) Error() string { return e.message }
func (e *redactedHTTPError) Unwrap() error { return e.cause }
func redactHTTPError(err error, token string) error {
	if token != "" && strings.Contains(err.Error(), token) {
		return &redactedHTTPError{cause: err, message: strings.ReplaceAll(err.Error(), token, "[redacted]")}
	}
	return err
}
