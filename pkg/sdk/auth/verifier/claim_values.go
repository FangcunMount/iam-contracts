package verifier

import (
	"encoding/json"
	"fmt"
)

func claimString(v interface{}) string {
	switch value := v.(type) {
	case string:
		return value
	case json.Number:
		return value.String()
	case float64:
		return fmt.Sprintf("%.0f", value)
	case int64:
		return fmt.Sprintf("%d", value)
	case int:
		return fmt.Sprintf("%d", value)
	case uint64:
		return fmt.Sprintf("%d", value)
	case uint:
		return fmt.Sprintf("%d", value)
	default:
		return ""
	}
}

func claimStringSlice(v interface{}) []string {
	arr, ok := v.([]interface{})
	if !ok {
		return nil
	}

	result := make([]string, 0, len(arr))
	for _, item := range arr {
		if s := claimString(item); s != "" {
			result = append(result, s)
		}
	}
	return result
}
