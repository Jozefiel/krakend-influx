package histogram

import (
	"regexp"
	"time"

	metrics "github.com/devopsfaith/krakend-metrics/v2"
	"github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/luraproject/lura/v2/logging"
)

func Points(hostname string, now time.Time, histograms map[string]metrics.HistogramData, logger logging.Logger, writeApi *api.WriteAPI) {
	latencyPoints(hostname, now, histograms, logger, writeApi)
	routerPoints(hostname, now, histograms, logger, writeApi)

	// empty data check in functions
	debugPoint(hostname, now, histograms, logger, writeApi)
	runtimePoint(hostname, now, histograms, logger, writeApi)

}

var (
	latencyPattern = `krakend\.proxy\.latency\.layer\.([a-zA-Z]+)\.name\.(.*)\.complete\.(true|false)\.error\.(true|false)`
	latencyRegexp  = regexp.MustCompile(latencyPattern)

	routerPattern = `krakend\.router\.response\.(.*)\.(size|time)`
	routerRegexp  = regexp.MustCompile(routerPattern)
)

func latencyPoints(hostname string, now time.Time, histograms map[string]metrics.HistogramData, logger logging.Logger, writeApi *api.WriteAPI) {
	for k, histogram := range histograms {
		if !latencyRegexp.MatchString(k) {
			continue
		}

		if isEmpty(histogram) {
			continue
		}

		params := latencyRegexp.FindAllStringSubmatch(k, -1)[0][1:]
		tags := map[string]string{
			"host":     hostname,
			"layer":    params[0],
			"name":     params[1],
			"complete": params[2],
			"error":    params[3],
		}

		histogramPoint := influxdb2.NewPoint("requests", tags, newFields(histogram), now)
		(*writeApi).WritePoint(histogramPoint)
	}
}

func routerPoints(hostname string, now time.Time, histograms map[string]metrics.HistogramData, logger logging.Logger, writeApi *api.WriteAPI) {
	for k, histogram := range histograms {
		if !routerRegexp.MatchString(k) {
			continue
		}

		if isEmpty(histogram) {
			continue
		}

		params := routerRegexp.FindAllStringSubmatch(k, -1)[0][1:]
		tags := map[string]string{
			"host": hostname,
			"name": params[0],
		}

		histogramPoint := influxdb2.NewPoint("router.response-"+params[1], tags, newFields(histogram), now)
		(*writeApi).WritePoint(histogramPoint)
	}
}

func debugPoint(hostname string, now time.Time, histograms map[string]metrics.HistogramData, logger logging.Logger, writeApi *api.WriteAPI) {
	hd, ok := histograms["krakend.service.debug.GCStats.Pause"]
	if !ok {
		return
	}
	tags := map[string]string{
		"host": hostname,
	}

	histogramPoint := influxdb2.NewPoint("service.debug.GCStats.Pause", tags, newFields(hd), now)
	(*writeApi).WritePoint(histogramPoint)
}

func runtimePoint(hostname string, now time.Time, histograms map[string]metrics.HistogramData, logger logging.Logger, writeApi *api.WriteAPI) {
	hd, ok := histograms["krakend.service.runtime.MemStats.PauseNs"]
	if !ok {
		return
	}
	tags := map[string]string{
		"host": hostname,
	}

	histogramPoint := influxdb2.NewPoint("service.runtime.MemStats.PauseNs", tags, newFields(hd), now)
	(*writeApi).WritePoint(histogramPoint)
}

func isEmpty(histogram metrics.HistogramData) bool {
	return histogram.Max == 0 && histogram.Min == 0 &&
		histogram.Mean == .0 && histogram.Stddev == .0 && histogram.Variance == 0 &&
		(len(histogram.Percentiles) == 0 ||
			histogram.Percentiles[0] == .0 && histogram.Percentiles[len(histogram.Percentiles)-1] == .0)
}

func newFields(h metrics.HistogramData) map[string]interface{} {
	fields := map[string]interface{}{
		"max":      int(h.Max),
		"min":      int(h.Min),
		"mean":     int(h.Mean),
		"stddev":   int(h.Stddev),
		"variance": int(h.Variance),
	}

	if len(h.Percentiles) != 7 {
		return fields
	}

	fields["p0.1"] = h.Percentiles[0]
	fields["p0.25"] = h.Percentiles[1]
	fields["p0.5"] = h.Percentiles[2]
	fields["p0.75"] = h.Percentiles[3]
	fields["p0.9"] = h.Percentiles[4]
	fields["p0.95"] = h.Percentiles[5]
	fields["p0.99"] = h.Percentiles[6]

	return fields
}
