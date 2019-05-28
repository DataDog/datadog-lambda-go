package metrics

import (
	"sync"
	"time"

	"github.com/cenkalti/backoff"
)

type (
	// Processor is used to batch metrics on a background thread, and send them on to a client.
	Processor interface {
		// AddMetric sends a metric to the agent
		AddMetric(metric Metric)
		// StartProcessing begins processing metrics asynchronously
		StartProcessing()
		// FinishProcessing shuts down the agent, and tries to flush any remaining metrics
		FinishProcessing()
	}

	processor struct {
		metricsChan       chan Metric
		timeService       TimeService
		waitGroup         sync.WaitGroup
		batchInterval     float64
		client            Client
		batcher           *Batcher
		shouldRetryOnFail bool
		isProcessing      bool
	}
)

// MakeProcessor creates a new metrics context
func MakeProcessor(client Client, timeService TimeService, batchInterval float64, shouldRetryOnFail bool) Processor {
	batcher := MakeBatcher(batchInterval)

	return &processor{
		metricsChan:       make(chan Metric, 2000),
		batchInterval:     batchInterval,
		waitGroup:         sync.WaitGroup{},
		client:            client,
		batcher:           batcher,
		shouldRetryOnFail: shouldRetryOnFail,
		timeService:       timeService,
		isProcessing:      false,
	}
}

func (p *processor) AddMetric(metric Metric) {
	// We use a large buffer in the metrics channel, to make this operation non-blocking.
	// However, if the channel does fill up, this will become a blocking operation.
	p.metricsChan <- metric
}

func (p *processor) StartProcessing() {
	if !p.isProcessing {
		p.isProcessing = true
		p.waitGroup.Add(1)
		go p.processMetrics()
	}

}

func (p *processor) FinishProcessing() {
	if !p.isProcessing {
		p.StartProcessing()
	}
	// Closes the metrics channel, and waits for the last send to complete
	close(p.metricsChan)
	p.waitGroup.Wait()
}

func (p *processor) processMetrics() {

	ticker := p.timeService.NewTicker(time.Duration(p.batchInterval) * time.Second)

	shouldExit := false
	for !shouldExit {
		shouldProcess := false
		// Batches metrics until timeout is reached
		select {
		case m, ok := <-p.metricsChan:
			if !ok {
				// The channel has now been closed
				shouldProcess = true
				shouldExit = true
			} else {
				p.batcher.AddMetric(p.timeService.Now(), m)
			}
		case <-ticker.C:
			shouldProcess = true
		}
		if shouldProcess {
			if shouldExit && p.shouldRetryOnFail {
				// If we are shutting down, and we just failed to send our last batch, do a retry
				bo := backoff.WithMaxRetries(backoff.NewConstantBackOff(defaultRetryInterval), 2)
				backoff.Retry(p.sendMetricsBatch, bo)
			} else {
				p.sendMetricsBatch()
			}
		}
	}
	ticker.Stop()
	p.waitGroup.Done()
}

func (p *processor) sendMetricsBatch() error {
	mts := p.batcher.ToAPIMetrics(p.timeService.Now())
	if len(mts) > 0 {
		err := p.client.SendMetrics(mts)
		if err != nil {
			return err
		}
		// All the metrics in the batcher were sent successfully,
		// the batcher can now be cleared. If there was an error,
		// the metrics will stay in the batcher and be sent in the next cycle.
		p.batcher = MakeBatcher(p.batchInterval)
	}
	return nil
}
