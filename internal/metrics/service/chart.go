package service

import (
	"context"
	"sort"
	"time"

	"github.com/tashirka1/k2/internal/metrics/model"
)

func (s *Metrics) QueryChartData(ctx context.Context, metricType string, from, to time.Time) (model.ChartData, error) {
	bucketSec, _ := bucketForPeriod(to.Sub(from))
	buckets, err := s.r.QueryResources(ctx, metricType, from, to, bucketSec)
	if err != nil {
		return model.ChartData{}, err
	}
	if bucketSec > 0 {
		return buildResourceBucketedChartData(metricType, buckets), nil
	}
	if metricType == "disk" {
		labels := make([]string, len(buckets))
		series := make([]*float64, len(buckets))
		for i, b := range buckets {
			labels[i] = b.Timestamp
			v := b.Avg
			series[i] = &v
		}
		return model.ChartData{Labels: labels, Series: []model.ChartSeries{{Label: "Disk %", Data: series}}}, nil
	}
	points := make([]model.ResourcePoint, len(buckets))
	for i, b := range buckets {
		points[i] = model.ResourcePoint{Timestamp: b.Timestamp, Type: metricType, Name: "percent", Value: b.Avg}
	}
	return buildChartData(metricType, points), nil
}

func (s *Metrics) QueryProcessChart(ctx context.Context, pid int, param string, from, to time.Time) (model.ChartData, error) {
	bucketSec, _ := bucketForPeriod(to.Sub(from))
	buckets, err := s.r.QueryProcessHistory(ctx, pid, from, to, bucketSec)
	if err != nil {
		return model.ChartData{}, err
	}
	if bucketSec > 0 {
		return buildProcessBucketedChartData(param, buckets), nil
	}
	labels := make([]string, len(buckets))
	values := make([]*float64, len(buckets))
	for i, b := range buckets {
		labels[i] = b.Timestamp
		var v float64
		switch param {
		case "cpu":
			v = b.CPUAvg
		case "ram":
			v = b.RAMAvg
		case "ram_bytes":
			v = ramBytesToMiB(b.RAMBytesAvg)
		}
		val := v
		values[i] = &val
	}
	return buildSeriesChartData(param, labels, values), nil
}

func (s *Metrics) QueryContainerChart(ctx context.Context, name, param string, from, to time.Time) (model.ChartData, error) {
	bucketSec, _ := bucketForPeriod(to.Sub(from))
	buckets, err := s.r.QueryContainerHistory(ctx, name, from, to, bucketSec)
	if err != nil {
		return model.ChartData{}, err
	}
	if bucketSec > 0 {
		return buildContainerBucketedChartData(param, buckets), nil
	}
	labels := make([]string, len(buckets))
	values := make([]*float64, len(buckets))
	for i, b := range buckets {
		labels[i] = b.Timestamp
		var v float64
		switch param {
		case "cpu":
			v = b.CPUAvg
		case "ram":
			v = b.RAMAvg
		case "ram_bytes":
			v = ramBytesToMiB(b.RAMBytesAvg)
		}
		val := v
		values[i] = &val
	}
	return buildSeriesChartData(param, labels, values), nil
}

func bucketForPeriod(period time.Duration) (int, int) {
	if period <= 0 {
		return 0, 0
	}
	const target = 400
	bucket := niceBucket(period / target)
	threshold := int(period / bucket)
	if threshold < 3 {
		return 0, 0
	}
	return int(bucket.Seconds()), threshold
}

func niceBucket(d time.Duration) time.Duration {
	steps := []time.Duration{
		30 * time.Second, time.Minute, 5 * time.Minute, 15 * time.Minute, 30 * time.Minute,
		time.Hour, 2 * time.Hour, 4 * time.Hour, 6 * time.Hour, 12 * time.Hour, 24 * time.Hour,
	}
	for _, s := range steps {
		if s >= d {
			return s
		}
	}
	return 24 * time.Hour
}

func buildResourceBucketedChartData(metricType string, buckets []model.ResourceBucket) model.ChartData {
	labels := make([]string, len(buckets))
	avg := make([]*float64, len(buckets))
	mn := make([]*float64, len(buckets))
	mx := make([]*float64, len(buckets))
	for i, b := range buckets {
		labels[i] = b.Timestamp
		a := b.Avg
		mi := b.Min
		ma := b.Max
		avg[i] = &a
		mn[i] = &mi
		mx[i] = &ma
	}
	base := chartLabel(metricType)
	if metricType == "disk" {
		base = "Disk %"
	}
	return model.ChartData{
		Labels: labels,
		Series: []model.ChartSeries{
			{Label: base, Data: avg},
			{Label: base + " min", Data: mn},
			{Label: base + " max", Data: mx},
		},
	}
}

func buildProcessBucketedChartData(param string, buckets []model.ProcessBucket) model.ChartData {
	labels := make([]string, len(buckets))
	avg := make([]*float64, len(buckets))
	mn := make([]*float64, len(buckets))
	mx := make([]*float64, len(buckets))
	for i, b := range buckets {
		labels[i] = b.Timestamp
		var a, mi, ma float64
		switch param {
		case "cpu":
			a = b.CPUAvg
			mi = b.CPUMin
			ma = b.CPUMax
		case "ram":
			a = b.RAMAvg
			mi = b.RAMMin
			ma = b.RAMMax
		case "ram_bytes":
			a = ramBytesToMiB(b.RAMBytesAvg)
			mi = ramBytesToMiB(b.RAMBytesMin)
			ma = ramBytesToMiB(b.RAMBytesMax)
		}
		av := a
		miv := mi
		mav := ma
		avg[i] = &av
		mn[i] = &miv
		mx[i] = &mav
	}
	base := chartLabel(param)
	return model.ChartData{
		Labels: labels,
		Series: []model.ChartSeries{
			{Label: base, Data: avg},
			{Label: base + " min", Data: mn},
			{Label: base + " max", Data: mx},
		},
	}
}

