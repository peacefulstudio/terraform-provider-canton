// Copyright (c) 2026 Peaceful Studio OÜ
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"net"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"golang.org/x/oauth2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func TestNew(t *testing.T) {
	factory := New("1.0.0")
	p := factory()
	if p == nil {
		t.Fatal("expected non-nil provider")
	}
}

func TestProviderMetadata(t *testing.T) {
	p := &cantonProvider{version: "1.2.3"}
	resp := &provider.MetadataResponse{}
	p.Metadata(context.Background(), provider.MetadataRequest{}, resp)

	if resp.TypeName != "canton" {
		t.Errorf("expected type name 'canton', got %q", resp.TypeName)
	}
	if resp.Version != "1.2.3" {
		t.Errorf("expected version '1.2.3', got %q", resp.Version)
	}
}

func TestProviderSchema(t *testing.T) {
	p := &cantonProvider{}
	resp := &provider.SchemaResponse{}
	p.Schema(context.Background(), provider.SchemaRequest{}, resp)

	if resp.Schema.Description == "" {
		t.Error("expected non-empty schema description")
	}

	attrs := resp.Schema.Attributes
	if _, ok := attrs["participant_url"]; !ok {
		t.Error("expected participant_url attribute")
	}
	if _, ok := attrs["oauth2"]; !ok {
		t.Error("expected oauth2 attribute")
	}
}

func TestProviderResources(t *testing.T) {
	p := &cantonProvider{}
	resources := p.Resources(context.Background())

	if len(resources) != 3 {
		t.Fatalf("expected 3 resource factories, got %d", len(resources))
	}

	// Verify each factory produces a non-nil resource.
	for i, factory := range resources {
		r := factory()
		if r == nil {
			t.Errorf("resource factory %d returned nil", i)
		}
	}
}

func TestProviderDataSources(t *testing.T) {
	p := &cantonProvider{}
	dataSources := p.DataSources(context.Background())

	if len(dataSources) != 3 {
		t.Fatalf("expected 3 data source factories, got %d", len(dataSources))
	}

	for i, factory := range dataSources {
		ds := factory()
		if ds == nil {
			t.Errorf("data source factory %d returned nil", i)
		}
	}
}

func TestProviderDataSources_TypeNames(t *testing.T) {
	p := &cantonProvider{}
	dataSources := p.DataSources(context.Background())

	expectedTypes := map[string]bool{
		"canton_party":   false,
		"canton_user":    false,
		"canton_parties": false,
	}

	for _, factory := range dataSources {
		ds := factory()
		metaResp := &datasource.MetadataResponse{}
		ds.Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "canton"}, metaResp)
		if _, ok := expectedTypes[metaResp.TypeName]; !ok {
			t.Errorf("unexpected data source type: %s", metaResp.TypeName)
		}
		if expectedTypes[metaResp.TypeName] {
			t.Errorf("duplicate data source type: %s", metaResp.TypeName)
		}
		expectedTypes[metaResp.TypeName] = true
	}

	for name, found := range expectedTypes {
		if !found {
			t.Errorf("expected data source type %s not found", name)
		}
	}
}

func TestProviderResources_TypeNames(t *testing.T) {
	p := &cantonProvider{}
	resources := p.Resources(context.Background())

	expectedTypes := map[string]bool{
		"canton_party":       false,
		"canton_user":        false,
		"canton_user_rights": false,
	}

	for _, factory := range resources {
		r := factory()
		metaResp := &resource.MetadataResponse{}
		r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "canton"}, metaResp)
		if _, ok := expectedTypes[metaResp.TypeName]; !ok {
			t.Errorf("unexpected resource type: %s", metaResp.TypeName)
		}
		if expectedTypes[metaResp.TypeName] {
			t.Errorf("duplicate resource type: %s", metaResp.TypeName)
		}
		expectedTypes[metaResp.TypeName] = true
	}

	for name, found := range expectedTypes {
		if !found {
			t.Errorf("expected resource type %s not found", name)
		}
	}
}

func TestProviderDataSources_ReturnsNonNilSlice(t *testing.T) {
	p := &cantonProvider{}
	ds := p.DataSources(context.Background())
	if ds == nil {
		t.Error("expected non-nil slice (even if empty)")
	}
}

