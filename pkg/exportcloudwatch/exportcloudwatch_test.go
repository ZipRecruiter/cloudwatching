package exportcloudwatch

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
)

func TestReadMetrics(t *testing.T) {
	scw := stubCloudWatch{}
	period := 5 * time.Minute
	start := time.Now().Add(-2 * period).Truncate(period)
	metricstats := newMetricstats()

	err := ReadMetrics(context.Background(), scw, start, period, metricstats)

	assert.NoError(t, err)
	for _, ms := range metricstats {
		stat := ms.statistic
		gaugeValue := ms.gauge.(*mockGauge).value
		if strings.HasSuffix(stat, "-skip") || strings.HasSuffix(stat, "-skip-prior") {
			assert.Nil(t, gaugeValue, "Gauge for %s was skipped and not reset", stat)
		} else {
			assert.NotNil(t, gaugeValue, "Gauge for %s was set or reset", stat)
			if gaugeValue != nil {
				if strings.HasSuffix(stat, "-skip-zero") {
					assert.Equal(t, float64(0.0), *gaugeValue, "Gauge for %s was skipped and reset to zero", ms.statistic)
				} else if strings.HasSuffix(stat, "-skip-nan") {
					assert.True(t, math.IsNaN(*gaugeValue), "Gauge for %s was skipped and set to NaN", ms.statistic)
				} else {
					assert.Equal(t, float64(1.0), *gaugeValue, "Gauge for %s was not skipped and was set to 1.0", ms.statistic)
				}
			}
		}
	}
}

type unrollTest struct {
	name string
	in   []MetricStat
	out  map[string]MetricStat
}

func TestUnrollMetrics(t *testing.T) {
	tests := []unrollTest{{
		name: "empty",
		in:   []MetricStat{},
		out:  map[string]MetricStat{},
	}, {
		name: "simple",
		in: []MetricStat{
			{
				statistic: "Sum",
				cloudwatchMetric: &types.Metric{
					Dimensions: []types.Dimension{
						{
							Name:  aws.String("QueueName"),
							Value: aws.String("dev-nosensitive-contact-profile-live"),
						},
						{
							Name:  aws.String("QueuePrio"),
							Value: aws.String("7"),
						},
					},
					MetricName: aws.String("ApproximateAgeOfOldestMessage"),
					Namespace:  aws.String("AWS/SQS"),
				},
			},
		},
		out: map[string]MetricStat{"i0": {
			statistic: "Sum",
			cloudwatchMetric: &types.Metric{
				Dimensions: []types.Dimension{
					{
						Name:  aws.String("QueueName"),
						Value: aws.String("dev-nosensitive-contact-profile-live"),
					},
					{
						Name:  aws.String("QueuePrio"),
						Value: aws.String("7"),
					},
				},
				MetricName: aws.String("ApproximateAgeOfOldestMessage"),
				Namespace:  aws.String("AWS/SQS"),
			},
		}},
	}, {
		name: "complex",
		in: []MetricStat{
			{
				statistic: "Maximum",
				cloudwatchMetric: &types.Metric{
					Dimensions: []types.Dimension{
						{
							Name:  aws.String("QueueName"),
							Value: aws.String("dev-nosensitive-contact-profile-live"),
						},
						{
							Name:  aws.String("QueuePrio"),
							Value: aws.String("7"),
						},
					},
					MetricName: aws.String("ApproximateAgeOfOldestMessage"),
					Namespace:  aws.String("AWS/SQS"),
				},
			},
			{
				statistic: "Sum",
				cloudwatchMetric: &types.Metric{
					Dimensions: []types.Dimension{
						{
							Name:  aws.String("QueueName"),
							Value: aws.String("dev-nosensitive-contact-profile-live"),
						},
						{
							Name:  aws.String("QueuePrio"),
							Value: aws.String("7"),
						},
					},
					MetricName: aws.String("ApproximateAgeOfOldestMessage"),
					Namespace:  aws.String("AWS/SQS"),
				},
			},
		},
		out: map[string]MetricStat{
			"i0": {
				statistic: "Maximum",
				cloudwatchMetric: &types.Metric{
					Dimensions: []types.Dimension{
						{
							Name:  aws.String("QueueName"),
							Value: aws.String("dev-nosensitive-contact-profile-live"),
						},
						{
							Name:  aws.String("QueuePrio"),
							Value: aws.String("7"),
						},
					},
					MetricName: aws.String("ApproximateAgeOfOldestMessage"),
					Namespace:  aws.String("AWS/SQS"),
				},
			},
			"i1": {
				statistic: "Sum",
				cloudwatchMetric: &types.Metric{
					Dimensions: []types.Dimension{
						{
							Name:  aws.String("QueueName"),
							Value: aws.String("dev-nosensitive-contact-profile-live"),
						},
						{
							Name:  aws.String("QueuePrio"),
							Value: aws.String("7"),
						},
					},
					MetricName: aws.String("ApproximateAgeOfOldestMessage"),
					Namespace:  aws.String("AWS/SQS"),
				},
			},
		},
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := unrollMetrics(test.in)

			assert.Equal(t, test.out, got)
		})
	}
}

