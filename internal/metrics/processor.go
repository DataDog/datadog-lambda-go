/*
 * Unless explicitly stated otherwise all files in this repository are licensed
 * under the Apache License Version 2.0.
 *
 * This product includes software developed at Datadog (https://www.datadoghq.com/).
 * Copyright 2019 Datadog, Inc.
 */

package metrics

import (
	"context"
	"sync"
	"time"

	"github.com/cenkalti/backoff"
)

type (
	// Processor is used to batch metrics on a background thread, and send them on to a client periodically.
	Processor interface {
		// AddMetric sends a metric to the agent
		AddMetric(metric Metric)
		// StartProcessing begins processing metrics asynchronously
		StartProcessing()
		// FinishProcessing shuts down the agent, and tries to flush any remaining metrics
		FinishProcessing()
	}

	processor struct {
		context           context.Context
		metricsChan       chan Metric
		timeService       TimeService
		waitGroup         sync.WaitGroup
		batchInterval     time.Duration
		client            Client
		batcher           *Batcher
		shouldRetryOnFail bool
		isProcessing      bool
	}
)

// MakeProcessor creates a new metrics context
func MakeProcessor(ctx context.Context, client Client, timeService TimeService, batchInterval time.Duration, shouldRetryOnFail bool) Processor {
	batcher := MakeBatcher(batchInterval)

	return &processor{
		context:           ctx,
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

	ticker := p.timeService.NewTicker(p.batchInterval)

	doneChan := p.context.Done()
	shouldExit := false
	for !shouldExit {
		shouldSendBatch := false
		// Batches metrics until timeout is reached
		select {
		case <-doneChan:
			// This process is being cancelled by the context,(probably due to a lambda deadline), exit without flushing.
			shouldExit = true
		case m, ok := <-p.metricsChan:
			if !ok {
				// The channel has now been closed
				shouldSendBatch = true
				shouldExit = true
			} else {
				p.batcher.AddMetric(m)
			}
		case <-ticker.C:
			// We are ready to send a batch to our backend
			shouldSendBatch = true
		}
		if shouldSendBatch {
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
	p.isProcessing = false
	p.waitGroup.Done()
}

func (p *processor) sendMetricsBatch() error {
	mts := p.batcher.ToAPIMetrics()
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
