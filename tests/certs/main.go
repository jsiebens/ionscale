// partially taken from https://github.com/FiloSottile/mkcert
package main

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/pem"
	"log"
	"math/big"
	"net"
	"net/mail"
	"net/url"
	"os"
	"path/filepath"
	"time"
)

func main() {
	path := "./tests/config"
	caCert, caKey := newCA(path)
	makeCert(path, caCert, caKey, []string{"ionscale"})
}

func generateKey(rootCA bool) (crypto.PrivateKey, error) {
	if rootCA {
		return rsa.GenerateKey(rand.Reader, 3072)
	}
	return rsa.GenerateKey(rand.Reader, 2048)
}

func makeCert(path string, caCert *x509.Certificate, caKey any, hosts []string) {
	priv, err := generateKey(false)
	fatalIfErr(err)
	pub := priv.(crypto.Signer).Public()

	expiration := time.Now().AddDate(2, 3, 0)

	tpl := &x509.Certificate{
		SerialNumber: randomSerialNumber(),
		Subject: pkix.Name{
			Organization:       []string{"ionscale tests cert"},
			OrganizationalUnit: []string{"ionscale"},
		},

		NotBefore: time.Now(), NotAfter: expiration,

		KeyUsage: x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
	}

	for _, h := range hosts {
		if ip := net.ParseIP(h); ip != nil {
			tpl.IPAddresses = append(tpl.IPAddresses, ip)
		} else if email, err := mail.ParseAddress(h); err == nil && email.Address == h {
			tpl.EmailAddresses = append(tpl.EmailAddresses, h)
		} else if uriName, err := url.Parse(h); err == nil && uriName.Scheme != "" && uriName.Host != "" {
			tpl.URIs = append(tpl.URIs, uriName)
		} else {
			tpl.DNSNames = append(tpl.DNSNames, h)
		}
	}

	if len(tpl.IPAddresses) > 0 || len(tpl.DNSNames) > 0 || len(tpl.URIs) > 0 {
		tpl.ExtKeyUsage = append(tpl.ExtKeyUsage, x509.ExtKeyUsageServerAuth)
	}
	if len(tpl.EmailAddresses) > 0 {
		tpl.ExtKeyUsage = append(tpl.ExtKeyUsage, x509.ExtKeyUsageEmailProtection)
	}

	cert, err := x509.CreateCertificate(rand.Reader, tpl, caCert, pub, caKey)
	fatalIfErr(err)

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert})
	privDER, err := x509.MarshalPKCS8PrivateKey(priv)
	fatalIfErr(err)
	privPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privDER})

	fatalIfErr(os.WriteFile(filepath.Join(path, "ionscale.pem"), certPEM, 0644))
	fatalIfErr(os.WriteFile(filepath.Join(path, "ionscale.key"), privPEM, 0600))
}

func newCA(path string) (*x509.Certificate, any) {
	priv, err := generateKey(true)
	fatalIfErr(err)
	pub := priv.(crypto.Signer).Public()

	spkiASN1, err := x509.MarshalPKIXPublicKey(pub)
	fatalIfErr(err)

	var spki struct {
		Algorithm        pkix.AlgorithmIdentifier
		SubjectPublicKey asn1.BitString
	}
	_, err = asn1.Unmarshal(spkiASN1, &spki)
	fatalIfErr(err)

	skid := sha1.Sum(spki.SubjectPublicKey.Bytes)

	tpl := &x509.Certificate{
		SerialNumber: randomSerialNumber(),
		Subject: pkix.Name{
			Organization:       []string{"ionscale tests CA"},
			OrganizationalUnit: []string{"ionscale"},
			CommonName:         "ionscale",
		},
		SubjectKeyId: skid[:],

		NotAfter:  time.Now().AddDate(10, 0, 0),
		NotBefore: time.Now(),

		KeyUsage: x509.KeyUsageCertSign,

		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLenZero:        true,
	}

	cert, err := x509.CreateCertificate(rand.Reader, tpl, tpl, pub, priv)
	fatalIfErr(err)
	fatalIfErr(os.WriteFile(filepath.Join(path, "ca.pem"), pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert}), 0644))

	return tpl, priv
}

func randomSerialNumber() *big.Int {
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	fatalIfErr(err)
	return serialNumber
}

func fatalIfErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
