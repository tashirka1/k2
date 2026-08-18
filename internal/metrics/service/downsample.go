package service

import (
	"math"
	"time"

	"github.com/tashirka1/k2/internal/metrics/model"
)

func chartThreshold(period time.Duration) int {
	if period <= 0 {
		return 0
	}
	const target = 600
	bucket := niceBucket(period / target)
	threshold := int(period / bucket)
	if threshold < 3 {
		return 0
	}
	return threshold
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

func downsampleChartData(d model.ChartData, threshold int) model.ChartData {
	if threshold < 3 || len(d.Labels) <= threshold {
		return d
	}

	var ref []float64
	for _, s := range d.Series {
		if len(s.Data) == 0 {
			continue
		}
		ref = make([]float64, len(s.Data))
		for i, v := range s.Data {
			if v != nil {
				ref[i] = *v
			}
		}
		break
	}
	if len(ref) == 0 {
		return d
	}

	idx := lttbIndices(ref, threshold)
	labels := make([]string, 0, len(idx))
	series := make([]model.ChartSeries, 0, len(d.Series))
	for _, i := range idx {
		labels = append(labels, d.Labels[i])
	}
	for _, s := range d.Series {
		data := make([]*float64, 0, len(idx))
		for _, i := range idx {
			if i < len(s.Data) {
				data = append(data, s.Data[i])
			}
		}
		series = append(series, model.ChartSeries{Label: s.Label, Data: data})
	}
	return model.ChartData{Labels: labels, Series: series}
}

func lttbIndices(values []float64, threshold int) []int {
	n := len(values)
	if threshold >= n || threshold < 3 {
		return nil
	}

	idx := make([]int, 0, threshold)
	idx = append(idx, 0)

	bucketSize := float64(n-2) / float64(threshold-2)
	a := 0
	for i := 0; i < threshold-2; i++ {
		avgStart := int(float64(i+1)*bucketSize) + 1
		avgEnd := int(float64(i+2)*bucketSize) + 1
		if avgEnd > n {
			avgEnd = n
		}
		if avgEnd < avgStart+1 {
			avgEnd = avgStart + 1
		}

		var avgX, avgY float64
		count := 0
		for j := avgStart; j < avgEnd; j++ {
			avgX += float64(j)
			avgY += values[j]
			count++
		}
		avgX /= float64(count)
		avgY /= float64(count)

		rangeStart := int(float64(i)*bucketSize) + 1
		rangeEnd := int(float64(i+1)*bucketSize) + 1
		if rangeEnd < rangeStart+2 {
			rangeEnd = rangeStart + 2
		}
		if rangeEnd > n {
			rangeEnd = n
		}

		maxArea := -1.0
		maxJ := rangeStart
		pointAX := float64(a)
		pointAY := values[a]
		for j := rangeStart; j < rangeEnd; j++ {
			area := math.Abs((pointAX-avgX)*(values[j]-pointAY) - (pointAX-float64(j))*(avgY-pointAY))
			if area > maxArea {
				maxArea = area
				maxJ = j
			}
		}
		a = maxJ
		idx = append(idx, a)
	}

	idx = append(idx, n-1)
	return idx
}
