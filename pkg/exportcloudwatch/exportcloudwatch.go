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
	"context"
	"fmt"
	"log"
	"math"
	"sort"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
)

var cloudwatchGetMetricDataMessagesCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
	Name: "monitoring_cloudwatch_get_metric_data_messages_count",
	Help: "Count of messages we got with code dimension; see https://docs.aws.amazon.com/AmazonCloudWatch/latest/APIReference/API_MessageData.html",
}, []string{"code"})

// MetricStat is a specific statistic for a *types.Metric with a related,
// registered *prometheus.Gauge
type MetricStat struct {
	statistic        string
	cloudwatchMetric *types.Metric
	gauge            prometheus.Gauge
	statDefault      StatDefaultType
}

func getMetricData(ctx context.Context, cw cloudwatch.GetMetricDataAPIClient, start, end time.Time, mdq []types.MetricDataQuery, unrolled map[string]MetricStat, seen map[string]struct{}) error {
	gmdi := &cloudwatch.GetMetricDataInput{
		StartTime: aws.Time(start),
		EndTime:   aws.Time(end),

		MetricDataQueries: mdq,
	}

	p := cloudwatch.NewGetMetricDataPaginator(cw, gmdi)
	for p.HasMorePages() {
		gmdo, err := p.NextPage(ctx)
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
				unrolled[*v.Id].gauge.Set(v.Values[0])
				seen[*v.Id] = struct{}{}
			}
		}
	}

	return nil
}

// ReadMetrics pulls metrics for the passed time over the period of duration
// into the metricstats map.
func ReadMetrics(ctx context.Context, cw cloudwatch.GetMetricDataAPIClient, start time.Time, period time.Duration, metricstats map[string]MetricStat) error {
	end := start.Add(period)

	seen := make(map[string]struct{}, len(metricstats))
	mdq := make([]types.MetricDataQuery, 0, 100)
	for k, v := range metricstats {
		mdq = append(mdq, types.MetricDataQuery{
			Id: aws.String(k),
			MetricStat: &types.MetricStat{
				Metric: v.cloudwatchMetric,
				Period: aws.Int32(int32(period / time.Second)),
				Stat:   aws.String(v.statistic),
			},
			ReturnData: aws.Bool(true),
		})

		if len(mdq) == 100 {
			if err := getMetricData(ctx, cw, start, end, mdq, metricstats, seen); err != nil {
				return err
			}

			mdq = make([]types.MetricDataQuery, 0, 100)
		}
	}

	if len(mdq) != 0 {
		if err := getMetricData(ctx, cw, start, end, mdq, metricstats, seen); err != nil {
			return err
		}
	}

	// set default values for stat according to config for stats no longer seen
	for k, ms := range metricstats {
		if _, ok := seen[k]; ok {
			continue
		}
		if ms.statDefault == Zero {
			ms.gauge.Set(0)
		} else if ms.statDefault == NaN {
			ms.gauge.Set(math.NaN())
		}
	}

	return nil
}

// MetricsToRead returns a map of MetricStats that match the criteria expressed
// in the ExportConfigs.
func MetricsToRead(ctx context.Context, ec []ExportConfig, cw cloudwatch.ListMetricsAPIClient) (map[string]MetricStat, error) {
	ms, err := metricsToRead(ctx, ec, cw)
	if err != nil {
		return nil, err
	}

	return unrollMetrics(ms), nil
}

type sortableDimensions []types.Dimension

func (s sortableDimensions) Len() int { return len(s) }

func (s sortableDimensions) Less(i, j int) bool { return *s[i].Name < *s[j].Name }

func (s sortableDimensions) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

func metricsToRead(ctx context.Context, ec []ExportConfig, cw cloudwatch.ListMetricsAPIClient) ([]MetricStat, error) {
	var metrics []MetricStat

	for _, exportConfig := range ec {
		lmi := &cloudwatch.ListMetricsInput{
			MetricName: aws.String(exportConfig.Name),
			Namespace:  aws.String(exportConfig.Namespace),
		}
		p := cloudwatch.NewListMetricsPaginator(cw, lmi)
		for p.HasMorePages() {
			lmo, err := p.NextPage(ctx)
			if err != nil {
				return nil, errors.Wrap(err, "cloudwatch.ListMetrics")
			}

			for j := range lmo.Metrics {
				metric := &lmo.Metrics[j]
				if !includeMetric(exportConfig, metric) {
					continue
				}
				sort.Sort(sortableDimensions(metric.Dimensions))

				for i, s := range exportConfig.Statistics {
					values := make([]string, 0, len(metric.Dimensions))
					for _, v := range metric.Dimensions {
						values = append(values, *v.Value)
					}

					metrics = append(metrics, MetricStat{
						statistic:        s,
						cloudwatchMetric: metric,
						gauge:            exportConfig.collectors[i].WithLabelValues(values...),
						statDefault:      exportConfig.StatDefault,
					})
				}
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
