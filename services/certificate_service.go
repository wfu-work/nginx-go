package services

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"nginx-go/domains"
	"os"

	commonServices "github.com/wfu-work/nav-common-go-lib/services"
	commonUtils "github.com/wfu-work/nav-common-go-lib/utils"
)

type CertificateService struct {
	commonServices.CrudService[domains.Certificate]
}

// List returns paginated certificate records.
func (s CertificateService) List(params map[string]string) (interface{}, int64, error) {
	return s.CrudService.List(commonUtils.ToPageInfo(params), "name,serverName,issuer")
}

// Create stores a certificate and derives issuer/validity metadata when the cert file is readable.
func (s CertificateService) Create(cert domains.Certificate) error {
	_ = fillCertificateMeta(&cert)
	return s.CrudService.Create(cert)
}

// Update modifies one certificate and refreshes parsed metadata when possible.
func (s CertificateService) Update(guid string, cert domains.Certificate) error {
	if guid == "" {
		return errors.New("missing certificate guid")
	}
	cert.Guid = guid
	_ = fillCertificateMeta(&cert)
	return s.CrudService.Updates(cert)
}

// Delete soft-deletes one certificate by guid.
func (s CertificateService) Delete(guid string) error {
	if guid == "" {
		return errors.New("missing certificate guid")
	}
	return s.CrudService.DeleteByGuid(guid)
}

// Get returns one certificate by guid.
func (s CertificateService) Get(guid string) (*domains.Certificate, error) {
	if guid == "" {
		return nil, errors.New("missing certificate guid")
	}
	cert, err := s.GetByGuid(guid)
	if err != nil {
		return nil, err
	}
	if cert == nil {
		return nil, errors.New("certificate not found")
	}
	return cert, nil
}

func fillCertificateMeta(cert *domains.Certificate) error {
	if cert.CertPath == "" {
		return nil
	}
	content, err := os.ReadFile(cert.CertPath)
	if err != nil {
		return err
	}
	block, _ := pem.Decode(content)
	if block == nil {
		return errors.New("invalid pem certificate")
	}
	parsed, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return err
	}
	if cert.ServerName == "" && len(parsed.DNSNames) > 0 {
		cert.ServerName = parsed.DNSNames[0]
	}
	cert.Issuer = parsed.Issuer.CommonName
	cert.NotBefore = parsed.NotBefore.UnixMilli()
	cert.NotAfter = parsed.NotAfter.UnixMilli()
	return nil
}
