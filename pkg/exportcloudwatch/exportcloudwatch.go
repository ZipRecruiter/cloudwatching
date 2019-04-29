// Package exportcloudwatch exports AWS CloudWatch metrics as prometheus metrics.
//
// To use this package
//
//   1. create one or more ExportConfigs
//   2. call Validate() on each of them
//   3. store the result of MetricsToRead
//   4. call ReadMetrics
package exportcloudwatch

import (
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
)

var cloudwatchGetMetricDataMessagesCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
	Name: "monitoring_cloudwatch_get_metric_data_messages_count",
	Help: "Count of messages we got with code dimension; see https://docs.aws.amazon.com/AmazonCloudWatch/latest/APIReference/API_MessageData.html",
}, []string{"code"})

// MetricStat is a specific statistic for a *cloudwatch.Metric with a related,
// registered *prometheus.Gauge
type MetricStat struct {
	statistic        string
	cloudwatchMetric *cloudwatch.Metric
	measure          func(float64)
}

func getMetricData(cw *cloudwatch.CloudWatch, start, end time.Time, mdq []*cloudwatch.MetricDataQuery, unrolled map[string]MetricStat) error {
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
				unrolled[*v.Id].measure(*v.Values[0])
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

// ReadMetrics pulls metrics for the passed time over the period of duration
// into the metricstats map.
func ReadMetrics(cw *cloudwatch.CloudWatch, start time.Time, period time.Duration, metricstats map[string]MetricStat) error {
	end := start.Add(period)

	mdq := make([]*cloudwatch.MetricDataQuery, 0, 100)
	for k, v := range metricstats {
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
			if err := getMetricData(cw, start, end, mdq, metricstats); err != nil {
				return err
			}

			mdq = make([]*cloudwatch.MetricDataQuery, 0, 100)
		}
	}

	if len(mdq) != 0 {
		if err := getMetricData(cw, start, end, mdq, metricstats); err != nil {
			return err
		}
	}

	return nil
}

// MetricsToRead returns a map of MetricStats that match the criteria expressed
// in the ExportConfigs.
func MetricsToRead(ec []ExportConfig, cw *cloudwatch.CloudWatch) (map[string]MetricStat, error) {
	ms, err := metricsToRead(ec, cw)
	if err != nil {
		return nil, err
	}

	return unrollMetrics(ms), nil
}

type sortableDimensions []*cloudwatch.Dimension

func (s sortableDimensions) Len() int { return len(s) }

func (s sortableDimensions) Less(i, j int) bool { return *s[i].Name < *s[j].Name }

func (s sortableDimensions) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

func metricsToRead(ec []ExportConfig, cw *cloudwatch.CloudWatch) ([]MetricStat, error) {
	var metrics []MetricStat

	for _, exportConfig := range ec {
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
				sort.Sort(sortableDimensions(metric.Dimensions))

				for i, s := range exportConfig.Statistics {
					values := make([]string, 0, len(metric.Dimensions))
					for _, v := range metric.Dimensions {
						values = append(values, *v.Value)
					}

					var measure func(float64)

					gauge := exportConfig.Collectors[i].WithLabelValues(values...)
					if len(exportConfig.Transform) == 0 || exportConfig.Transform[i] == nil {
						measure = func(v float64) { gauge.Set(v) }
					} else {
						transform := exportConfig.Transform[i]
						measure = func(v float64) {
							v = transform(v)
							gauge.Set(v)
						}
					}
					metrics = append(metrics, MetricStat{
						statistic:        s,
						cloudwatchMetric: metric,
						measure:          measure,
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

func unrollMetrics(ms []MetricStat) map[string]MetricStat {
	ret := make(map[string]MetricStat, len(ms))

	for i, m := range ms {
		ret[fmt.Sprintf("i%d", i)] = m
	}

	return ret
}
