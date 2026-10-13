package localtls

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var DefaultHosts = []string{
	"api.app.pokemonchampions.jp",
	"dev-api.app.pokemonchampions.jp",
	"stg-api.app.pokemonchampions.jp",
	"ws.app.pokemonchampions.jp",
	"localhost",
}

func serial() *big.Int {
	limit := new(big.Int).Lsh(big.NewInt(1), 128)
	s, err := rand.Int(rand.Reader, limit)
	if err != nil {
		return big.NewInt(time.Now().UnixNano())
	}
	return s
}

// EnsureCerts generates a local Root CA and default Server certificate in certDir if not already present.
func EnsureCerts(certDir string, extraIPs []string) (caCertPath, caKeyPath, srvCertPath, srvKeyPath string, err error) {
	if err := os.MkdirAll(certDir, 0755); err != nil {
		return "", "", "", "", fmt.Errorf("mkdir %s: %w", certDir, err)
	}
	caCertPath = filepath.Join(certDir, "ca.crt")
	caKeyPath = filepath.Join(certDir, "ca.key")
	srvCertPath = filepath.Join(certDir, "server.crt")
	srvKeyPath = filepath.Join(certDir, "server.key")

	if _, err := os.Stat(caCertPath); os.IsNotExist(err) {
		caPriv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			return "", "", "", "", err
		}
		now := time.Now().UTC()
		caTpl := &x509.Certificate{
			SerialNumber: serial(),
			Subject: pkix.Name{
				CommonName:   "MyPokeChamp Local Root CA",
				Organization: []string{"MyPokeChamp Private Server"},
			},
			NotBefore:             now.Add(-24 * time.Hour),
			NotAfter:              now.AddDate(10, 0, 0),
			KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign | x509.KeyUsageDigitalSignature,
			BasicConstraintsValid: true,
			IsCA:                  true,
			MaxPathLen:            1,
		}
		caDER, err := x509.CreateCertificate(rand.Reader, caTpl, caTpl, &caPriv.PublicKey, caPriv)
		if err != nil {
			return "", "", "", "", err
		}
		if err := writePEM(caCertPath, "CERTIFICATE", caDER); err != nil {
			return "", "", "", "", err
		}
		caKeyDER, err := x509.MarshalECPrivateKey(caPriv)
		if err != nil {
			return "", "", "", "", err
		}
		if err := writePEM(caKeyPath, "EC PRIVATE KEY", caKeyDER); err != nil {
			return "", "", "", "", err
		}
	}

	if _, err := os.Stat(srvCertPath); os.IsNotExist(err) {
		caPair, err := tls.LoadX509KeyPair(caCertPath, caKeyPath)
		if err != nil {
			return "", "", "", "", err
		}
		caCert, err := x509.ParseCertificate(caPair.Certificate[0])
		if err != nil {
			return "", "", "", "", err
		}
		srvPriv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			return "", "", "", "", err
		}
		ips := collectIPs(extraIPs)
		now := time.Now().UTC()
		srvTpl := &x509.Certificate{
			SerialNumber: serial(),
			Subject: pkix.Name{
				CommonName:   "api.app.pokemonchampions.jp",
				Organization: []string{"MyPokeChamp Private Server"},
			},
			DNSNames:    DefaultHosts,
			IPAddresses: ips,
			NotBefore:   now.Add(-24 * time.Hour),
			NotAfter:    now.AddDate(5, 0, 0),
			KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
			ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		}
		srvDER, err := x509.CreateCertificate(rand.Reader, srvTpl, caCert, &srvPriv.PublicKey, caPair.PrivateKey)
		if err != nil {
			return "", "", "", "", err
		}
		if err := writePEM(srvCertPath, "CERTIFICATE", srvDER); err != nil {
			return "", "", "", "", err
		}
		srvKeyDER, err := x509.MarshalECPrivateKey(srvPriv)
		if err != nil {
			return "", "", "", "", err
		}
		if err := writePEM(srvKeyPath, "EC PRIVATE KEY", srvKeyDER); err != nil {
			return "", "", "", "", err
		}
	}

	return caCertPath, caKeyPath, srvCertPath, srvKeyPath, nil
}

