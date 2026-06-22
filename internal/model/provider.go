package model

import (
	"context"
	"errors"
)

type Message struct {
	Role    string
	Content string
}

type Request struct {
	Model    string
	Messages []Message
}

type Response struct {
	Content  string
	Metadata map[string]string
}

type Provider interface {
	Generate(ctx context.Context, request Request) (Response, error)
}

type ProviderFunc func(ctx context.Context, request Request) (Response, error)

func (f ProviderFunc) Generate(ctx context.Context, request Request) (Response, error) {
	return f(ctx, request)
}

type Client struct {
	Provider Provider
}

func (c Client) Generate(ctx context.Context, request Request) (Response, error) {
	if c.Provider == nil {
		return Response{}, errors.New("model provider is not configured")
	}
	return c.Provider.Generate(ctx, request)
}
