package config

import "time"

func (l *ViperLoader) getValue(key string) interface{} {
	return l.getter(key)
}

func (l *ViperLoader) getString(key string) string {
	v := l.getter(key)
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func (l *ViperLoader) getStringDefault(key, def string) string {
	s := l.getString(key)
	if s == "" {
		return def
	}
	return s
}

func (l *ViperLoader) getBool(key string, def bool) bool {
	v := l.getter(key)
	if v == nil {
		return def
	}
	if b, ok := v.(bool); ok {
		return b
	}
	return def
}

func (l *ViperLoader) getInt(key string, def int) int {
	v := l.getter(key)
	if v == nil {
		return def
	}
	switch t := v.(type) {
	case int:
		return t
	case int64:
		return int(t)
	case float64:
		return int(t)
	}
	return def
}

func (l *ViperLoader) getFloat64(key string, def float64) float64 {
	v := l.getter(key)
	if v == nil {
		return def
	}
	switch t := v.(type) {
	case float64:
		return t
	case int:
		return float64(t)
	case int64:
		return float64(t)
	}
	return def
}

func (l *ViperLoader) getDuration(key string, def time.Duration) time.Duration {
	v := l.getter(key)
	if v == nil {
		return def
	}
	switch t := v.(type) {
	case time.Duration:
		return t
	case string:
		if d, err := time.ParseDuration(t); err == nil {
			return d
		}
	case int64:
		return time.Duration(t)
	}
	return def
}
