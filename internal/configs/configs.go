package configs

type Configs interface {
	Validate() error
	LoadFromEnv() error
}