func TestNormalizeParticipantURL(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "https scheme", raw: "https://participant.example.com:443", want: "participant.example.com:443"},
		{name: "http scheme", raw: "http://localhost:3901", want: "localhost:3901"},
		{name: "no scheme", raw: "localhost:3901", want: "localhost:3901"},
		{name: "https with trailing slash", raw: "https://host:443/", want: "host:443"},
		{name: "bare host trailing slash", raw: "host:443/", want: "host:443"},
		{name: "empty", raw: "", want: ""},
		{name: "double scheme", raw: "https://https://double", want: "https://double"},
		{name: "uppercase https scheme", raw: "HTTPS://Host:443", want: "Host:443"},
		{name: "mixed case http scheme", raw: "HtTp://localhost:3901", want: "localhost:3901"},
		{name: "whitespace around url", raw: "  https://host:443  ", want: "host:443"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeParticipantURL(tt.raw)
			if got != tt.want {
				t.Errorf("normalizeParticipantURL(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}

func TestParticipantScheme(t *testing.T) {
	tests := []struct {
		name         string
		raw          string
		wantUseTLS   bool
		wantExplicit bool
	}{
		{name: "https", raw: "https://host:443", wantUseTLS: true, wantExplicit: true},
		{name: "http", raw: "http://host:3901", wantUseTLS: false, wantExplicit: true},
		{name: "no scheme", raw: "host:3901", wantUseTLS: false, wantExplicit: false},
		{name: "empty", raw: "", wantUseTLS: false, wantExplicit: false},
		{name: "https uppercase", raw: "HTTPS://host", wantUseTLS: true, wantExplicit: true},
		{name: "http uppercase", raw: "HTTP://host", wantUseTLS: false, wantExplicit: true},
		{name: "https mixed case", raw: "HtTpS://host", wantUseTLS: true, wantExplicit: true},
		{name: "whitespace", raw: "  https://host  ", wantUseTLS: true, wantExplicit: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			useTLS, explicit := participantScheme(tt.raw)
			if useTLS != tt.wantUseTLS || explicit != tt.wantExplicit {
				t.Errorf("participantScheme(%q) = (%v, %v), want (%v, %v)",
					tt.raw, useTLS, explicit, tt.wantUseTLS, tt.wantExplicit)
			}
		})
	}
}

// recordingTokenSource counts Token() calls and returns a canned token each time.
type recordingTokenSource struct {
	token *oauth2.Token
	err   error
	calls atomic.Int64
}

func (r *recordingTokenSource) Token() (*oauth2.Token, error) {
	r.calls.Add(1)
	if r.err != nil {
		return nil, r.err
	}
	return r.token, nil
}

func TestTokenUnaryInterceptor_InjectsBearer(t *testing.T) {
	ts := &recordingTokenSource{token: &oauth2.Token{AccessToken: "s3cret"}}
	interceptor := tokenUnaryInterceptor(ts)

	var gotAuth []string
	invoker := func(ctx context.Context, _ string, _, _ interface{}, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
		md, _ := metadata.FromOutgoingContext(ctx)
		gotAuth = md.Get("authorization")
		return nil
	}

	if err := interceptor(context.Background(), "/svc/Method", nil, nil, nil, invoker); err != nil {
		t.Fatalf("interceptor returned error: %v", err)
	}
	if ts.calls.Load() != 1 {
		t.Errorf("expected 1 Token() call, got %d", ts.calls.Load())
	}
	if len(gotAuth) != 1 || gotAuth[0] != "Bearer s3cret" {
		t.Errorf("expected authorization=[Bearer s3cret], got %v", gotAuth)
	}
}

func TestTokenUnaryInterceptor_PropagatesTokenError(t *testing.T) {
	ts := &recordingTokenSource{err: errors.New("token endpoint down")}
	interceptor := tokenUnaryInterceptor(ts)

	invokerCalled := false
	invoker := func(context.Context, string, interface{}, interface{}, *grpc.ClientConn, ...grpc.CallOption) error {
		invokerCalled = true
		return nil
	}

	err := interceptor(context.Background(), "/svc/Method", nil, nil, nil, invoker)
	if err == nil || !strings.Contains(err.Error(), "token endpoint down") {
		t.Errorf("expected token error to propagate, got %v", err)
	}
	if invokerCalled {
		t.Error("invoker should not be called when token fetch fails")
	}
}

func TestTokenStreamInterceptor_InjectsBearer(t *testing.T) {
	ts := &recordingTokenSource{token: &oauth2.Token{AccessToken: "s3cret"}}
	interceptor := tokenStreamInterceptor(ts)

	var gotAuth []string
	streamer := func(ctx context.Context, _ *grpc.StreamDesc, _ *grpc.ClientConn, _ string, _ ...grpc.CallOption) (grpc.ClientStream, error) {
		md, _ := metadata.FromOutgoingContext(ctx)
		gotAuth = md.Get("authorization")
		return nil, nil
	}

	if _, err := interceptor(context.Background(), &grpc.StreamDesc{}, nil, "/svc/Method", streamer); err != nil {
		t.Fatalf("interceptor returned error: %v", err)
	}
	if len(gotAuth) != 1 || gotAuth[0] != "Bearer s3cret" {
		t.Errorf("expected authorization=[Bearer s3cret], got %v", gotAuth)
	}
}

func TestTokenUnaryInterceptor_RejectsNilToken(t *testing.T) {
	// A pathological TokenSource that returns (nil, nil) would otherwise
	// produce a "Bearer " header and a cryptic downstream auth error.
	ts := &recordingTokenSource{token: nil}
	interceptor := tokenUnaryInterceptor(ts)

	invokerCalled := false
	invoker := func(context.Context, string, interface{}, interface{}, *grpc.ClientConn, ...grpc.CallOption) error {
		invokerCalled = true
		return nil
	}

	err := interceptor(context.Background(), "/svc/Method", nil, nil, nil, invoker)
	if err == nil || !strings.Contains(err.Error(), "nil token") {
		t.Errorf("expected nil-token error, got %v", err)
	}
	if invokerCalled {
		t.Error("invoker should not be called when token is nil")
	}
}

func TestTokenUnaryInterceptor_RejectsEmptyAccessToken(t *testing.T) {
	ts := &recordingTokenSource{token: &oauth2.Token{AccessToken: "   "}}
	interceptor := tokenUnaryInterceptor(ts)

	invokerCalled := false
	invoker := func(context.Context, string, interface{}, interface{}, *grpc.ClientConn, ...grpc.CallOption) error {
		invokerCalled = true
		return nil
	}

	err := interceptor(context.Background(), "/svc/Method", nil, nil, nil, invoker)
	if err == nil || !strings.Contains(err.Error(), "empty access token") {
		t.Errorf("expected empty-access-token error, got %v", err)
	}
	if invokerCalled {
		t.Error("invoker should not be called when access token is empty")
	}
}

func TestTokenStreamInterceptor_PropagatesTokenError(t *testing.T) {
	ts := &recordingTokenSource{err: errors.New("refresh failed")}
	interceptor := tokenStreamInterceptor(ts)

	streamerCalled := false
	streamer := func(context.Context, *grpc.StreamDesc, *grpc.ClientConn, string, ...grpc.CallOption) (grpc.ClientStream, error) {
		streamerCalled = true
		return nil, nil
	}

	_, err := interceptor(context.Background(), &grpc.StreamDesc{}, nil, "/svc/Method", streamer)
	if err == nil || !strings.Contains(err.Error(), "refresh failed") {
		t.Errorf("expected refresh error to propagate, got %v", err)
	}
	if streamerCalled {
		t.Error("streamer should not be called when token fetch fails")
	}
}

// newTLSServerWithALPN stands up a raw TLS listener advertising the given
// ALPN protocols. Returns the host:port and a cleanup function. We avoid
// httptest because it forces "http/1.1" into NextProtos and Go's TLS server
// strictly rejects unmatched ALPN — we need a server that CAN simulate
// "client requests h2, server offers nothing back" (real Envoy/Qovery case).
func newTLSServerWithALPN(t *testing.T, nextProtos []string) (string, func()) {
	t.Helper()
	cert := generateSelfSignedCert(t)
	cfg := &tls.Config{Certificates: []tls.Certificate{cert}, NextProtos: nextProtos}
	ln, err := tls.Listen("tcp", "127.0.0.1:0", cfg)
	if err != nil {
		t.Fatalf("tls listen: %v", err)
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				_ = c.(*tls.Conn).Handshake()
				_ = c.Close()
			}(conn)
		}
	}()
	return ln.Addr().String(), func() {
		_ = ln.Close()
		<-done
	}
}

