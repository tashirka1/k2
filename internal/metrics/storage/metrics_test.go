package storage

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tashirka1/k2/internal/metrics/model"
)

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	_, err = db.Exec(`CREATE TABLE metrics_resource (id INTEGER PRIMARY KEY AUTOINCREMENT, timestamp DATETIME NOT NULL, type TEXT NOT NULL, name TEXT NOT NULL, device TEXT, value REAL NOT NULL);
		CREATE INDEX idx_resource_type_ts ON metrics_resource(type, timestamp);
		CREATE TABLE metrics_process (id INTEGER PRIMARY KEY AUTOINCREMENT, timestamp DATETIME NOT NULL, pid INTEGER NOT NULL, name TEXT NOT NULL, cpu REAL NOT NULL, ram REAL NOT NULL, ram_bytes INTEGER NOT NULL DEFAULT 0);
		CREATE INDEX idx_process_pid_ts ON metrics_process(pid, timestamp);
		CREATE TABLE metrics_container (id INTEGER PRIMARY KEY AUTOINCREMENT, timestamp DATETIME NOT NULL, name TEXT NOT NULL, image TEXT NOT NULL, cpu REAL NOT NULL, ram REAL NOT NULL, ram_bytes INTEGER NOT NULL DEFAULT 0);
		CREATE INDEX idx_container_name_ts ON metrics_container(name, timestamp);`)
	require.NoError(t, err)
	return db
}

func TestQueryResources_BucketedEnvelope(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()
	m := NewMetrics(db)
	ctx := context.Background()
	base := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	to := base.Add(1 * time.Hour)

	// insert 30s ticks for 1h = 120 points, with peak 100 at 10:15
	for i := 0; i < 120; i++ {
		ts := base.Add(time.Duration(i) * 30 * time.Second).Format(time.RFC3339)
		val := 10.0
		if i == 30 {
			val = 100
		}
		require.NoError(t, m.InsertResourceBatch(ctx, []model.ResourcePoint{{Timestamp: ts, Type: "cpu", Name: "percent", Value: val}}))
	}

	// bucketSec 300 = 5m, 1h/300=12 buckets
	buckets, err := m.QueryResources(ctx, "cpu", base, to, 300)
	require.NoError(t, err)
	assert.Len(t, buckets, 12)
	for _, b := range buckets {
		assert.LessOrEqual(t, b.Min, b.Avg)
		assert.LessOrEqual(t, b.Avg, b.Max)
	}
	// peak bucket should have max 100
	found := false
	for _, b := range buckets {
		if b.Max == 100 {
			found = true
			assert.Greater(t, b.Avg, 10.0)
			assert.Less(t, b.Avg, 100.0)
		}
	}
	assert.True(t, found, "peak bucket with max 100 not found")

	// raw fallback
	raw, err := m.QueryResources(ctx, "cpu", base, to, 0)
	require.NoError(t, err)
	assert.Len(t, raw, 120)
	for _, b := range raw {
		assert.Equal(t, b.Min, b.Avg)
		assert.Equal(t, b.Max, b.Avg)
	}
}

func TestQueryResources_DiskBucketed(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()
	m := NewMetrics(db)
	ctx := context.Background()
	base := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)

	// two devices per timestamp, 4 timestamps 30s apart
	for i := 0; i < 4; i++ {
		ts := base.Add(time.Duration(i) * 30 * time.Second).Format(time.RFC3339)
		require.NoError(t, m.InsertResourceBatch(ctx, []model.ResourcePoint{
			{Timestamp: ts, Type: "disk", Name: "used", Device: "/", Value: 50},
			{Timestamp: ts, Type: "disk", Name: "total", Device: "/", Value: 100},
			{Timestamp: ts, Type: "disk", Name: "used", Device: "/data", Value: 25},
			{Timestamp: ts, Type: "disk", Name: "total", Device: "/data", Value: 50},
		}))
	}
	to := base.Add(2 * time.Minute)

	// bucketed with 60s bucket -> 2 buckets (4 timestamps over 2m)
	buckets, err := m.QueryResources(ctx, "disk", base, to, 60)
	require.NoError(t, err)
	assert.Len(t, buckets, 2)
	for _, b := range buckets {
		assert.InDelta(t, 50.0, b.Avg, 0.01)
		assert.InDelta(t, 50.0, b.Min, 0.01)
		assert.InDelta(t, 50.0, b.Max, 0.01)
	}

	// raw disk per timestamp pct
	raw, err := m.QueryResources(ctx, "disk", base, to, 0)
	require.NoError(t, err)
	assert.Len(t, raw, 4)
	for _, b := range raw {
		assert.InDelta(t, 50.0, b.Avg, 0.01)
	}
}

