package ports

import (
	"time"
)

type Metrics interface {
	RecordOrderPlacement(duration time.Duration, status string)
	RecordMatchingLatency(duration time.Duration)
	RecordEndToEndLatency(duration time.Duration)
	RecordTrade(quantity int64)
}