func generateSelfSignedCert(t *testing.T) tls.Certificate {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "127.0.0.1"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create cert: %v", err)
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		t.Fatalf("x509 keypair: %v", err)
	}
	return cert
}

func TestTLSDialer_AcceptsEmptyALPN(t *testing.T) {
	// An Envoy-style proxy that doesn't advertise h2 back — the whole reason
	// this dialer exists. Handshake should succeed.
	addr, cleanup := newTLSServerWithALPN(t, nil)
	defer cleanup()

	dialer := tlsDialer(&tls.Config{
		NextProtos:         []string{"h2"},
		InsecureSkipVerify: true, // httptest cert is self-signed
		MinVersion:         tls.VersionTLS12,
	})
	conn, err := dialer(context.Background(), addr)
	if err != nil {
		t.Fatalf("dialer failed for server with empty ALPN: %v", err)
	}
	_ = conn.Close()
}

func TestTLSDialer_AcceptsH2ALPN(t *testing.T) {
	addr, cleanup := newTLSServerWithALPN(t, []string{"h2"})
	defer cleanup()

	dialer := tlsDialer(&tls.Config{
		NextProtos:         []string{"h2"},
		InsecureSkipVerify: true,
		MinVersion:         tls.VersionTLS12,
	})
	conn, err := dialer(context.Background(), addr)
	if err != nil {
		t.Fatalf("dialer failed for h2-advertising server: %v", err)
	}
	_ = conn.Close()
}

