package transport

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBuildTLSConfigVerifyDefaultsToSecure(t *testing.T) {
	cfg, err := BuildTLSConfig(TLSOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.InsecureSkipVerify {
		t.Fatal("InsecureSkipVerify = true, want false by default")
	}
}

func TestBuildTLSConfigVerifyNoSkipsVerification(t *testing.T) {
	cfg, err := BuildTLSConfig(TLSOptions{Verify: "no"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.InsecureSkipVerify {
		t.Fatal("InsecureSkipVerify = false, want true for --verify=no")
	}
}

func TestBuildTLSConfigVerifyCABundle(t *testing.T) {
	caPEM, _, _, _, _ := generateTestCertChain(t)
	f := writeTempFile(t, "ca.pem", caPEM)

	cfg, err := BuildTLSConfig(TLSOptions{Verify: f})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.RootCAs == nil {
		t.Fatal("RootCAs not set from --verify=<CA bundle path>")
	}
}

func TestBuildTLSConfigVerifyBadPathErrors(t *testing.T) {
	if _, err := BuildTLSConfig(TLSOptions{Verify: "/no/such/file.pem"}); err == nil {
		t.Fatal("expected error for unreadable --verify path")
	}
}

func TestBuildTLSConfigSSLVersionPinsRange(t *testing.T) {
	cfg, err := BuildTLSConfig(TLSOptions{SSLVersion: "tls1.2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.MinVersion != tls.VersionTLS12 || cfg.MaxVersion != tls.VersionTLS12 {
		t.Fatalf("MinVersion=%x MaxVersion=%x, want both %x", cfg.MinVersion, cfg.MaxVersion, tls.VersionTLS12)
	}
}

func TestBuildTLSConfigSSL23LeavesRangeUnset(t *testing.T) {
	cfg, err := BuildTLSConfig(TLSOptions{SSLVersion: "ssl2.3"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.MinVersion != 0 || cfg.MaxVersion != 0 {
		t.Fatalf("MinVersion=%x MaxVersion=%x, want both unset (0)", cfg.MinVersion, cfg.MaxVersion)
	}
}

func TestBuildTLSConfigSSL3Errors(t *testing.T) {
	if _, err := BuildTLSConfig(TLSOptions{SSLVersion: "ssl3"}); err == nil {
		t.Fatal("expected error: SSLv3 is not supported by Go's crypto/tls")
	}
}

func TestBuildTLSConfigUnknownSSLVersionErrors(t *testing.T) {
	if _, err := BuildTLSConfig(TLSOptions{SSLVersion: "tls9"}); err == nil {
		t.Fatal("expected error for unknown --ssl value")
	}
}

func writeTempFile(t *testing.T, name string, content []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("writing temp file: %v", err)
	}
	return path
}

// generateTestCertChain returns a self-signed CA cert (PEM), a server cert
// signed by it (PEM) with its key (PEM), and a client cert (PEM) with its
// key (PEM), all ECDSA P-256, valid for the test's lifetime only.
func generateTestCertChain(t *testing.T) (caPEM, serverCertPEM, serverKeyPEM, clientCertPEM, clientKeyPEM []byte) {
	t.Helper()

	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generating CA key: %v", err)
	}
	caTmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "htee test CA"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTmpl, caTmpl, &caKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("creating CA cert: %v", err)
	}
	caPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER})
	caCert, _ := x509.ParseCertificate(caDER)

	issue := func(cn string, eku x509.ExtKeyUsage) (certPEM, keyPEM []byte) {
		key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			t.Fatalf("generating key: %v", err)
		}
		tmpl := &x509.Certificate{
			SerialNumber: big.NewInt(2),
			Subject:      pkix.Name{CommonName: cn},
			NotBefore:    time.Now().Add(-time.Hour),
			NotAfter:     time.Now().Add(time.Hour),
			KeyUsage:     x509.KeyUsageDigitalSignature,
			ExtKeyUsage:  []x509.ExtKeyUsage{eku},
			DNSNames:     []string{"127.0.0.1", "localhost"},
			IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
		}
		der, err := x509.CreateCertificate(rand.Reader, tmpl, caCert, &key.PublicKey, caKey)
		if err != nil {
			t.Fatalf("creating cert for %s: %v", cn, err)
		}
		keyDER, err := x509.MarshalECPrivateKey(key)
		if err != nil {
			t.Fatalf("marshaling key: %v", err)
		}
		certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
		keyPEM = pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
		return
	}

	serverCertPEM, serverKeyPEM = issue("localhost", x509.ExtKeyUsageServerAuth)
	clientCertPEM, clientKeyPEM = issue("htee test client", x509.ExtKeyUsageClientAuth)
	return
}
