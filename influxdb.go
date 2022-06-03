package influxdb2

import (
	"context"
	"os"
	"time"

	ginmetrics "github.com/devopsfaith/krakend-metrics/v2/gin"
    "github.com/influxdata/influxdb-client-go/v2"
	"github.com/luraproject/lura/v2/config"
	"github.com/luraproject/lura/v2/logging"

	"github.com/devopsfaith/krakend-influx2/v2/counter"
	"github.com/devopsfaith/krakend-influx2/v2/gauge"
	"github.com/devopsfaith/krakend-influx2/v2/histogram"
)

const Namespace = "github_com/devopsfaith/krakend-influx2"
const logPrefix = "[SERVICE: Influx2]"

type clientWrapper struct {
	client 	 	influxdb2.Client
	collector   *ginmetrics.Metrics
	logger      logging.Logger
	org			string
	bucket	 	string
}

func New(ctx context.Context, extraConfig config.ExtraConfig, metricsCollector *ginmetrics.Metrics, logger logging.Logger) error {
	
	logger.Debug(logPrefix, "Parsing influx config client")

	cfg, ok := configGetter(extraConfig).(influx2Config)
	if !ok {
		return ErrNoConfig
	}

	logger.Debug(logPrefix, "Creating client")

    client := influxdb2.NewClientWithOptions(
		cfg.address,
		cfg.token,
		influxdb2.DefaultOptions().SetBatchSize(50),
	)

	go func() {
		pingDuration, pingMsg := client.Ping(ctx)
		logger.Debug(logPrefix, "Ping results: duration =", pingDuration, "msg =", pingMsg)
	}()

	t := time.NewTicker(cfg.ttl)

	cw := clientWrapper{
		client: 	  client,
		collector:    metricsCollector,
		logger:       logger,
		bucket:       cfg.bucket,
		org:          cfg.org,
	}

	go cw.keepUpdated(ctx, t.C)

	logger.Debug(logPrefix, "Client up and running")

	return nil
}

func (cw clientWrapper) keepUpdated(ctx context.Context, ticker <-chan time.Time) {

	hostname, err := os.Hostname()
	if err != nil {
		cw.logger.Error("influxdb resolving the local hostname:", err.Error())
	}

    writeAPI :=cw.client.WriteAPI(
		cw.org,
		cw.bucket,
	)

    errorsCh := writeAPI.Errors()
    // Create go proc for reading and logging errors
    go func() {
        for err := range errorsCh {
			cw.logger.Debug(logPrefix, err.Error() )
        }
    }()

	for {
		select {
		case <-ticker:
		case <-ctx.Done():
			return
		}

		cw.logger.Debug(logPrefix, "Preparing data points to send")

		snapshot := cw.collector.Snapshot()

		if shouldSendPoints := len(snapshot.Counters) > 0 || len(snapshot.Gauges) > 0; !shouldSendPoints {
			cw.logger.Debug(logPrefix, "No metrics to send")
			continue
		}

		now := time.Now()

		gauge.Points(hostname, now, snapshot.Gauges, cw.logger, &writeAPI)
		counter.Points(hostname, now, snapshot.Counters, cw.logger, &writeAPI)
		histogram.Points(hostname, now, snapshot.Histograms, cw.logger, &writeAPI)
		writeAPI.Flush()

	}
}
