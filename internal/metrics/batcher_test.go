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
		name:       "metric-2",
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