func TestQueryProcessHistory_BucketedAndRounding(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()
	m := NewMetrics(db)
	ctx := context.Background()
	base := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)

	// two points in same bucket for rounding test: bucket 60s, points 0s and 30s same minute
	ts0 := base.Format(time.RFC3339)
	ts1 := base.Add(30 * time.Second).Format(time.RFC3339)
	require.NoError(t, m.InsertProcessBatch(ctx, []model.ProcessPoint{
		{Timestamp: ts0, PID: 123, Name: "test", CPU: 10, RAM: 20, RAMBytes: 1000},
		{Timestamp: ts1, PID: 123, Name: "test", CPU: 20, RAM: 30, RAMBytes: 1001},
	}))

	to := base.Add(time.Minute)
	buckets, err := m.QueryProcessHistory(ctx, 123, base, to, 60)
	require.NoError(t, err)
	require.Len(t, buckets, 1)
	b := buckets[0]
	assert.InDelta(t, 15.0, b.CPUAvg, 0.01)
	assert.Equal(t, 10.0, b.CPUMin)
	assert.Equal(t, 20.0, b.CPUMax)
	assert.LessOrEqual(t, b.RAMMin, b.RAMAvg)
	assert.LessOrEqual(t, b.RAMAvg, b.RAMMax)
	// ram_bytes avg 1000.5 -> CAST to 1000
	assert.Equal(t, int64(1000), b.RAMBytesAvg)
	assert.Equal(t, int64(1000), b.RAMBytesMin)
	assert.Equal(t, int64(1001), b.RAMBytesMax)
	assert.LessOrEqual(t, float64(b.RAMBytesMin), float64(b.RAMBytesAvg))
	assert.LessOrEqual(t, float64(b.RAMBytesAvg), float64(b.RAMBytesMax))

	// raw
	raw, err := m.QueryProcessHistory(ctx, 123, base, to, 0)
	require.NoError(t, err)
	assert.Len(t, raw, 2)
	assert.Equal(t, raw[0].CPUMin, raw[0].CPUAvg)
}

func TestQueryContainerHistory_Bucketed(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()
	m := NewMetrics(db)
	ctx := context.Background()
	base := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)

	ts0 := base.Format(time.RFC3339)
	ts1 := base.Add(30 * time.Second).Format(time.RFC3339)
	require.NoError(t, m.InsertContainerBatch(ctx, []model.ContainerPoint{
		{Timestamp: ts0, Name: "web", Image: "nginx", CPU: 5, RAM: 10, RAMBytes: 1000},
		{Timestamp: ts1, Name: "web", Image: "nginx", CPU: 15, RAM: 20, RAMBytes: 2000},
	}))

	to := base.Add(time.Minute)
	buckets, err := m.QueryContainerHistory(ctx, "web", base, to, 60)
	require.NoError(t, err)
	require.Len(t, buckets, 1)
	assert.InDelta(t, 10.0, buckets[0].CPUAvg, 0.01)
	assert.Equal(t, 5.0, buckets[0].CPUMin)
	assert.Equal(t, 15.0, buckets[0].CPUMax)
}

func TestStrftimeParsing(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()
	m := NewMetrics(db)
	ctx := context.Background()
	ts := "2026-08-02T10:00:00Z"
	require.NoError(t, m.InsertResourceBatch(ctx, []model.ResourcePoint{{Timestamp: ts, Type: "cpu", Name: "percent", Value: 42}}))
	base, _ := time.Parse(time.RFC3339, ts)
	to := base.Add(time.Minute)
	buckets, err := m.QueryResources(ctx, "cpu", base, to, 60)
	require.NoError(t, err)
	require.Len(t, buckets, 1)
	assert.Equal(t, "2026-08-02T10:00:00Z", buckets[0].Timestamp)
	assert.InDelta(t, 42.0, buckets[0].Avg, 0.01)
}
