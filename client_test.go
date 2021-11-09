package alternative

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/rest"
)

func TestClientWithAlternateServices(t *testing.T) {
	var altSvcHeader string
	ts := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Alt-Svc", altSvcHeader)
		fmt.Fprintf(w, "Hello, %s", r.Proto)
	}))
	ts.EnableHTTP2 = true
	ts.StartTLS()
	defer ts.Close()
	tsURL, err := url.ParseRequestURI(ts.URL)
	if err != nil {
		t.Fatalf("Unexpected error %v", err)
	}

	altTs := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello alternate, %s", r.Proto)
	}))
	altTs.EnableHTTP2 = true
	altTs.StartTLS()
	altURL, err := url.ParseRequestURI(altTs.URL)
	if err != nil {
		t.Fatalf("Unexpected error %v", err)
	}
	// use localhost to check there are no certificate issues
	// the test certificate allows only example.com, 127.0.0.1 and ::1
	altSvcHeader = fmt.Sprintf(`h2="localhost:%s"`, altURL.Port())
	defer altTs.Close()

	transport, ok := ts.Client().Transport.(*http.Transport)
	if !ok {
		t.Fatalf("failed to assert *http.Transport")
	}
	transport.TLSClientConfig.ServerName = "example.com"
	fn := func(rt http.RoundTripper) http.RoundTripper {
		return NewAlternativeServiceRoundTripperWithOptions(rt,
			WithAlternativeServerName("example.com"),
			WithLocalhostAllowed(),
		)
	}
	config := &rest.Config{
		Host: fmt.Sprintf("https://localhost:%s", tsURL.Port()),
		// These fields are required to create a REST client.
		ContentConfig: rest.ContentConfig{
			GroupVersion:         &schema.GroupVersion{},
			NegotiatedSerializer: &serializer.CodecFactory{},
		},
		Transport:     transport,
		WrapTransport: fn,
	}
	client, err := rest.RESTClientFor(config)
	if err != nil {
		t.Fatalf("failed to create REST client: %v", err)
	}
	data, err := client.Get().AbsPath("/").DoRaw(context.TODO())
	if err != nil {
		t.Fatalf("unexpected err: %s: %v", data, err)
	}
	if string(data) != "Hello, HTTP/2.0" {
		t.Fatalf("unexpected response: %s", data)
	}
	time.Sleep(1 * time.Second)
	data, err = client.Get().AbsPath("/").DoRaw(context.TODO())
	if err != nil {
		t.Fatalf("unexpected err: %s: %v", data, err)
	}
	if string(data) != "Hello alternate, HTTP/2.0" {
		t.Fatalf("unexpected response: %s", data)
	}
}

// If the TLS client ServerName is not set the alternative service will fail because of a certificate error
func TestClientWithAlternateServicesFallBack(t *testing.T) {
	var altSvcHeader string
	ts := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Alt-Svc", altSvcHeader)
		fmt.Fprintf(w, "Hello, %s", r.Proto)
	}))
	ts.EnableHTTP2 = true
	ts.StartTLS()
	defer ts.Close()
	tsURL, err := url.ParseRequestURI(ts.URL)
	if err != nil {
		t.Fatalf("Unexpected error %v", err)
	}

	altTs := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello alternate, %s", r.Proto)
	}))
	altTs.EnableHTTP2 = true
	altTs.StartTLS()
	altURL, err := url.ParseRequestURI(altTs.URL)
	if err != nil {
		t.Fatalf("Unexpected error %v", err)
	}
	// use localhost to check there are no certificate issues
	// the test certificate allows only example.com, 127.0.0.1 and ::1
	altSvcHeader = fmt.Sprintf(`h2="localhost:%s"`, altURL.Port())
	defer altTs.Close()

	transport, ok := ts.Client().Transport.(*http.Transport)
	if !ok {
		t.Fatalf("failed to assert *http.Transport")
	}
	fn := func(rt http.RoundTripper) http.RoundTripper {
		return NewAlternativeServiceRoundTripperWithOptions(rt,
			WithLocalhostAllowed(),
		)
	}
	config := &rest.Config{
		Host: fmt.Sprintf("https://127.0.0.1:%s", tsURL.Port()),
		// These fields are required to create a REST client.
		ContentConfig: rest.ContentConfig{
			GroupVersion:         &schema.GroupVersion{},
			NegotiatedSerializer: &serializer.CodecFactory{},
		},
		Transport:     transport,
		WrapTransport: fn,
	}
	client, err := rest.RESTClientFor(config)
	if err != nil {
		t.Fatalf("failed to create REST client: %v", err)
	}
	data, err := client.Get().AbsPath("/").DoRaw(context.TODO())
	if err != nil {
		t.Fatalf("unexpected err: %s: %v", data, err)
	}
	if string(data) != "Hello, HTTP/2.0" {
		t.Fatalf("unexpected response: %s", data)
	}
	time.Sleep(1 * time.Second)
	data, err = client.Get().AbsPath("/").DoRaw(context.TODO())
	if err != nil {
		t.Fatalf("expected err: %s: %v", data, err)
	}
	if string(data) != "Hello, HTTP/2.0" {
		t.Fatalf("unexpected response: %s", data)
	}
	data, err = client.Get().AbsPath("/").DoRaw(context.TODO())
	if err != nil {
		t.Fatalf("unexpected err: %s: %v", data, err)
	}
	if string(data) != "Hello, HTTP/2.0" {
		t.Fatalf("unexpected response: %s", data)
	}
}
