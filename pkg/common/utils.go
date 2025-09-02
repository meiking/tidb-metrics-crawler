package common

import (
	"encoding/json"
)

// MapToJSONString converts a map[string]string to a JSON string
func MapToJSONString(m map[string]string) (string, error) {
	if len(m) == 0 {
		return "{}", nil
	}

	data, err := json.Marshal(m)
	if err != nil {
		return "", err
	}

	return string(data), nil
}
