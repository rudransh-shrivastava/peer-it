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
	"time"

	"github.com/quic-go/quic-go"
)

const (
	alpn            = "peer-it"
	certValidityDur = 365 * 24 * time.Hour
)

func DefaultQUICConfig() *quic.Config {
	return &quic.Config{
		KeepAlivePeriod: 10 * time.Second,
		MaxIdleTimeout:  30 * time.Second,
	}
}

func DefaultTLSConfig() (*tls.Config, error) {
	cert, err := GenerateSelfSignedCert()
	if err != nil {
		return nil, err
	}

	return &tls.Config{
		Certificates:       []tls.Certificate{cert},
		InsecureSkipVerify: true,
		NextProtos:         []string{alpn},
	}, nil
}

func GenerateSelfSignedCert() (tls.Certificate, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, err
	}

	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return tls.Certificate{}, err
	}

	template := x509.Certificate{
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature,
		NotAfter:     time.Now().Add(certValidityDur),
		NotBefore:    time.Now(),
		SerialNumber: serialNumber,
		Subject:      pkix.Name{Organization: []string{"peer-it"}},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		return tls.Certificate{}, err
	}

	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return tls.Certificate{}, err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Bytes: certDER, Type: "CERTIFICATE"})
	keyPEM := pem.EncodeToMemory(&pem.Block{Bytes: keyDER, Type: "EC PRIVATE KEY"})

	return tls.X509KeyPair(certPEM, keyPEM)
}
