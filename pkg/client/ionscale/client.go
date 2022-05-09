package ionscale

import (
	"crypto/tls"
	"github.com/jsiebens/ionscale/pkg/gen/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"io"
	"net/url"
)

func NewClient(clientAuth ClientAuth, serverURL string, insecureSkipVerify bool) (api.IonscaleClient, io.Closer, error) {
	parsedUrl, err := url.Parse(serverURL)
	if err != nil {
		return nil, nil, err
	}

	var targetAddr = parsedUrl.Host
	if parsedUrl.Port() == "" {
		targetAddr = targetAddr + ":443"
	}

	var transportCreds = credentials.NewTLS(&tls.Config{InsecureSkipVerify: insecureSkipVerify})

	if parsedUrl.Scheme != "https" {
		transportCreds = insecure.NewCredentials()
	}

	conn, err := grpc.Dial(targetAddr, grpc.WithPerRPCCredentials(clientAuth), grpc.WithTransportCredentials(transportCreds))
	if err != nil {
		return nil, nil, err
	}

	return api.NewIonscaleClient(conn), conn, nil
}
