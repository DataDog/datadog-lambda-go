/*
 * Unless explicitly stated otherwise all files in this repository are licensed
 * under the Apache License Version 2.0.
 * 
 * This product includes software developed at Datadog (https://www.datadoghq.com/).
 * Copyright 2019 Datadog, Inc.
 */

package metrics

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetMetricDifferentTagOrder(t *testing.T) {

	tm := time.Now()
	batcher := MakeBatcher(10)
	dm1 := Distribution{
		Name:   "metric-1",
		Values: []float64{1, 2},
		Tags:   []string{"a", "b", "c"},
	}
	dm2 := Distribution{
		Name:   "metric-1",
		Values: []float64{3, 4},
		Tags:   []string{"c", "b", "a"},
	}

	batcher.AddMetric(tm, &dm1)
	batcher.AddMetric(tm, &dm2)

	assert.Equal(t, []float64{1, 2, 3, 4}, dm1.Values)
}

func TestGetMetricFailDifferentName(t *testing.T) {

	tm := time.Now()
	batcher := MakeBatcher(10)

	dm1 := Distribution{
		Name:   "metric-1",
		Values: []float64{1, 2},
		Tags:   []string{"a", "b", "c"},
	}
	dm2 := Distribution{
		Name:   "metric-2",
		Values: []float64{3, 4},
		Tags:   []string{"c", "b", "a"},
	}

	batcher.AddMetric(tm, &dm1)
	batcher.AddMetric(tm, &dm2)

	assert.Equal(t, []float64{1, 2}, dm1.Values)

}

func TestGetMetricFailDifferentHost(t *testing.T) {
	tm := time.Now()
	batcher := MakeBatcher(10)

	host1 := "my-host-1"
	host2 := "my-host-2"

	dm1 := Distribution{
		Name:   "metric-1",
		Values: []float64{1, 2},
		Tags:   []string{"a", "b", "c"},
		Host:   &host1,
	}
	dm2 := Distribution{
		Name:   "metric-1",
		Values: []float64{3, 4},
		Tags:   []string{"a", "b", "c"},
		Host:   &host2,
	}

	batcher.AddMetric(tm, &dm1)
	batcher.AddMetric(tm, &dm2)

	assert.Equal(t, []float64{1, 2}, dm1.Values)
}

func TestGetMetricSameHost(t *testing.T) {

	tm := time.Now()
	batcher := MakeBatcher(10)

	host := "my-host"

	dm1 := Distribution{
		Name:   "metric-1",
		Values: []float64{1, 2},
		Tags:   []string{"a", "b", "c"},
		Host:   &host,
	}
	dm2 := Distribution{
		Name:   "metric-1",
		Values: []float64{3, 4},
		Tags:   []string{"a", "b", "c"},
		Host:   &host,
	}

	batcher.AddMetric(tm, &dm1)
	batcher.AddMetric(tm, &dm2)

	assert.Equal(t, []float64{1, 2, 3, 4}, dm1.Values)
}

func TestToAPIMetricsSameInterval(t *testing.T) {
	tm := time.Now()
	hostname := "host-1"

	batcher := MakeBatcher(10)
	dm := Distribution{
		Name:   "metric-1",
		Tags:   []string{"a", "b", "c"},
		Host:   &hostname,
		Values: []float64{},
	}

	dm.AddPoint(1)
	dm.AddPoint(2)
	dm.AddPoint(3)

	batcher.AddMetric(tm, &dm)

	floatTime := float64(tm.Unix())
	result := batcher.ToAPIMetrics(tm)
	expected := []APIMetric{
		{
			Name:       "metric-1",
			Host:       &hostname,
			Tags:       []string{"a", "b", "c"},
			MetricType: DistributionType,
			Interval:   nil,
			Points: [][]float64{
				{floatTime, 1}, {floatTime, 2}, {floatTime, 3},
			},
		},
	}

	assert.Equal(t, expected, result)
}
