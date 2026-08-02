package model

type ResourcePoint struct {
	Timestamp string
	Type      string
	Name      string
	Device    string
	Value     float64
}

type ProcessPoint struct {
	Timestamp string
	Name      string
	PID       int
	CPU       float64
	RAM       float64
}

type ContainerPoint struct {
	Timestamp string
	Name      string
	Image     string
	CPU       float64
	RAM       float64
}
