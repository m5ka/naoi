package pipeline

import (
	"fmt"
	"github.com/go-playground/validator/v10"
	"gopkg.in/yaml.v3"
	"strings"
	"unicode/utf8"
)

type TriggerConfig struct {
	Event    string   `yaml:"event" validate:"oneof=push merge_request tag manual"`
	Branches []string `yaml:"branches"`
	If       string   `yaml:"if"`
}

type Config struct {
	Name      string               `yaml:"name" validate:"required"`
	When      []TriggerConfig      `yaml:"when" validate:"required,dive"`
	Steps     []string             `yaml:"steps" validate:"required"`
	Variables map[string]string    `yaml:"variables"`
	Default   JobConfig            `yaml:"default,omitempty"`
	Jobs      map[string]JobConfig `yaml:",inline"`
}

func Parse(raw []byte) (*Config, error) {
	config := Config{}
	if err := yaml.Unmarshal(raw, &config); err != nil {
		return nil, err
	}
	for key := range config.Jobs {
		if strings.HasPrefix(key, ".") {
			delete(config.Jobs, key)
		}
	}
	config.Hydrate()
	return &config, nil
}

func (c *Config) Hydrate() {
	for name, job := range c.Jobs {
		c.Jobs[name] = job.Hydrate(c.Default)
	}
}

func (c *Config) Validate() error {
	for _, step := range c.Steps {
		if _, ok := c.Jobs[step]; !ok {
			return fmt.Errorf("step %s is not defined as a job", step)
		}
	}
	validate := validator.New()
	if err := validate.Struct(c); err != nil {
		return err
	}
	return nil
}

func (c *Config) PrettyPrint() {
	fmt.Printf("Pipeline \"%s\"\n%s\n", c.Name, strings.Repeat("=", utf8.RuneCountInString(c.Name)+11))
	fmt.Printf("Triggered:\n")
	for _, when := range c.When {
		fmt.Printf("  - %s", triggerString(when.Event))
		if len(when.If) != 0 {
			fmt.Printf(" (if %s)", when.If)
		}
		fmt.Print("\n")
	}
	fmt.Print("Jobs:\n")
	for name, job := range c.Jobs {
		fmt.Printf("  - %s\n", name)
		prettyPrintJob(job, 6)
	}
}

func triggerString(key string) string {
	switch key {
	case "push":
		return "on push to repository"
	case "tag":
		return "on tag push to repository"
	case "merge_request":
		return "on merge request"
	case "manual":
		return "on manual trigger"
	default:
		return "unknown"
	}
}

func prettyPrintJob(job JobConfig, indentLength int) {
	indent := strings.Repeat(" ", indentLength)
	if len(job.Image) != 0 {
		fmt.Printf("%simage: %s\n", indent, job.Image)
	}
	if len(job.BeforeScript) != 0 {
		fmt.Printf("%sbefore script:\n", indent)
		for _, beforeScript := range job.BeforeScript {
			fmt.Printf("%s  - %s\n", indent, beforeScript)
		}
	}
	if len(job.Script) != 0 {
		fmt.Printf("%sscript:\n", indent)
		for _, script := range job.Script {
			fmt.Printf("%s  - %s\n", indent, script)
		}
	}
	if len(job.AfterScript) != 0 {
		fmt.Printf("%safter script:\n", indent)
		for _, afterScript := range job.AfterScript {
			fmt.Printf("%s  - %s\n", indent, afterScript)
		}
	}
}
