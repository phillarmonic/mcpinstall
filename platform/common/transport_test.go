package common

import "testing"

func TestIsHTTPInfersFromTheURL(t *testing.T) {
	cases := []struct {
		name string
		opts Options
		want bool
	}{
		{"empty is stdio", Options{}, false},
		{"url is http", Options{URL: "https://example.test/mcp"}, true},
		{"explicit stdio wins over a url", Options{Transport: TransportStdio, URL: "https://example.test/mcp"}, false},
		{"explicit http needs no url to answer", Options{Transport: TransportHTTP}, true},
		{"remote alias", Options{Transport: "remote"}, true},
		{"local alias", Options{Transport: "local", URL: "https://example.test/mcp"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsHTTP(tc.opts); got != tc.want {
				t.Fatalf("IsHTTP(%#v) = %v, want %v", tc.opts, got, tc.want)
			}
		})
	}
}

func TestValidateRequiresTheIdentityTheTransportNeeds(t *testing.T) {
	cases := []struct {
		name string
		opts Options
		ok   bool
	}{
		{"stdio with a command", Options{ServerName: "srv", Command: "srv-mcp"}, true},
		{"stdio without a command", Options{ServerName: "srv"}, false},
		{"http with a url", Options{ServerName: "srv", URL: "https://example.test/mcp"}, true},
		{"http without a url", Options{ServerName: "srv", Transport: TransportHTTP}, false},
		{"missing name", Options{Command: "srv-mcp"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := Validate(tc.opts)
			if tc.ok && err != nil {
				t.Fatalf("Validate = %v, want nil", err)
			}
			if !tc.ok && err == nil {
				t.Fatal("Validate = nil, want an error")
			}
		})
	}
}

func TestHeadersMapCopiesAndOmitsEmpty(t *testing.T) {
	if HeadersMap(Options{}) != nil {
		t.Fatal("an entry with no headers should not gain an empty object")
	}
	headers := HeadersMap(Options{Headers: map[string]string{"Authorization": "Bearer t"}})
	if headers["Authorization"] != "Bearer t" {
		t.Fatalf("headers = %#v", headers)
	}
}