func buildContainerBucketedChartData(param string, buckets []model.ContainerBucket) model.ChartData {
	labels := make([]string, len(buckets))
	avg := make([]*float64, len(buckets))
	mn := make([]*float64, len(buckets))
	mx := make([]*float64, len(buckets))
	for i, b := range buckets {
		labels[i] = b.Timestamp
		var a, mi, ma float64
		switch param {
		case "cpu":
			a = b.CPUAvg
			mi = b.CPUMin
			ma = b.CPUMax
		case "ram":
			a = b.RAMAvg
			mi = b.RAMMin
			ma = b.RAMMax
		case "ram_bytes":
			a = ramBytesToMiB(b.RAMBytesAvg)
			mi = ramBytesToMiB(b.RAMBytesMin)
			ma = ramBytesToMiB(b.RAMBytesMax)
		}
		av := a
		miv := mi
		mav := ma
		avg[i] = &av
		mn[i] = &miv
		mx[i] = &mav
	}
	base := chartLabel(param)
	return model.ChartData{
		Labels: labels,
		Series: []model.ChartSeries{
			{Label: base, Data: avg},
			{Label: base + " min", Data: mn},
			{Label: base + " max", Data: mx},
		},
	}
}

func buildChartData(metricType string, points []model.ResourcePoint) model.ChartData {
	if metricType == "disk" {
		return buildDiskChartData(points)
	}

	values := make(map[string]map[string]float64)
	timestamps := make(map[string]bool)

	for _, p := range points {
		if p.Name != "percent" {
			continue
		}
		label := chartLabel(metricType)
		if values[label] == nil {
			values[label] = make(map[string]float64)
		}
		values[label][p.Timestamp] = p.Value
		timestamps[p.Timestamp] = true
	}

	labels := make([]string, 0, len(timestamps))
	for ts := range timestamps {
		labels = append(labels, ts)
	}
	sort.Strings(labels)

	names := make([]string, 0, len(values))
	for name := range values {
		names = append(names, name)
	}
	sort.Strings(names)

	data := model.ChartData{Labels: labels, Series: []model.ChartSeries{}}
	for _, name := range names {
		series := make([]*float64, 0, len(labels))
		for _, ts := range labels {
			if v, ok := values[name][ts]; ok {
				val := v
				series = append(series, &val)
			} else {
				series = append(series, nil)
			}
		}
		data.Series = append(data.Series, model.ChartSeries{Label: name, Data: series})
	}
	return data
}

func chartLabel(metricType string) string {
	switch metricType {
	case "cpu":
		return "CPU %"
	case "ram":
		return "RAM %"
	case "ram_bytes":
		return "RAM Used (MB)"
	}
	return metricType
}

func buildDiskChartData(points []model.ResourcePoint) model.ChartData {
	usedByTs := make(map[string]float64)
	totalByTs := make(map[string]float64)

	for _, p := range points {
		switch p.Name {
		case "used":
			usedByTs[p.Timestamp] += p.Value
		case "total":
			totalByTs[p.Timestamp] += p.Value
		}
	}

	labels := make([]string, 0, len(usedByTs))
	for ts := range usedByTs {
		labels = append(labels, ts)
	}
	sort.Strings(labels)

	series := make([]*float64, 0, len(labels))
	for _, ts := range labels {
		if total := totalByTs[ts]; total > 0 {
			v := usedByTs[ts] / total * 100
			series = append(series, &v)
		} else {
			series = append(series, nil)
		}
	}

	return model.ChartData{Labels: labels, Series: []model.ChartSeries{{Label: "Disk %", Data: series}}}
}

func buildProcessChartData(param string, points []model.ProcessPoint) model.ChartData {
	labels := make([]string, 0, len(points))
	values := make([]*float64, 0, len(points))
	for _, p := range points {
		labels = append(labels, p.Timestamp)
		values = append(values, processValue(param, p))
	}
	return buildSeriesChartData(param, labels, values)
}

func buildContainerChartData(param string, points []model.ContainerPoint) model.ChartData {
	labels := make([]string, 0, len(points))
	values := make([]*float64, 0, len(points))
	for _, p := range points {
		labels = append(labels, p.Timestamp)
		values = append(values, containerValue(param, p))
	}
	return buildSeriesChartData(param, labels, values)
}

func buildSeriesChartData(param string, labels []string, values []*float64) model.ChartData {
	return model.ChartData{Labels: labels, Series: []model.ChartSeries{{Label: chartLabel(param), Data: values}}}
}

func processValue(param string, p model.ProcessPoint) *float64 {
	switch param {
	case "cpu":
		return &p.CPU
	case "ram":
		return &p.RAM
	case "ram_bytes":
		v := ramBytesToMiB(p.RAMBytes)
		return &v
	}
	return nil
}

func containerValue(param string, p model.ContainerPoint) *float64 {
	switch param {
	case "cpu":
		return &p.CPU
	case "ram":
		return &p.RAM
	case "ram_bytes":
		v := ramBytesToMiB(p.RAMBytes)
		return &v
	}
	return nil
}

func ramBytesToMiB(b int64) float64 {
	return float64(b) / 1048576
}
