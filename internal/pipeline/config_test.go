package pipeline

import (
	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
	"testing"
)

const validYaml = `

name: test pipeline

when:
  - event: merge_request
    if: $GIT_REPOSITORY == 'test/test'
  - event: push
    branches:
      - main
  - event: manual

steps: [build, release]

variables:
  MY_VARIABLE: 123

default:
  image: ubuntu:latest
  before_script:
    - echo "This is before."
  variables:
    MY_OTHER_VARIABLE: 456

build:
  script:
    - echo "Build step one."
    - echo "Build step two."
  cache:
    key: build
    files:
      - package.json
    paths:
      - .

release:
  image: example.com/container@1.2.3
  script:
    - echo "Release step one."
  after_script:
    - echo "This is after."
  variables:
    MY_OTHER_VARIABLE: overridden

`

const defaultsYaml = `

name: test pipeline

when:
  - event: manual

steps: [default_job, override_job]

default:
  image: default_image
  before_script:
    - default_before
  script:
    - default_script
  after_script:
    - default_after
  cache:
    key: default_cache
    paths:
      - default_cache/

default_job:

override_job:
  image: override_image
  before_script:
    - override_before
  script:
    - override_script
  after_script:
    - override_after
  cache:
    key: override_cache
    paths:
      - override_cache/

`

const invalidWhenYaml = `

name: test pipeline

when:
  - event: always

steps: [build]

build:
  script:
    - "Build step one."

`

const invalidStepsYaml = `

name: test pipeline

when:
  - event: merge_request

steps: [build, nonsense]

build:
  script:
    - "Build step one."

`

const dotJobYaml = `

name: test pipeline

when:
  - event: merge_request

steps: [build]

.dot: &dot
  image: ubuntu:latest

build:
  <<: *dot
  script:
    - echo "̆Build step one."

`

func TestParse(t *testing.T) {
	config, err := Parse([]byte(validYaml))
	assert.NotNil(t, config)
	assert.Nil(t, err)
	assert.Equal(t, "test pipeline", config.Name)
	assert.Contains(t, config.When, TriggerConfig{Event: "merge_request", If: "$GIT_REPOSITORY == 'test/test'"})
	assert.Contains(t, config.When, TriggerConfig{Event: "push", Branches: []string{"main"}})
	assert.Contains(t, config.When, TriggerConfig{Event: "manual"})
	assert.Equal(t, []string{"build", "release"}, config.Steps)
	assert.Equal(t, map[string]string{"MY_VARIABLE": "123"}, config.Variables)
	assert.Equal(t, "ubuntu:latest", config.Default.Image)
	assert.Equal(t, []string{"echo \"This is before.\""}, config.Default.BeforeScript)
	assert.Equal(t, map[string]string{"MY_OTHER_VARIABLE": "456"}, config.Default.Variables)
	assert.Contains(t, config.Jobs, "build")
	assert.Equal(t, []string{"echo \"Build step one.\"", "echo \"Build step two.\""}, config.Jobs["build"].Script)
	assert.Equal(t, CacheConfig{Key: "build", Files: []string{"package.json"}, Paths: []string{"."}},
		config.Jobs["build"].Cache)
	assert.Contains(t, config.Jobs, "release")
	assert.Equal(t, "example.com/container@1.2.3", config.Jobs["release"].Image)
	assert.Equal(t, []string{"echo \"Release step one.\""}, config.Jobs["release"].Script)
	assert.Equal(t, []string{"echo \"This is after.\""}, config.Jobs["release"].AfterScript)
	assert.Equal(t, map[string]string{"MY_OTHER_VARIABLE": "overridden"}, config.Jobs["release"].Variables)
}

func TestDefault(t *testing.T) {
	config, err := Parse([]byte(defaultsYaml))
	assert.NotNil(t, config)
	assert.Nil(t, err)
	assert.Equal(t, "default_image", config.Jobs["default_job"].Image)
	assert.Equal(t, config.Default.Image, config.Jobs["default_job"].Image)
	assert.Equal(t, "override_image", config.Jobs["override_job"].Image)
	assert.NotEqual(t, config.Default.Image, config.Jobs["override_job"].Image)
	assert.Equal(t, []string{"default_script"}, config.Jobs["default_job"].Script)
	assert.Equal(t, config.Default.Script, config.Jobs["default_job"].Script)
	assert.Equal(t, []string{"override_script"}, config.Jobs["override_job"].Script)
	assert.NotEqual(t, config.Default.Script, config.Jobs["override_job"].Script)
}

func TestDotJob(t *testing.T) {
	config, err := Parse([]byte(dotJobYaml))
	assert.NotNil(t, config)
	assert.Nil(t, err)
	assert.Contains(t, config.Jobs, "build")
	assert.NotContains(t, config.Jobs, ".dot")
	assert.Equal(t, "ubuntu:latest", config.Jobs["build"].Image)
}

func TestValidate(t *testing.T) {
	config, _ := Parse([]byte(validYaml))
	err := config.Validate()
	assert.Nil(t, err)
}

func TestInvalidWhenValidate(t *testing.T) {
	config, _ := Parse([]byte(invalidWhenYaml))
	err := config.Validate()
	assert.Error(t, err)
	assert.ErrorAs(t, err, &validator.ValidationErrors{})
}

func TestInvalidStepsValidate(t *testing.T) {
	config, _ := Parse([]byte(invalidStepsYaml))
	err := config.Validate()
	assert.Error(t, err)
	assert.EqualError(t, err, "step nonsense is not defined as a job")
}
