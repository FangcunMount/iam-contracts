package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

func (l *EnvLoader) fullKey(key string) string {
	return l.prefix + "_" + key
}

func (l *EnvLoader) hasValue(key string) bool {
	_, ok := os.LookupEnv(l.fullKey(key))
	return ok
}

func (l *EnvLoader) sectionEnabled(flag string, keys ...string) bool {
	if l.hasValue(flag) {
		return l.getBool(flag, false)
	}
	for _, key := range keys {
		if l.hasValue(key) {
			return true
		}
	}
	return false
}

func (l *EnvLoader) getString(key, def string) string {
	if v, ok := os.LookupEnv(l.fullKey(key)); ok {
		return v
	}
	return def
}

func (l *EnvLoader) getBool(key string, def bool) bool {
	v, ok := os.LookupEnv(l.fullKey(key))
	if !ok {
		return def
	}
	v = strings.ToLower(v)
	return v == "true" || v == "1" || v == "yes"
}

func (l *EnvLoader) getInt(key string, def int) int {
	v, ok := os.LookupEnv(l.fullKey(key))
	if !ok {
		return def
	}
	if i, err := strconv.Atoi(v); err == nil {
		return i
	}
	return def
}

func (l *EnvLoader) getFloat64(key string, def float64) float64 {
	v, ok := os.LookupEnv(l.fullKey(key))
	if !ok {
		return def
	}
	if parsed, err := strconv.ParseFloat(v, 64); err == nil {
		return parsed
	}
	return def
}

func (l *EnvLoader) getDuration(key string, def time.Duration) time.Duration {
	v, ok := os.LookupEnv(l.fullKey(key))
	if !ok {
		return def
	}
	if d, err := time.ParseDuration(v); err == nil {
		return d
	}
	return def
}
