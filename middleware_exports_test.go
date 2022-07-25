package speakeasy

import "time"

func ExportSetTimeNow(t time.Time) {
	timeNow = func() time.Time {
		return t
	}
}

func ExportSetTimeSince(d time.Duration) {
	timeSince = func(t time.Time) time.Duration {
		return d
	}
}

func ExportSetMaxCaptureSize(ms int) {
	maxCaptureSize = ms
}
