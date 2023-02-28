package aws_signing_helper

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"io"
	"log"
)

type FileSystemSigner struct {
	PrivateKey crypto.PrivateKey
	cert       *x509.Certificate
	certChain  []*x509.Certificate
}

func (fileSystemSigner *FileSystemSigner) Public() crypto.PublicKey {
	return nil
}

func (fileSystemSigner *FileSystemSigner) Close() {
}

func (fileSystemSigner *FileSystemSigner) Sign(rand io.Reader, digest []byte, opts crypto.SignerOpts) (signature []byte, err error) {
	ecdsaPrivateKey, ok := fileSystemSigner.PrivateKey.(ecdsa.PrivateKey)
	if ok {
		sig, err := ecdsa.SignASN1(rand, &ecdsaPrivateKey, digest[:])
		if err == nil {
			return sig, nil
		}
	}

	rsaPrivateKey, ok := fileSystemSigner.PrivateKey.(rsa.PrivateKey)
	if ok {
		sig, err := rsa.SignPKCS1v15(rand, &rsaPrivateKey, opts.HashFunc(), digest[:])
		if err == nil {
			return sig, nil
		}
	}

	log.Println("unsupported algorithm")
	return nil, errors.New("unsupported algorithm")
}

func (fileSystemSigner *FileSystemSigner) Certificate() (*x509.Certificate, error) {
	return fileSystemSigner.cert, nil
}

func (fileSystemSigner *FileSystemSigner) CertificateChain() ([]*x509.Certificate, error) {
	return fileSystemSigner.certChain, nil
}

// Returns a FileSystemSigner, that signs a payload using the
// private key passed in
func GetFileSystemSigner(privateKey crypto.PrivateKey, certificateId string, certificateBundleId string) (signer Signer, signingAlgorithm string, err error) {
	certificateData, err := ReadCertificateData(certificateId)
	if err != nil {
		return nil, "", err
	}
	certificateDerData, err := base64.StdEncoding.DecodeString(certificateData.CertificateData)
	if err != nil {
		return nil, "", err
	}
	certificate, err := x509.ParseCertificate([]byte(certificateDerData))
	if err != nil {
		return nil, "", err
	}
	var certificateChain []*x509.Certificate
	if certificateBundleId != "" {
		certificateChainPointers, err := ReadCertificateBundleData(certificateBundleId)
		if err != nil {
			return nil, "", err
		}
		certificateChain = append(certificateChain, certificateChainPointers...)
	}

	// Find the signing algorithm
	_, isRsaKey := privateKey.(rsa.PrivateKey)
	if isRsaKey {
		signingAlgorithm = aws4_x509_rsa_sha256
	}
	_, isEcKey := privateKey.(ecdsa.PrivateKey)
	if isEcKey {
		signingAlgorithm = aws4_x509_ecdsa_sha256
	}
	if signingAlgorithm == "" {
		log.Println("unsupported algorithm")
		return nil, "", errors.New("unsupported algorithm")
	}
	return &FileSystemSigner{privateKey, certificate, certificateChain}, signingAlgorithm, nil
}
