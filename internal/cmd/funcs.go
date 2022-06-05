package cmd

import (
	"context"
	"fmt"
	"github.com/bufbuild/connect-go"
	api "github.com/jsiebens/ionscale/pkg/gen/ionscale/v1"
	apiconnect "github.com/jsiebens/ionscale/pkg/gen/ionscale/v1/ionscalev1connect"
)

func findTailnet(client apiconnect.IonscaleServiceClient, tailnet string, tailnetID uint64) (*api.Tailnet, error) {
	if tailnetID == 0 && tailnet == "" {
		return nil, fmt.Errorf("requested tailnet not found or you are not authorized for this tailnet")
	}

	tailnets, err := client.ListTailnets(context.Background(), connect.NewRequest(&api.ListTailnetRequest{}))
	if err != nil {
		return nil, err
	}

	for _, t := range tailnets.Msg.Tailnet {
		if t.Id == tailnetID || t.Name == tailnet {
			return t, nil
		}
	}

	return nil, fmt.Errorf("requested tailnet not found or you are not authorized for this tailnet")
}

func findAuthMethod(client apiconnect.IonscaleServiceClient, authMethod string, authMethodID uint64) (*api.AuthMethod, error) {
	if authMethodID == 0 && authMethod == "" {
		return nil, fmt.Errorf("requested auth method not found or you are not authorized for this tailnet")
	}

	resp, err := client.ListAuthMethods(context.Background(), connect.NewRequest(&api.ListAuthMethodsRequest{}))
	if err != nil {
		return nil, err
	}

	for _, t := range resp.Msg.AuthMethods {
		if t.Id == authMethodID || t.Name == authMethod {
			return t, nil
		}
	}

	return nil, fmt.Errorf("requested auth method not found or you are not authorized for this tailnet")
}
