package utils

import "time"

// GetCurrentTimeMillis returns the current time in milliseconds
func GetCurrentTimeMillis() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}
