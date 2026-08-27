package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBucketForPeriod(t *testing.T) {
	tests := []struct {
		name       string
		period     time.Duration
		wantBucket int
		wantThr    int
	}{
		{name: "one hour", period: time.Hour, wantBucket: 30, wantThr: 120},
		{name: "six hours", period: 6 * time.Hour, wantBucket: 60, wantThr: 360},
		{name: "one day", period: 24 * time.Hour, wantBucket: 300, wantThr: 288},
		{name: "seven days", period: 168 * time.Hour, wantBucket: 1800, wantThr: 336},
		{name: "one month", period: 720 * time.Hour, wantBucket: 7200, wantThr: 360},
		{name: "zero period", period: 0, wantBucket: 0, wantThr: 0},
		{name: "negative period", period: -time.Hour, wantBucket: 0, wantThr: 0},
		{name: "small period below threshold", period: time.Minute, wantBucket: 0, wantThr: 0},
		{name: "two minutes", period: 2 * time.Minute, wantBucket: 30, wantThr: 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotBucket, gotThr := bucketForPeriod(tt.period)
			assert.Equal(t, tt.wantBucket, gotBucket)
			assert.Equal(t, tt.wantThr, gotThr)
		})
	}
}

func TestNiceBucket(t *testing.T) {
	tests := []struct {
		name string
		d    time.Duration
		want time.Duration
	}{
		{name: "9s -> 30s", d: 9 * time.Second, want: 30 * time.Second},
		{name: "54s -> 1m", d: 54 * time.Second, want: time.Minute},
		{name: "216s -> 5m", d: 216 * time.Second, want: 5 * time.Minute},
		{name: "1512s -> 30m", d: 1512 * time.Second, want: 30 * time.Minute},
		{name: "6480s -> 2h", d: 6480 * time.Second, want: 2 * time.Hour},
		{name: "100h -> 24h", d: 100 * time.Hour, want: 24 * time.Hour},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, niceBucket(tt.d))
		})
	}
}
