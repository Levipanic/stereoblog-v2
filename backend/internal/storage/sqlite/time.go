package sqlite

import (
	"errors"
	"time"
)

const TimeFormat = "2006-01-02 15:04:05"

func ParseTime(value string) (time.Time, error) {
	for _, layout := range []string{TimeFormat, "2006-01-02 15:04:05.999999999"} {
		if parsed, err := time.ParseInLocation(layout, value, time.UTC); err == nil {
			return parsed, nil
		}
	}
	if parsed, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return parsed.UTC(), nil
	}
	return time.Time{}, errors.New("unsupported SQLite timestamp")
}

func FormatTime(value time.Time) string {
	return value.UTC().Format(TimeFormat)
}
