package service

import (
	"context"
	"sort"
	"time"

	"github.com/tashirka1/k2/internal/metrics/model"
)

func (s *Metrics) QueryChartData(ctx context.Context, metricType string, from, to time.Time) (model.ChartData, error) {
	points, err := s.r.QueryResources(ctx, metricType, from, to)
	if err != nil {
		return model.ChartData{}, err
	}
	return downsampleChartData(buildChartData(metricType, points), chartThreshold(to.Sub(from))), nil
}

func (s *Metrics) QueryProcessChart(ctx context.Context, pid int, param string, from, to time.Time) (model.ChartData, error) {
	points, err := s.r.QueryProcessHistory(ctx, pid, from, to)
	if err != nil {
		return model.ChartData{}, err
	}
	return downsampleChartData(buildProcessChartData(param, points), chartThreshold(to.Sub(from))), nil
}

func (s *Metrics) QueryContainerChart(ctx context.Context, name, param string, from, to time.Time) (model.ChartData, error) {
	points, err := s.r.QueryContainerHistory(ctx, name, from, to)
	if err != nil {
		return model.ChartData{}, err
	}
	return downsampleChartData(buildContainerChartData(param, points), chartThreshold(to.Sub(from))), nil
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
