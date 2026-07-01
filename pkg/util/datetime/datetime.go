package datetime

import (
	"fmt"
	"time"

	"github.com/milvus-io/milvus-proto/go-api/v3/schemapb"
)

const DateFormat = "2006-01-02"

const (
	DataTypeDate schemapb.DataType = 28 // ~ponytail: pending proto enum
	DataTypeTime schemapb.DataType = 29 // ~ponytail: pending proto enum
)

var timeLayouts = []string{
	"15:04:05.999999",
	"15:04:05.999999999",
	"15:04:05",
	"15:04",
}

func ParseDateISO(s string) (int32, error) {
	t, err := time.Parse(DateFormat, s)
	if err != nil {
		return 0, fmt.Errorf("invalid date: %q (expected YYYY-MM-DD)", s)
	}
	return int32(t.Unix() / 86400), nil
}

func DateDaysToISO(days int32) string {
	return time.Unix(int64(days)*86400, 0).UTC().Format(DateFormat)
}

func ParseTimeISO(s string) (int64, error) {
	for _, layout := range timeLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return microsSinceMidnight(t), nil
		}
	}
	return 0, fmt.Errorf("invalid time: %q (expected HH:MM:SS[.ffffff])", s)
}

func microsSinceMidnight(t time.Time) int64 {
	return int64(t.Hour())*3600_000_000 +
		int64(t.Minute())*60_000_000 +
		int64(t.Second())*1_000_000 +
		int64(t.Nanosecond())/1000
}

func TimeMicrosToISO(us int64) string {
	seconds := us / 1_000_000
	frac := us % 1_000_000
	h := seconds / 3600
	m := (seconds % 3600) / 60
	s := seconds % 60
	if frac > 0 {
		return fmt.Sprintf("%02d:%02d:%02d.%06d", h, m, s, frac)
	}
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}