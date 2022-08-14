/*
 * Copyright Skyramp Authors 2022
 */

package main

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"fmt"
	"net"
	"time"
	"io"
	"net/http"
	"crypto/tls"
	"github.com/apache/thrift/lib/go/thrift"
)

type Protocol int

const (
	BINARY Protocol = iota
	JSON
	SIMPLEJSON
	COMPACT
)

type Option struct {
	httpTransport bool
	httpUrl string
	Protocol Protocol
	Secure   bool
	Buffered bool
	Framed   bool
}

func NewDefaultOption() *Option {
	return &Option{
		Protocol: BINARY,
		httpTransport: false,
		Secure:   false,
		Buffered: true,
		Framed:   false,
	}
}

var (
	protocolFactoryMap          = make(map[Protocol]thrift.TProtocolFactory)
	bufferedTransportFactoryMap = make(map[bool]thrift.TTransportFactory)
)

func init() {
	protocolFactoryMap[BINARY] = thrift.NewTBinaryProtocolFactoryConf(nil)
	protocolFactoryMap[JSON] = thrift.NewTJSONProtocolFactory()
	protocolFactoryMap[SIMPLEJSON] = thrift.NewTSimpleJSONProtocolFactoryConf(nil)
	protocolFactoryMap[COMPACT] = thrift.NewTCompactProtocolFactoryConf(nil)
	bufferedTransportFactoryMap[true] = thrift.NewTBufferedTransportFactory(8192)
	bufferedTransportFactoryMap[false] = thrift.NewTTransportFactory()
}

func NewThriftClient(addr string, opt *Option) (*thrift.TStandardClient, io.Closer, error)   {
	protocolFactory, ok := protocolFactoryMap[opt.Protocol]
	if !ok {
		return nil, nil, fmt.Errorf("unknown protocol")
	}

	var transportFactory thrift.TTransportFactory
	cfg := &thrift.TConfiguration{
		TLSConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	transportFactory = bufferedTransportFactoryMap[opt.Buffered]

	if opt.Framed {
		transportFactory = thrift.NewTFramedTransportFactoryConf(transportFactory, cfg)
	}

	var transport thrift.TTransport
	if opt.Secure {
		transport = thrift.NewTSSLSocketConf(addr, cfg)
	} else {
		transport = thrift.NewTSocketConf(addr, cfg)
	}
	transport, err := transportFactory.GetTransport(transport)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get transportFactory %w n", err)
	}
	if err := transport.Open(); err != nil {
		return nil, nil, fmt.Errorf("failed to Open transport: %w", err)
	}
	iprot := protocolFactory.GetProtocol(transport)
	oprot := protocolFactory.GetProtocol(transport)
	return thrift.NewTStandardClient(iprot, oprot), transport, nil
}

