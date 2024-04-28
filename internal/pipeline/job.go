package pipeline

type CacheConfig struct {
	Key   string
	Files []string
	Paths []string
}

type JobConfig struct {
	Image        string            `yaml:"image" validate:"required"`
	BeforeScript []string          `yaml:"before_script"`
	Script       []string          `yaml:"script"`
	AfterScript  []string          `yaml:"after_script"`
	Variables    map[string]string `yaml:"variables"`
	Cache        CacheConfig       `yaml:"cache"`
}

func (cache *CacheConfig) Empty() bool {
	return len(cache.Key) == 0 && len(cache.Files) == 0 && len(cache.Paths) == 0
}

func (j JobConfig) Hydrate(d JobConfig) JobConfig {
	if len(j.Image) == 0 && len(d.Image) != 0 {
		j.Image = d.Image
	}
	if len(j.BeforeScript) == 0 && len(d.BeforeScript) != 0 {
		j.BeforeScript = d.BeforeScript
	}
	if len(j.Script) == 0 && len(d.Script) != 0 {
		j.Script = d.Script
	}
	if len(j.AfterScript) == 0 && len(d.AfterScript) != 0 {
		j.AfterScript = d.AfterScript
	}
	if len(j.Variables) == 0 && len(d.Variables) != 0 {
		j.Variables = d.Variables
	}
	if j.Cache.Empty() && !d.Cache.Empty() {
		j.Cache = d.Cache
	}
	return j
}
