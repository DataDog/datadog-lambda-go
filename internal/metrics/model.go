package metrics

import (
	"time"
)

type (
	// Metric represents a metric that can have any kind of
	Metric interface {
		AddPoint(value float64)
		ToAPIMetric(timestamp time.Time, interval time.Duration) []APIMetric
	}

	// APIMetric is a metric that can be marshalled to send to the metrics API
	APIMetric struct {
		Name       string      `json:"metric"`
		Host       *string     `json:"host,omitempty"`
		Tags       []string    `json:"tags,omitempty"`
		MetricType MetricType  `json:"type"`
		Interval   *float64    `json:"interval,omitempty"`
		Points     [][]float64 `json:"points"`
	}

	// Distribution is a type of metric that is aggregated over multiple hosts
	Distribution struct {
		Name   string
		Tags   []string
		Host   *string
		Values []float64
	}
)

// AddPoint adds a point to the distribution metric
func (d *Distribution) AddPoint(value float64) {
	d.Values = append(d.Values, value)
}

// ToAPIMetric converts a distribution into an API ready format.
func (d *Distribution) ToAPIMetric(timestamp time.Time, interval time.Duration) []APIMetric {

	intervalSeconds := new(float64)
	*intervalSeconds = interval.Seconds()

	points := make([][]float64, len(d.Values))

	currentTime := float64(timestamp.Unix())

	for i, val := range d.Values {
		points[i] = []float64{currentTime, val}
	}

	return []APIMetric{
		APIMetric{
			Name:       d.Name,
			Host:       d.Host,
			Tags:       d.Tags,
			MetricType: DistributionType,
			Points:     points,
			Interval:   intervalSeconds,
		},
	}
}
