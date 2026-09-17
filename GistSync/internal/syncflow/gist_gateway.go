package syncflow

import (
	"context"
	"errors"

	"GistSync/internal/gistapi"
)

type GistGateway struct {
	client *gistapi.Client
}

func NewGistGateway(client *gistapi.Client) *GistGateway {
	return &GistGateway{client: client}
}

func (g *GistGateway) EnsureManifestGist(ctx context.Context) (string, error) {
	return g.client.EnsureManifestGist(ctx)
}

func (g *GistGateway) FindManifestGist(ctx context.Context) (string, error) {
	gistID, err := g.client.FindManifestGist(ctx)
	if errors.Is(err, gistapi.ErrManifestNotFound) {
		return "", nil
	}
	return gistID, err
}

func (g *GistGateway) UpsertFile(ctx context.Context, req UpsertFileRequest) error {
	return g.client.UpsertFile(ctx, gistapi.UpsertFileRequest{
		GistID: req.GistID, FileName: req.FileName, Content: req.Content,
	})
}

func (g *GistGateway) GetFileContent(ctx context.Context, req FileRequest) (string, error) {
	return g.client.GetFileContent(ctx, gistapi.FileRequest{
		GistID: req.GistID, FileName: req.FileName,
	})
}
