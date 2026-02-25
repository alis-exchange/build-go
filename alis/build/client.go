package build

import (
	"crypto/tls"
	"crypto/x509"

	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const defaultHost = "gateway-build-v1-597696786316.europe-west1.run.app:443"

// NewClient creates a new Ideate client.
func NewClient() (BuildServiceClient, error) {
	maxSizeOptions := grpc.WithDefaultCallOptions(
		grpc.MaxCallSendMsgSize(2_000_000_000),
		grpc.MaxCallRecvMsgSize(2_000_000_000),
	)

	if connIdeate, err := newConn(defaultHost, maxSizeOptions); err != nil {
		return nil, err
	} else {
		return NewBuildServiceClient(connIdeate), nil
	}
}

func newConn(host string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	// Validate the host argument using a regular expression to ensure it matches the required format
	if host != "" {
		opts = append(opts, grpc.WithAuthority(host))
	}
	// If the connection is secure, get the system root CAs and create a transport credentials option
	// using TLS with the system root CAs.
	systemRoots, err := x509.SystemCertPool()
	if err != nil {
		return nil, err
	}
	cred := credentials.NewTLS(&tls.Config{RootCAs: systemRoots})
	opts = append(opts, grpc.WithTransportCredentials(cred))

	return grpc.NewClient(host, opts...)
}
