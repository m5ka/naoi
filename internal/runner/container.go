package runner

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	imageTypes "github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/lithammer/shortuuid/v4"
	"github.com/moby/term"
	"go.m5ka.dev/naoi/internal/pipeline"
	"io"
	"os"
	"strings"
	"time"
)

type Container struct {
	id         string
	name       string
	image      string
	started    bool
	isTerminal bool
	client     *client.Client
}

func (r *Runner) Create(ctx context.Context, j *pipeline.JobConfig) (*Container, error) {
	c := &Container{
		name:       fmt.Sprintf("naoi_%s", shortuuid.New()),
		image:      resolveImage(j.Image),
		started:    false,
		isTerminal: term.IsTerminal(os.Stdout.Fd()),
		client:     r.Client,
	}

	if err := c.pullImage(ctx); err != nil {
		return nil, err
	}

	resp, err := c.client.ContainerCreate(ctx, &container.Config{
		Image: c.image,
		Tty:   c.isTerminal,
		Cmd:   []string{"/bin/bash"},
	}, nil, nil, nil, c.name)
	if err != nil {
		return nil, err
	}

	c.id = resp.ID
	return c, nil
}

func (c *Container) Start(ctx context.Context) error {
	err := c.client.ContainerStart(ctx, c.id, container.StartOptions{})
	if err != nil {
		return err
	}

	// TODO: replace with more robust health-check
	time.Sleep(2 * time.Second)

	c.started = true
	return nil
}

func (c *Container) Exec(ctx context.Context, cmd string, env map[string]string) (int, error) {
	if !c.started {
		if err := c.Start(ctx); err != nil {
			return 0, err
		}
	}

	envList := make([]string, 0)
	for key, value := range env {
		envList = append(envList, fmt.Sprintf("%s=%s", key, value))
	}

	exec, err := c.client.ContainerExecCreate(ctx, c.id, types.ExecConfig{
		Cmd:          []string{"/bin/bash", "-c", cmd},
		Env:          envList,
		Tty:          c.isTerminal,
		AttachStdout: true,
		AttachStderr: true,
	})
	if err != nil {
		return 0, err
	}

	if err := c.attach(ctx, exec.ID); err != nil {
		return 0, err
	}

	inspect, err := c.client.ContainerExecInspect(ctx, exec.ID)
	if err != nil {
		return 0, err
	}
	return inspect.ExitCode, nil
}

func (c *Container) Close(ctx context.Context) error {
	return c.client.ContainerRemove(ctx, c.id, container.RemoveOptions{
		RemoveVolumes: true,
		Force:         true,
	})
}

func (c *Container) pullImage(ctx context.Context) error {
	imageIo, err := c.client.ImagePull(ctx, c.image, imageTypes.PullOptions{})
	if err != nil {
		return err
	}
	defer func() {
		if err := imageIo.Close(); err != nil {
			panic(err)
		}
	}()

	termFd, isTerm := term.GetFdInfo(os.Stderr)
	if err := jsonmessage.DisplayJSONMessagesStream(
		imageIo, os.Stdout, termFd, isTerm, nil); err != nil {
		return err
	}

	return nil
}

func (c *Container) attach(ctx context.Context, execID string) (err error) {
	resp, err := c.client.ContainerExecAttach(ctx, execID, types.ExecStartCheck{Tty: c.isTerminal})
	if err != nil {
		return err
	}
	defer resp.Close()

	if c.isTerminal {
		_, err = io.Copy(os.Stdout, resp.Reader)
	} else {
		_, err = stdcopy.StdCopy(os.Stdout, os.Stderr, resp.Reader)
	}
	return
}

func resolveImage(image string) string {
	if strings.Contains(image, "/") {
		return image
	}
	return fmt.Sprintf("docker.io/library/%s", image)
}
