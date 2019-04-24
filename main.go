package main

import (
	"log"
	"net/http"
	"time"

	"github.com/ZipRecruiter/monitoring--cloudwatch/pkg/exportcloudwatch"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Version is the GIT_SHA used to build
var Version string

var metrics map[string]exportcloudwatch.MetricStat

var listMetricsSleep = prometheus.NewSummary(prometheus.SummaryOpts{
	Name: "monitoring_cloudwatch_list_metrics_sleep",
	Help: "Amount of time we are going to sleep between updating our metrics list",
})

func init() {
	prometheus.MustRegister(listMetricsSleep)
}

func handler(c configuration, cw *cloudwatch.CloudWatch, inner http.Handler) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		period := 60 * time.Second
		start := time.Now().Add(-2 * period).Truncate(time.Minute)
		if err := exportcloudwatch.ReadMetrics(cw, start, period, metrics); err != nil {
			rw.WriteHeader(500)
			log.Print(err)
			return
		}

		inner.ServeHTTP(rw, r)
	}
}

func sleepRange(got, min, max time.Duration) time.Duration {
	if got < min {
		return min
	}

	if got > max {
		return max
	}

	return got
}

func main() {
	var c configuration
	// if err := config.Read(&c); err != nil {
	// 	log.Err(context.Background(), logger, err)
	// 	os.Exit(1)
	// }

	cw, err := initDependencies(c)
	if err != nil {
		log.Fatal(err)
	}

	var listMetricsDuration time.Duration

	start := time.Now()
	metrics, err = exportcloudwatch.MetricsToRead(c.ExportConfigs, cw)
	if err != nil {
		log.Fatal(err)
	}
	listMetricsDuration = time.Now().Sub(start)

	go func() {
		for {
			duration := sleepRange(10*listMetricsDuration, 5*time.Minute, time.Hour)

			listMetricsSleep.Observe(duration.Seconds())
			time.Sleep(duration)

			start := time.Now()
			metrics, err = exportcloudwatch.MetricsToRead(c.ExportConfigs, cw)
			if err != nil {
				log.Fatal(err)
			}
			listMetricsDuration = time.Now().Sub(start)
		}
	}()

	log.Print("starting httpserver", "port", "8080")
	http.Handle("/metrics", handler(c, cw, promhttp.Handler()))
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
