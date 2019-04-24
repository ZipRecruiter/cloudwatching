package main

import (
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type metricStat struct {
	statistic        string
	cloudwatchMetric *cloudwatch.Metric
	gauge            prometheus.Gauge
}

func metricsToRead(c configuration, cw *cloudwatch.CloudWatch) ([]metricStat, error) {
	var metrics []metricStat

	for _, exportConfig := range c.ExportConfigs {
		lmi := &cloudwatch.ListMetricsInput{
			MetricName: aws.String(exportConfig.Name),
			Namespace:  aws.String(exportConfig.Namespace),
		}
		for {
			lmo, err := cw.ListMetrics(lmi)
			if err != nil {
				return nil, errors.Wrap(err, "cloudwatch.ListMetrics")
			}

			for _, metric := range lmo.Metrics {
				if !includeMetric(exportConfig, metric) {
					continue
				}
				for i, s := range exportConfig.Statistics {
					values := make([]string, 0, len(metric.Dimensions))
					for _, v := range metric.Dimensions {
						values = append(values, *v.Value)
					}

					metrics = append(metrics, metricStat{
						statistic:        s,
						cloudwatchMetric: metric,
						gauge:            exportConfig.collectors[i].WithLabelValues(values...),
					})
				}
			}

			if lmo.NextToken != nil {
				lmi.NextToken = lmo.NextToken
			} else {
				break
			}
		}
	}

	return metrics, nil
}

var re = regexp.MustCompile("[A-Z][a-z0-9_]+")

func pascalToUnderScores(in string) string {
	found := re.FindAllString(in, -1)

	ret := strings.ToLower(found[0])
	for _, s := range found[1:] {
		ret += "_" + strings.ToLower(s)
	}

	return ret
}

var cloudwatchGetMetricDataMessagesCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
	Name: "monitoring_cloudwatch_get_metric_data_messages_count",
	Help: "Count of messages we got with code dimension; see https://docs.aws.amazon.com/AmazonCloudWatch/latest/APIReference/API_MessageData.html",
}, []string{"code"})

func getMetricData(cw *cloudwatch.CloudWatch, start, end time.Time, mdq []*cloudwatch.MetricDataQuery, unrolled map[string]metricStat) error {
	gmdi := &cloudwatch.GetMetricDataInput{
		StartTime: aws.Time(start),
		EndTime:   aws.Time(end),

		MetricDataQueries: mdq,
		NextToken:         nil,
	}

	for {
		gmdo, err := cw.GetMetricData(gmdi)
		if err != nil {
			return errors.Wrap(err, "cloudwatch.GetMetricData")
		}
		if len(gmdo.Messages) != 0 {
			for _, m := range gmdo.Messages {
				log.Print("Got messages from cloudwatcn.GetMetricData", "code", *m.Code, "value", *m.Value)
				cloudwatchGetMetricDataMessagesCounter.With(prometheus.Labels{"code": *m.Code}).Inc()
			}
		}

		for _, v := range gmdo.MetricDataResults {
			if len(v.Values) != 0 {
				unrolled[*v.Id].gauge.Set(*v.Values[0])
			}
		}

		if gmdo.NextToken != nil {
			gmdi.NextToken = gmdo.NextToken
		} else {
			break
		}
	}

	return nil
}

func readMetrics(cw *cloudwatch.CloudWatch, start time.Time, period time.Duration, unrolled map[string]metricStat) error {
	end := start.Add(period)

	mdq := make([]*cloudwatch.MetricDataQuery, 0, 100)
	for k, v := range unrolled {
		mdq = append(mdq, &cloudwatch.MetricDataQuery{
			Id: aws.String(k),
			MetricStat: &cloudwatch.MetricStat{
				Metric: v.cloudwatchMetric,
				Period: aws.Int64(int64(period / time.Second)),
				Stat:   aws.String(v.statistic),
			},
			ReturnData: aws.Bool(true),
		})

		if len(mdq) == 100 {
			if err := getMetricData(cw, start, end, mdq, unrolled); err != nil {
				return err
			}

			mdq = make([]*cloudwatch.MetricDataQuery, 0, 100)
		}
	}

	if len(mdq) != 0 {
		if err := getMetricData(cw, start, end, mdq, unrolled); err != nil {
			return err
		}
	}

	return nil
}

// XXX have warden review this?
func handler(c configuration, cw *cloudwatch.CloudWatch, inner http.Handler) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		metrics, err := metricsToRead(c, cw)
		if err != nil {
			rw.WriteHeader(500)
			return
		}

		unrolled := unrollMetrics(metrics)

		period := 60 * time.Second
		start := time.Now().Add(-2 * period).Truncate(time.Minute)
		if err := readMetrics(cw, start, period, unrolled); err != nil {
			rw.WriteHeader(500)
			log.Print(err)
			return
		}

		inner.ServeHTTP(rw, r)
	}
}

func main() {
	c := defaultConfig()
	cw, err := initDependencies(c)
	if err != nil {
		log.Fatal(err)
	}

	log.Print("starting httpserver", "port", "8080")
	http.Handle("/metrics", handler(c, cw, promhttp.Handler()))
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
