package infura

import "context"

type Client interface {
	GetSuggestedGasFees(ctx context.Context, chainID int) (*SuggestedGasFees, error)
}

func NewClient(apiKey string) Client {
	return &client{apiKey: apiKey}
}

type client struct {
	apiKey string
}

func (cli *client) GetSuggestedGasFees(ctx context.Context, chainID int) (*SuggestedGasFees, error) {
	return GetSuggestedGasFees(ctx, cli.apiKey, chainID)
}
