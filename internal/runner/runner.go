package runner

import (
	"context"
	"fmt"
	"github.com/docker/docker/client"
	"go.m5ka.dev/naoi/internal/pipeline"
	"os"
)

type Runner struct {
	Client   *client.Client
	Pipeline *pipeline.Config
}

func New(pipelineConfig *pipeline.Config) (*Runner, error) {
	cli, err := client.NewClientWithOpts(client.WithHost(os.Getenv("NAOI_DOCKER_HOST")))
	if err != nil {
		return nil, err
	}
	return &Runner{
		Client:   cli,
		Pipeline: pipelineConfig,
	}, nil
}

func (r *Runner) Run() (int, error) {
	for _, step := range r.Pipeline.Steps {
		job := r.Pipeline.Jobs[step]
		if ret, err := r.runJob(&job, step); ret > 0 || err != nil {
			return ret, err
		}
	}
	return 0, nil
}

func (r *Runner) runJob(job *pipeline.JobConfig, jobName string) (int, error) {
	ctx := context.Background()
	c, err := r.Create(ctx, job, jobName)
	if err != nil {
		return 0, err
	}
	defer c.Close(ctx)

	// TODO: unpack repository into working directory of container

	env := r.hydrateVariables(job)
	for _, scripts := range [][]string{job.BeforeScript, job.Script, job.AfterScript} {
		for _, script := range scripts {
			ret, err := c.Exec(ctx, script, env)
			if err != nil {
				return 0, err
			}
			if ret > 0 {
				fmt.Fprintf(os.Stderr, "non-zero exit status from command: %s\n", script)
				return ret, nil
			}
		}
	}
	return 0, nil
}

func (r *Runner) Close() error {
	return r.Client.Close()
}

func (r *Runner) hydrateVariables(job *pipeline.JobConfig) []string {
	env := make(map[string]string)
	for key, variable := range r.Pipeline.Variables {
		env[key] = variable
	}
	for key, variable := range job.Variables {
		env[key] = variable
	}
	envList := make([]string, 0, len(env))
	for key, variable := range env {
		envList = append(envList, fmt.Sprintf("%s=%s", key, variable))
	}
	return envList
}