func NewThriftServer(addr string, opt *Option, processor thrift.TProcessor)  error {
	protocolFactory, ok := protocolFactoryMap[opt.Protocol]
	if !ok {
		return fmt.Errorf("unknown protocol for thrift server %d" , opt.Protocol)
	}

	if opt.httpTransport  {
		http.HandleFunc( opt.httpUrl, thrift.NewThriftHandlerFunc(processor, protocolFactory, protocolFactory))
		fmt.Printf("Starting Thrift http server... on %s \n", addr)
		var err error
		if opt.Secure {
			startHttps(addr)
		} else {
			err = http.ListenAndServe(addr, nil)
		}
		if err != nil {
			fmt.Printf("Failed to start http server: %v\n", err)
		}
		return nil
	}


	var transportFactory thrift.TTransportFactory
	cfg := &thrift.TConfiguration{
		TLSConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	transportFactory = bufferedTransportFactoryMap[opt.Buffered]

	if opt.Framed {
		transportFactory = thrift.NewTFramedTransportFactoryConf(transportFactory, cfg)
	}
	var transport thrift.TServerTransport
	var err error
	if opt.Secure {
		serverTLSConf, clientTLSConf, caPEM, err := generateCertificate()
		_, _ = clientTLSConf, caPEM
		if err != nil {
			return  fmt.Errorf("failed to create tls certificate %w", err)
		}
		transport, err = thrift.NewTSSLServerSocket(addr, serverTLSConf)
		if err != nil {
			return  fmt.Errorf("failed to create tls certificate %w", err)
		}
	} else {
		transport, err = thrift.NewTServerSocket(addr)
		if err != nil {
			return  fmt.Errorf("failed to create thrift server %w", err)
		}
	}
	fmt.Printf("Starting Thrift server... on port %s \n", addr)
	server := thrift.NewTSimpleServer4(processor, transport, transportFactory, protocolFactory)
	err = server.Serve()
	if err != nil {
		fmt.Printf("failed to start Thrift server... on port %s \n", addr)
	}
	fmt.Printf("Thrift server... on port %s terminated \n", addr)
	return nil
}

// Starts https with in memory generated certificate
func startHttps(addr string) {
	serverTLSConf, clientTLSConf, caPEM, _ := generateCertificate()
	_, _ = clientTLSConf, caPEM
	getCertificate := func(info *tls.ClientHelloInfo) (*tls.Certificate, error) {
		return &serverTLSConf.Certificates[0], nil
	}

	srv := &http.Server{
		Addr:    addr,
		Handler: http.DefaultServeMux,
		TLSConfig: &tls.Config{
			MinVersion:               tls.VersionTLS13,
			GetCertificate:           getCertificate,
		},
	}
	srv.ListenAndServeTLS("", "")
}

// Generates am in memory TLS Certificate
func generateCertificate() (serverTLSConf *tls.Config, clientTLSConf *tls.Config, caPEMBytes []byte, err error) {
	ca := &x509.Certificate{
		SerialNumber: big.NewInt(2019),
		Subject: pkix.Name{
			Organization:  []string{"Skyramp, Inc."},
			Country:       []string{"US"},
			Province:      []string{"CA"},
			Locality:      []string{"San Francisco"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	// create  private and public key
	caPrivKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, nil, []byte{}, err
	}

	// create the CA
	caBytes, err := x509.CreateCertificate(rand.Reader, ca, ca, &caPrivKey.PublicKey, caPrivKey)
	if err != nil {
		return nil, nil, []byte{}, err
	}

	// pem encode
	caPEM := new(bytes.Buffer)
	pem.Encode(caPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caBytes,
	})

	caPrivKeyPEM := new(bytes.Buffer)
	pem.Encode(caPrivKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(caPrivKey),
	})

	cert := &x509.Certificate{
		SerialNumber: big.NewInt(2019),
		Subject: pkix.Name{
			Organization:  []string{"Skyramp, Inc."},
			Country:       []string{"US"},
			Province:      []string{"CA"},
			Locality:      []string{"San Francisco"},
		},
		IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(10, 0, 0),
		SubjectKeyId: []byte{1, 2, 3, 4, 6},
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature,
		DNSNames:     []string{"mockworker"},
	}

	certPrivKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, nil, []byte{}, err
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, cert, ca, &certPrivKey.PublicKey, caPrivKey)
	if err != nil {
		return nil, nil, []byte{}, err
	}

	certPEM := new(bytes.Buffer)
	pem.Encode(certPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})

	certPrivKeyPEM := new(bytes.Buffer)
	pem.Encode(certPrivKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(certPrivKey),
	})

	serverCert, err := tls.X509KeyPair(certPEM.Bytes(), certPrivKeyPEM.Bytes())
	if err != nil {
		return nil, nil, []byte{}, err
	}

	serverTLSConf = &tls.Config{
		Certificates: []tls.Certificate{serverCert},
	}

	certpool := x509.NewCertPool()
	certpool.AppendCertsFromPEM(caPEM.Bytes())
	clientTLSConf = &tls.Config{
		RootCAs: certpool,
	}

	caPEMBytes = caPEM.Bytes()

	return
}