func writePEM(path, typ string, der []byte) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return pem.Encode(f, &pem.Block{Type: typ, Bytes: der})
}

func collectIPs(extra []string) []net.IP {
	ips := []net.IP{
		net.ParseIP("127.0.0.1"),
		net.ParseIP("10.0.2.2"),
		net.ParseIP("164.152.40.114"),
	}
	for _, s := range extra {
		if ip := net.ParseIP(s); ip != nil {
			ips = append(ips, ip)
		}
	}
	if ifaces, err := net.Interfaces(); err == nil {
		for _, iface := range ifaces {
			if addrs, err := iface.Addrs(); err == nil {
				for _, addr := range addrs {
					if ipNet, ok := addr.(*net.IPNet); ok {
						ips = append(ips, ipNet.IP)
					}
				}
			}
		}
	}
	return ips
}

// DynamicTLSConfig returns a *tls.Config that dynamically signs leaf certificates matching the client's SNI hostname.
func DynamicTLSConfig(certDir string, extraIPs []string) (*tls.Config, error) {
	caCertPath, caKeyPath, srvCertPath, srvKeyPath, err := EnsureCerts(certDir, extraIPs)
	if err != nil {
		return nil, err
	}
	caPair, err := tls.LoadX509KeyPair(caCertPath, caKeyPath)
	if err != nil {
		return nil, err
	}
	caCert, err := x509.ParseCertificate(caPair.Certificate[0])
	if err != nil {
		return nil, err
	}
	defaultPair, err := tls.LoadX509KeyPair(srvCertPath, srvKeyPath)
	if err != nil {
		return nil, err
	}
	signer, ok := defaultPair.PrivateKey.(crypto.Signer)
	if !ok {
		return nil, fmt.Errorf("server private key does not implement crypto.Signer")
	}

	ips := collectIPs(extraIPs)
	var mu sync.RWMutex
	cache := make(map[string]*tls.Certificate)

	createCert := func(cn string) (*tls.Certificate, error) {
		now := time.Now().UTC()
		dnsNames := append([]string(nil), DefaultHosts...)
		found := false
		for _, d := range dnsNames {
			if d == cn {
				found = true
				break
			}
		}
		if !found && cn != "" {
			dnsNames = append(dnsNames, cn)
		}
		tpl := &x509.Certificate{
			SerialNumber: serial(),
			Subject: pkix.Name{
				CommonName:   cn,
				Organization: []string{"MyPokeChamp Private Server"},
			},
			DNSNames:    dnsNames,
			IPAddresses: ips,
			NotBefore:   now.Add(-time.Hour),
			NotAfter:    now.AddDate(2, 0, 0),
			KeyUsage:    x509.KeyUsageDigitalSignature,
			ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		}
		der, err := x509.CreateCertificate(rand.Reader, tpl, caCert, signer.Public(), caPair.PrivateKey)
		if err != nil {
			return nil, err
		}
		return &tls.Certificate{
			Certificate: [][]byte{der, caPair.Certificate[0]},
			PrivateKey:  defaultPair.PrivateKey,
		}, nil
	}

	for _, h := range DefaultHosts {
		if c, err := createCert(h); err == nil {
			cache[h] = c
		}
	}
	defaultCert := cache[DefaultHosts[0]]
	if defaultCert == nil {
		defaultCert = &defaultPair
	}

	return &tls.Config{
		GetCertificate: func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
			sni := hello.ServerName
			if sni == "" {
				return defaultCert, nil
			}
			mu.RLock()
			c, ok := cache[sni]
			mu.RUnlock()
			if ok {
				return c, nil
			}
			mu.Lock()
			defer mu.Unlock()
			if c, ok := cache[sni]; ok {
				return c, nil
			}
			newCert, err := createCert(sni)
			if err != nil {
				return defaultCert, nil
			}
			cache[sni] = newCert
			return newCert, nil
		},
	}, nil
}
