package metrics

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetMetricDifferentTagOrder(t *testing.T) {

	tm := time.Now()
	key1 := BatchKey{
		timestamp:  tm,
		metricType: DistributionType,
		name:       "metric-1",
		tags:       []string{"a", "b", "c"},
	}
	key2 := BatchKey{
		timestamp:  tm,
		metricType: DistributionType,
		name:       "metric-1",
		tags:       []string{"c", "b", "a"},
	}
	batcher := MakeBatcher(10)
	dm := Distribution{
		Name: "metric-1",
	}

	batcher.AddMetric(key1, &dm)
	result := batcher.GetMetric(key2)
	assert.Equal(t, &dm, result)
}

func TestGetMetricFailDifferentName(t *testing.T) {

	tm := time.Now()
	key1 := BatchKey{
		timestamp:  tm,
		metricType: DistributionType,
		name:       "metric-1",
		tags:       []string{"a", "b", "c"},
	}
	key2 := BatchKey{
		timestamp:  tm,
		metricType: DistributionType,
		name:       "metric-2",
		tags:       []string{"a", "b", "c"},
	}
	batcher := MakeBatcher(10)
	dm := Distribution{
		Name: "metric-1",
	}

	batcher.AddMetric(key1, &dm)
	result := batcher.GetMetric(key2)
	assert.Nil(t, result)
}

func TestGetMetricFailDifferentHost(t *testing.T) {

	tm := time.Now()
	hostname := "host-1"
	key1 := BatchKey{
		timestamp:  tm,
		metricType: DistributionType,
		name:       "metric-1",
		tags:       []string{"a", "b", "c"},
		host:       &hostname,
	}
	key2 := BatchKey{
		timestamp:  tm,
		metricType: DistributionType,
		name:       "metric-2",
		tags:       []string{"a", "b", "c"},
	}
	batcher := MakeBatcher(10)
	dm := Distribution{
		Name: "metric-1",
	}

	batcher.AddMetric(key1, &dm)
	result := batcher.GetMetric(key2)
	assert.Nil(t, result)
}

func TestGetMetricSameHost(t *testing.T) {

	tm := time.Now()
	hostname := "host-1"
	key1 := BatchKey{
		timestamp:  tm,
		metricType: DistributionType,
		name:       "metric-1",
		tags:       []string{"a", "b", "c"},
		host:       &hostname,
	}
	key2 := BatchKey{
		timestamp:  tm,
		metricType: DistributionType,
		name:       "metric-1",
		tags:       []string{"a", "b", "c"},
		host:       &hostname,
	}
	batcher := MakeBatcher(10)
	dm := Distribution{
		Name: "metric-1",
	}

	batcher.AddMetric(key1, &dm)
	result := batcher.GetMetric(key2)
	assert.Equal(t, &dm, result)
}

func TestFlushSameInterval(t *testing.T) {
	tm := time.Now()
	hostname := "host-1"
	key1 := BatchKey{
		timestamp:  tm,
		metricType: DistributionType,
		name:       "metric-1",
		tags:       []string{"a", "b", "c"},
		host:       &hostname,
	}
	batcher := MakeBatcher(10)
	dm := Distribution{
		Name:   "metric-1",
		Tags:   key1.tags,
		Host:   &hostname,
		Values: []float64{},
	}

	dm.AddPoint(1)
	dm.AddPoint(2)
	dm.AddPoint(3)

	batcher.AddMetric(key1, &dm)

	floatTime := float64(tm.Unix())
	result := batcher.Flush(tm)
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
