package cmd

import (
	"context"
	"fmt"
	"github.com/jsiebens/ionscale/pkg/gen/api"
	"io"
)

func findTailnet(client api.IonscaleClient, tailnet string, tailnetID uint64) (*api.Tailnet, error) {
	if tailnetID == 0 && tailnet == "" {
		return nil, fmt.Errorf("requested tailnet not found or you are not authorized for this tailnet")
	}

	tailnets, err := client.ListTailnets(context.Background(), &api.ListTailnetRequest{})
	if err != nil {
		return nil, err
	}

	for _, t := range tailnets.Tailnet {
		if t.Id == tailnetID || t.Name == tailnet {
			return t, nil
		}
	}

	return nil, fmt.Errorf("requested tailnet not found or you are not authorized for this tailnet")
}

func safeClose(c io.Closer) {
	if c != nil {
		_ = c.Close()
	}
}
