package counter

import (
	"regexp"
	"strings"
	"sync"
	"time"
	
	"github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/luraproject/lura/v2/logging"
)

var (
	lastRequestCount  = map[string]int{}
	lastResponseCount = map[string]int{}
	mu                = new(sync.Mutex)
)
func Points(hostname string, now time.Time, counters map[string]int64, logger logging.Logger, writeApi *api.WriteAPI) {
	requestPoints(hostname, now, counters, logger, writeApi)
	responsePoints(hostname, now, counters, logger, writeApi)
	connectionPoints(hostname, now, counters, logger, writeApi)
}

var (
	requestCounterPattern = `krakend\.proxy\.requests\.layer\.([a-zA-Z]+)\.name\.(.*)\.complete\.(true|false)\.error\.(true|false)`
	requestCounterRegexp  = regexp.MustCompile(requestCounterPattern)

	responseCounterPattern = `krakend\.router\.response\.(.*)\.status\.([\d]{3})\.count`
	responseCounterRegexp  = regexp.MustCompile(responseCounterPattern)
)
func requestPoints(hostname string, now time.Time, counters map[string]int64, logger logging.Logger, writeApi *api.WriteAPI) {
	mu.Lock()
	for k, count := range counters {
		if !requestCounterRegexp.MatchString(k) {
			continue
		}
		params := requestCounterRegexp.FindAllStringSubmatch(k, -1)[0][1:]
		tags := map[string]string{
			"host":     hostname,
			"layer":    params[0],
			"name":     params[1],
			"complete": params[2],
			"error":    params[3],
		}
		last, ok := lastRequestCount[strings.Join(params, ".")]
		if !ok {
			last = 0
		}
		fields := map[string]interface{}{
			"total": int(count),
			"count": int(count) - last,
		}
		countersPoint := influxdb2.NewPoint("requests", tags, fields, now)
		(*writeApi).WritePoint(countersPoint)
	}
	mu.Unlock()
}

func responsePoints(hostname string, now time.Time, counters map[string]int64, logger logging.Logger, writeApi *api.WriteAPI) {
	mu.Lock()
	for k, count := range counters {
		if !responseCounterRegexp.MatchString(k) {
			continue
		}
		params := responseCounterRegexp.FindAllStringSubmatch(k, -1)[0][1:]
		tags := map[string]string{
			"host":   hostname,
			"name":   params[0],
			"status": params[1],
		}
		last, ok := lastResponseCount[strings.Join(params, ".")]
		if !ok {
			last = 0
		}
		fields := map[string]interface{}{
			"total": int(count),
			"count": int(count) - last,
		}
		lastResponseCount[strings.Join(params, ".")] = int(count)

		countersPoint := influxdb2.NewPoint("responses", tags, fields, now)
		(*writeApi).WritePoint(countersPoint)
	}
	mu.Unlock()
}

func connectionPoints(hostname string, now time.Time, counters map[string]int64, logger logging.Logger, writeApi *api.WriteAPI) {

	in := map[string]interface{}{
		"current": int(counters["krakend.router.connected"]),
		"total":   int(counters["krakend.router.connected-total"]),
	}

	incoming := influxdb2.NewPoint("router",map[string]string{"host": hostname, "direction": "in"}, in, now)
	(*writeApi).WritePoint(incoming)

	out := map[string]interface{}{
		"current": int(counters["krakend.router.disconnected"]),
		"total":   int(counters["krakend.router.disconnected-total"]),
	}

	outgoing := influxdb2.NewPoint("router", map[string]string{"host": hostname, "direction": "out"}, out, now)
	(*writeApi).WritePoint(outgoing)

}
