package service

import (
	"sort"

	"github.com/tashirka1/k2/internal/metrics/model"
)

func sortProcessPoints(points []model.ProcessPoint, s model.Sort) {
	if s.Field == "" {
		return
	}
	sort.SliceStable(points, func(i, j int) bool {
		return lessProcessPoint(points[i], points[j], s)
	})
}

func sortContainerPoints(points []model.ContainerPoint, s model.Sort) {
	if s.Field == "" {
		return
	}
	sort.SliceStable(points, func(i, j int) bool {
		return lessContainerPoint(points[i], points[j], s)
	})
}

func lessProcessPoint(a, b model.ProcessPoint, s model.Sort) bool {
	switch s.Field {
	case "pid":
		return lessNumeric(float64(a.PID), float64(b.PID), s.Desc)
	case "name":
		return lessString(a.Name, b.Name, s.Desc)
	case "cpu":
		return lessNumeric(a.CPU, b.CPU, s.Desc)
	case "ram":
		return lessNumeric(a.RAM, b.RAM, s.Desc)
	case "ram_bytes":
		return lessNumeric(float64(a.RAMBytes), float64(b.RAMBytes), s.Desc)
	}
	return false
}

func lessContainerPoint(a, b model.ContainerPoint, s model.Sort) bool {
	switch s.Field {
	case "name":
		return lessString(a.Name, b.Name, s.Desc)
	case "image":
		return lessString(a.Image, b.Image, s.Desc)
	case "cpu":
		return lessNumeric(a.CPU, b.CPU, s.Desc)
	case "ram":
		return lessNumeric(a.RAM, b.RAM, s.Desc)
	case "ram_bytes":
		return lessNumeric(float64(a.RAMBytes), float64(b.RAMBytes), s.Desc)
	}
	return false
}

func lessNumeric(a, b float64, desc bool) bool {
	if desc {
		return a > b
	}
	return a < b
}

func lessString(a, b string, desc bool) bool {
	if desc {
		return a > b
	}
	return a < b
}
