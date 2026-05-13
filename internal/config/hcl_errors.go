package config

type ConfigNotFoundError struct {
	Dir string
}

func (e *ConfigNotFoundError) Error() string {
	return "config file not found in " + e.Dir
}