func TestTLSDialer_RejectsNonH2ALPN(t *testing.T) {
	// Simulate a TLS-terminating HTTP/1.1 proxy that actually completes
	// negotiation on "http/1.1". To reach the dialer's post-handshake check
	// (rather than stdlib's own ALPN-mismatch alert), the client must offer
	// both protocols; in production we only offer h2, so this path is
	// defense-in-depth against a future expansion.
	addr, cleanup := newTLSServerWithALPN(t, []string{"http/1.1"})
	defer cleanup()

	dialer := tlsDialer(&tls.Config{
		NextProtos:         []string{"h2", "http/1.1"},
		InsecureSkipVerify: true,
		MinVersion:         tls.VersionTLS12,
	})
	_, err := dialer(context.Background(), addr)
	if err == nil {
		t.Fatal("expected error for server negotiating http/1.1 ALPN, got nil")
	}
	if !strings.Contains(err.Error(), "http/1.1") {
		t.Errorf("expected error to mention negotiated ALPN, got %v", err)
	}
}

func TestTLSDialer_HandshakeAlertOnALPNMismatch(t *testing.T) {
	// Stdlib server sends no_application_protocol alert when the client
	// offers ["h2"] and the server only offers ["http/1.1"]. The dialer
	// surfaces this as a clear dial error rather than a silent success.
	addr, cleanup := newTLSServerWithALPN(t, []string{"http/1.1"})
	defer cleanup()

	dialer := tlsDialer(&tls.Config{
		NextProtos:         []string{"h2"},
		InsecureSkipVerify: true,
		MinVersion:         tls.VersionTLS12,
	})
	_, err := dialer(context.Background(), addr)
	if err == nil {
		t.Fatal("expected TLS alert error, got nil")
	}
	if !strings.Contains(err.Error(), "tls dial") {
		t.Errorf("expected wrapped dial error, got %v", err)
	}
}

func TestTLSDialer_DialError(t *testing.T) {
	// Pick a port that isn't listening. Use a local listener we immediately close
	// to get a guaranteed-dead port.
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := l.Addr().String()
	_ = l.Close()

	dialer := tlsDialer(&tls.Config{NextProtos: []string{"h2"}, MinVersion: tls.VersionTLS12})
	_, err = dialer(context.Background(), addr)
	if err == nil {
		t.Fatal("expected dial error against closed port, got nil")
	}
	if !strings.Contains(err.Error(), "tls dial") {
		t.Errorf("expected wrapped dial error, got %v", err)
	}
}

func TestStringValueOrEnv(t *testing.T) {
	tests := []struct {
		name     string
		value    types.String
		envVar   string
		envValue string
		want     string
	}{
		{
			name:   "uses value when set",
			value:  types.StringValue("from-config"),
			envVar: "TEST_UNUSED_VAR",
			want:   "from-config",
		},
		{
			name:     "falls back to env var",
			value:    types.StringNull(),
			envVar:   "TEST_STRING_VALUE_OR_ENV",
			envValue: "from-env",
			want:     "from-env",
		},
		{
			name:   "returns empty when neither set",
			value:  types.StringNull(),
			envVar: "TEST_NONEXISTENT_VAR_12345",
			want:   "",
		},
		{
			name:   "unknown value falls back to env",
			value:  types.StringUnknown(),
			envVar: "TEST_STRING_VALUE_UNKNOWN",
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				t.Setenv(tt.envVar, tt.envValue)
			}
			got := stringValueOrEnv(tt.value, tt.envVar)
			if got != tt.want {
				t.Errorf("stringValueOrEnv() = %q, want %q", got, tt.want)
			}
		})
	}
}
