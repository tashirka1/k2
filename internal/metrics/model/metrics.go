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

type ResourceBucket struct {
	Timestamp string
	Avg       float64
	Min       float64
	Max       float64
}

type ProcessBucket struct {
	Timestamp   string
	CPUAvg      float64
	CPUMin      float64
	CPUMax      float64
	RAMAvg      float64
	RAMMin      float64
	RAMMax      float64
	RAMBytesAvg int64
	RAMBytesMin int64
	RAMBytesMax int64
}

type ContainerBucket struct {
	Timestamp   string
	CPUAvg      float64
	CPUMin      float64
	CPUMax      float64
	RAMAvg      float64
	RAMMin      float64
	RAMMax      float64
	RAMBytesAvg int64
	RAMBytesMin int64
	RAMBytesMax int64
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
