package metrics

import (
	"time"
)

type (
	// Metric represents a metric that can have any kind of
	Metric interface {
		AddPoint(value float64)
		ToAPIMetric(timestamp time.Time, interval time.Duration) []APIMetric
		ToBatchKey() BatchKey
		Join(metric Metric)
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

// ToBatchKey returns a key that can be used to batch the metric
func (d *Distribution) ToBatchKey() BatchKey {
	return BatchKey{
		name:       d.Name,
		host:       d.Host,
		tags:       d.Tags,
		metricType: DistributionType,
	}
}

// Join creates a union between two metric sets
func (d *Distribution) Join(metric Metric) {
	otherDist, ok := metric.(*Distribution)
	if !ok {
		return
	}
	for _, val := range otherDist.Values {
		d.AddPoint(val)
	}

}

// ToAPIMetric converts a distribution into an API ready format.
func (d *Distribution) ToAPIMetric(timestamp time.Time, interval time.Duration) []APIMetric {
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
			Interval:   nil,
		},
	}
}
