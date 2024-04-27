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
  image: ubuntu@latest
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

func TestParse(t *testing.T) {
	config, err := Parse([]byte(validYaml))
	assert.NotNil(t, config)
	assert.Nil(t, err)
	assert.Equal(t, config.Name, "test pipeline")
	assert.Contains(t, config.When, TriggerConfig{Event: "merge_request", If: "$GIT_REPOSITORY == 'test/test'"})
	assert.Contains(t, config.When, TriggerConfig{Event: "push", Branches: []string{"main"}})
	assert.Contains(t, config.When, TriggerConfig{Event: "manual"})
	assert.Equal(t, config.Steps, []string{"build", "release"})
	assert.Equal(t, config.Variables, map[string]string{"MY_VARIABLE": "123"})
	assert.Equal(t, config.Default.Image, "ubuntu@latest")
	assert.Equal(t, config.Default.BeforeScript, []string{"echo \"This is before.\""})
	assert.Equal(t, config.Default.Variables, map[string]string{"MY_OTHER_VARIABLE": "456"})
	assert.Contains(t, config.Jobs, "build")
	assert.Len(t, config.Jobs["build"].Image, 0)
	assert.Equal(t, config.Jobs["build"].Script, []string{"echo \"Build step one.\"", "echo \"Build step two.\""})
	assert.Equal(t, config.Jobs["build"].Cache,
		CacheConfig{Key: "build", Files: []string{"package.json"}, Paths: []string{"."}})
	assert.Contains(t, config.Jobs, "release")
	assert.Equal(t, config.Jobs["release"].Image, "example.com/container@1.2.3")
	assert.Equal(t, config.Jobs["release"].Script, []string{"echo \"Release step one.\""})
	assert.Equal(t, config.Jobs["release"].AfterScript, []string{"echo \"This is after.\""})
	assert.Equal(t, config.Jobs["release"].Variables, map[string]string{"MY_OTHER_VARIABLE": "overridden"})
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
