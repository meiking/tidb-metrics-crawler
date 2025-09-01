package prometheus

import (
	"net/http"
)

// createAuthTransport creates an HTTP transport with basic authentication if credentials are provided
func createAuthTransport(username, password string) http.RoundTripper {
	transport := http.DefaultTransport

	if username != "" && password != "" {
		return &authTransport{
			username:  username,
			password:  password,
			transport: transport,
		}
	}

	return transport
}

// authTransport handles basic authentication for HTTP requests
type authTransport struct {
	username  string
	password  string
	transport http.RoundTripper
}

// RoundTrip implements the http.RoundTripper interface
func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Create a copy of the request to avoid modifying the original
	reqCopy := new(http.Request)
	*reqCopy = *req
	reqCopy.Header = make(http.Header, len(req.Header))
	for k, s := range req.Header {
		reqCopy.Header[k] = append([]string(nil), s...)
	}

	// Add basic authentication header if credentials are set
	if t.username != "" && t.password != "" {
		reqCopy.SetBasicAuth(t.username, t.password)
	}

	// Handle Bearer token authentication if needed in the future
	// if t.token != "" {
	// 	reqCopy.Header.Set("Authorization", "Bearer "+t.token)
	// }

	return t.transport.RoundTrip(reqCopy)
}
