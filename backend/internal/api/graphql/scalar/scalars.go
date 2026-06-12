//go:build graphql

package scalar

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/99designs/gqlgen/graphql"
)

// DateTime scalar represents a date and time in RFC3339 format
type DateTime time.Time

func (d DateTime) MarshalGQL(w io.Writer) {
	t := time.Time(d)
	b, _ := json.Marshal(t.Format(time.RFC3339))
	w.Write(b)
}

func (d *DateTime) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("DateTime must be a string")
	}

	t, err := time.Parse(time.RFC3339, str)
	if err != nil {
		return fmt.Errorf("invalid DateTime format: %w", err)
	}

	*d = DateTime(t)
	return nil
}

// JSON scalar represents arbitrary JSON data
type JSON map[string]interface{}

func (j JSON) MarshalGQL(w io.Writer) {
	b, _ := json.Marshal(j)
	w.Write(b)
}

func (j *JSON) UnmarshalGQL(v interface{}) error {
	switch v := v.(type) {
	case map[string]interface{}:
		*j = v
		return nil
	case string:
		var data map[string]interface{}
		if err := json.Unmarshal([]byte(v), &data); err != nil {
			return fmt.Errorf("invalid JSON: %w", err)
		}
		*j = data
		return nil
	default:
		return fmt.Errorf("JSON must be an object or JSON string")
	}
}

// Duration scalar represents a time duration
type Duration time.Duration

func (d Duration) MarshalGQL(w io.Writer) {
	dur := time.Duration(d)
	graphql.MarshalString(dur.String()).MarshalGQL(w)
}

func (d *Duration) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("Duration must be a string")
	}

	dur, err := time.ParseDuration(str)
	if err != nil {
		return fmt.Errorf("invalid Duration format: %w", err)
	}

	*d = Duration(dur)
	return nil
}

// ByteSize scalar represents a size in bytes
type ByteSize int64

func (b ByteSize) MarshalGQL(w io.Writer) {
	// Return as a string with human-readable format
	size := int64(b)
	var formatted string

	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
		PB = TB * 1024
	)

	switch {
	case size >= PB:
		formatted = fmt.Sprintf("%.2f PB", float64(size)/float64(PB))
	case size >= TB:
		formatted = fmt.Sprintf("%.2f TB", float64(size)/float64(TB))
	case size >= GB:
		formatted = fmt.Sprintf("%.2f GB", float64(size)/float64(GB))
	case size >= MB:
		formatted = fmt.Sprintf("%.2f MB", float64(size)/float64(MB))
	case size >= KB:
		formatted = fmt.Sprintf("%.2f KB", float64(size)/float64(KB))
	default:
		formatted = fmt.Sprintf("%d B", size)
	}

	graphql.MarshalString(formatted).MarshalGQL(w)
}

func (b *ByteSize) UnmarshalGQL(v interface{}) error {
	switch v := v.(type) {
	case int:
		*b = ByteSize(v)
		return nil
	case int64:
		*b = ByteSize(v)
		return nil
	case float64:
		*b = ByteSize(int64(v))
		return nil
	case string:
		// Try to parse as integer first
		if val, err := strconv.ParseInt(v, 10, 64); err == nil {
			*b = ByteSize(val)
			return nil
		}
		return fmt.Errorf("ByteSize must be a number or parseable string")
	default:
		return fmt.Errorf("ByteSize must be a number")
	}
}

// Helper functions for converting between custom scalars and Go types
func TimeToDateTime(t time.Time) DateTime {
	return DateTime(t)
}

func DateTimeToTime(d DateTime) time.Time {
	return time.Time(d)
}

func DurationToScalar(d time.Duration) Duration {
	return Duration(d)
}

func ScalarToDuration(s Duration) time.Duration {
	return time.Duration(s)
}

func Int64ToByteSize(i int64) ByteSize {
	return ByteSize(i)
}

func ByteSizeToInt64(b ByteSize) int64 {
	return int64(b)
}

func MapToJSON(m map[string]interface{}) JSON {
	return JSON(m)
}

func JSONToMap(j JSON) map[string]interface{} {
	return map[string]interface{}(j)
}
