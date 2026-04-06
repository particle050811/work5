package main

import (
	"errors"
	"os"
	"strings"
)

func isRemoteErr(err error, target string) bool {
	if err == nil {
		return false
	}
	var joined interface{ Unwrap() []error }
	if errors.As(err, &joined) {
		for _, e := range joined.Unwrap() {
			if strings.Contains(e.Error(), target) {
				return true
			}
		}
	}
	return strings.Contains(err.Error(), target)
}

func getEnv(key, fallback string) string {
	val := strings.TrimSpace(os.Getenv(key))
	if val == "" {
		return fallback
	}
	return val
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
