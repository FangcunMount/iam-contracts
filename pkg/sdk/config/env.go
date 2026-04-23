package config

// FromEnv 从环境变量加载配置。
func FromEnv() (*Config, error) {
	return FromEnvWithPrefix("IAM")
}

// FromEnvWithPrefix 从带前缀的环境变量加载配置。
func FromEnvWithPrefix(prefix string) (*Config, error) {
	return NewEnvLoader(prefix).Load()
}