func newMetricstats() map[string]MetricStat {
	// build large set of stats
	numMetrics := 150
	retval := make(map[string]MetricStat, numMetrics)
	for i := 0; i < numMetrics; i++ {
		ms := MetricStat{
			statistic: fmt.Sprintf("stat-%d", i),
			gauge:     &mockGauge{},
		}

		// sprinkle in some that won't return a value and may need to be reset
		switch i {
		case 91:
			ms.statistic = "stat-91-skip"
		case 93:
			ms.statistic = "stat-93-skip-prior"
			ms.statDefault = Prior
		case 145:
			ms.statistic = "stat-145-skip-zero"
			ms.statDefault = Zero
		case 147:
			ms.statistic = "stat-147-skip-nan"
			ms.statDefault = NaN
		}
		retval[strconv.Itoa(i)] = ms
	}

	return retval
}

type stubCloudWatch struct{}

func (scw stubCloudWatch) GetMetricData(_ context.Context, gmdi *cloudwatch.GetMetricDataInput, _ ...func(*cloudwatch.Options)) (*cloudwatch.GetMetricDataOutput, error) {
	// handle in batches of 20 at a time to make sure implementation is properly
	// using the NextToken
	start := 0
	if gmdi.NextToken != nil {
		start, _ = strconv.Atoi(*gmdi.NextToken)
	}
	end := start + 20

	gmdo := cloudwatch.GetMetricDataOutput{
		MetricDataResults: make([]types.MetricDataResult, 0, 20),
	}

	for i := start; i < end && i < len(gmdi.MetricDataQueries); i++ {
		mdq := gmdi.MetricDataQueries[i]
		if strings.Contains(*mdq.MetricStat.Stat, "-skip") {
			continue
		}
		gmdo.MetricDataResults = append(gmdo.MetricDataResults, types.MetricDataResult{
			Id:     mdq.Id,
			Values: []float64{1.0},
		})
	}

	if end < len(gmdi.MetricDataQueries) {
		nt := strconv.Itoa(end)
		gmdo.NextToken = &nt
	}

	return &gmdo, nil
}

// stubMetricLister returns pages of metrics, one page per call, chained by
// NextToken
type stubMetricLister struct {
	pages [][]types.Metric
}

func (s stubMetricLister) ListMetrics(_ context.Context, lmi *cloudwatch.ListMetricsInput, _ ...func(*cloudwatch.Options)) (*cloudwatch.ListMetricsOutput, error) {
	page := 0
	if lmi.NextToken != nil {
		page, _ = strconv.Atoi(*lmi.NextToken)
	}

	lmo := &cloudwatch.ListMetricsOutput{Metrics: s.pages[page]}
	if page+1 < len(s.pages) {
		lmo.NextToken = aws.String(strconv.Itoa(page + 1))
	}

	return lmo, nil
}

func sqsMetric(queueName string) types.Metric {
	return types.Metric{
		Namespace:  aws.String("AWS/SQS"),
		MetricName: aws.String("ApproximateAgeOfOldestMessage"),
		Dimensions: []types.Dimension{{Name: aws.String("QueueName"), Value: aws.String(queueName)}},
	}
}

func TestMetricsToRead(t *testing.T) {
	ec := ExportConfig{
		Namespace:       "AWS/SQS",
		Name:            "ApproximateAgeOfOldestMessage",
		Dimensions:      []string{"QueueName"},
		Statistics:      []string{"Maximum", "Sum"},
		DimensionsMatch: map[string]*regexp.Regexp{"QueueName": regexp.MustCompile("prod")},
		collectors: []*prometheus.GaugeVec{
			prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "max"}, []string{"queue_name"}),
			prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "sum"}, []string{"queue_name"}),
		},
	}
	lister := stubMetricLister{pages: [][]types.Metric{
		{sqsMetric("prod-a"), sqsMetric("dev-a")},
		{sqsMetric("prod-b")},
	}}

	got, err := MetricsToRead(context.Background(), []ExportConfig{ec}, lister)

	assert.NoError(t, err)
	// 2 matching metrics (across both pages) x 2 statistics
	assert.Len(t, got, 4)
	seen := map[string]bool{}
	for _, ms := range got {
		seen[*ms.cloudwatchMetric.Dimensions[0].Value+"/"+ms.statistic] = true
	}
	assert.Equal(t, map[string]bool{
		"prod-a/Maximum": true, "prod-a/Sum": true,
		"prod-b/Maximum": true, "prod-b/Sum": true,
	}, seen)
}

type mockGauge struct {
	value *float64
}

func (g *mockGauge) Desc() *prometheus.Desc {
	return nil
}

func (g *mockGauge) Write(*dto.Metric) error {
	return nil
}

func (g *mockGauge) Describe(chan<- *prometheus.Desc) {
}

func (g *mockGauge) Collect(chan<- prometheus.Metric) {
}

func (g *mockGauge) Set(f float64) {
	g.value = &f
}

func (g *mockGauge) Inc() {
}

func (g *mockGauge) Dec() {
}

func (g *mockGauge) Add(float64) {
}

func (g *mockGauge) Sub(float64) {
}

func (g *mockGauge) SetToCurrentTime() {
}
