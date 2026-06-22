package tests

import (
	"context"
	"testing"

	modelpkg "github.com/VemorPhose/TailChase/internal/model"
)

func TestModelClientUsesProvider(t *testing.T) {
	var got modelpkg.Request
	client := modelpkg.Client{Provider: modelpkg.ProviderFunc(func(ctx context.Context, request modelpkg.Request) (modelpkg.Response, error) {
		got = request
		return modelpkg.Response{Content: "repair prompt", Metadata: map[string]string{"provider": "fake"}}, nil
	})}

	response, err := client.Generate(context.Background(), modelpkg.Request{
		Model:    "fake-model",
		Messages: []modelpkg.Message{{Role: "user", Content: "write prompt"}},
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if response.Content != "repair prompt" || response.Metadata["provider"] != "fake" {
		t.Fatalf("response = %#v, want fake response", response)
	}
	if got.Model != "fake-model" || got.Messages[0].Content != "write prompt" {
		t.Fatalf("request = %#v, want forwarded request", got)
	}
}

func TestModelClientRequiresProvider(t *testing.T) {
	if _, err := (modelpkg.Client{}).Generate(context.Background(), modelpkg.Request{}); err == nil {
		t.Fatal("Generate() error = nil, want missing provider error")
	}
}
