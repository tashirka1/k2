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
	RAMBytes  int64
}

type ContainerPoint struct {
	Timestamp string
	Name      string
	Image     string
	CPU       float64
	RAM       float64
	RAMBytes  int64
}

type Sort struct {
	Field string
	Desc  bool
}

type ChartSeries struct {
	Label string     `json:"label"`
	Data  []*float64 `json:"data"`
}

type ChartData struct {
	Labels []string      `json:"labels"`
	Series []ChartSeries `json:"series"`
}
