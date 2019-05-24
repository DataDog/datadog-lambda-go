package metrics

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

type (
	// Batcher batches
	Batcher struct {
		metrics       map[string]Metric
		batchInterval float64
	}
	// BatchKey identifies a batch of metrics
	BatchKey struct {
		timestamp  time.Time
		metricType MetricType
		name       string
		tags       []string
		host       *string
	}
)

// MakeBatcher creates a new batcher object
func MakeBatcher(batchInterval float64) *Batcher {
	return &Batcher{
		batchInterval: batchInterval,
		metrics:       map[string]Metric{},
	}
}

// GetMetric gets an existing metric
func (b *Batcher) GetMetric(bk BatchKey) Metric {
	sk := b.getStringKey(bk)
	return b.metrics[sk]
}

// AddMetric adds a point to a given metric
func (b *Batcher) AddMetric(bk BatchKey, metric Metric) {
	sk := b.getStringKey(bk)
	b.metrics[sk] = metric
}

// Flush converts the current batch of metrics into API metrics
func (b *Batcher) Flush(timestamp time.Time) []APIMetric {

	ar := []APIMetric{}
	interval := time.Duration(0) // TODO Get actual interval

	for _, metric := range b.metrics {
		values := metric.ToAPIMetric(timestamp, interval)
		for _, val := range values {
			ar = append(ar, val)
		}
	}
	b.metrics = map[string]Metric{}

	return ar
}

func (b *Batcher) getInterval(timestamp time.Time) float64 {
	return float64(timestamp.Unix()) - math.Mod(float64(timestamp.Unix()), b.batchInterval)
}

func (b *Batcher) getStringKey(bk BatchKey) string {
	interval := b.getInterval(bk.timestamp)
	tagKey := getTagKey(bk.tags)

	if bk.host != nil {
		return fmt.Sprintf("(%g)-(%s)-(%s)-(%s)-(%s)", interval, bk.metricType, bk.name, tagKey, *bk.host)
	}
	return fmt.Sprintf("(%g)-(%s)-(%s)-(%s)", interval, bk.metricType, bk.name, tagKey)
}

func getTagKey(tags []string) string {
	sortedTags := make([]string, len(tags))
	copy(sortedTags, tags)
	sort.Strings(sortedTags)
	return strings.Join(sortedTags, ":")
}
