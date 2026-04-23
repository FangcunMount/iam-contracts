package config

// Validate 验证配置有效性。
func (c *Config) Validate() error {
	if c.Endpoint == "" {
		return ErrEndpointRequired
	}
	return nil
}
