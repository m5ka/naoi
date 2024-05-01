package runner

import (
	"context"
	"github.com/docker/docker/client"
	"go.m5ka.dev/naoi/internal/pipeline"
	"os"
)

type Runner struct {
	Client     *client.Client
	Containers []*Container
}

func New(c *pipeline.Config) (*Runner, error) {
	cli, err := client.NewClientWithOpts(client.WithHost(os.Getenv("NAOI_DOCKER_HOST")))
	if err != nil {
		return nil, err
	}
	return &Runner{
		Client: cli,
	}, nil
}

func (r *Runner) Close(ctx context.Context) error {
	var err error
	for _, container := range r.Containers {
		err = container.Close(ctx)
	}
	err = r.Client.Close()
	return err
}
