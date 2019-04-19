package main

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/stretchr/testify/assert"
)

type unrollTest struct {
	name string
	in   []metricStat
	out  map[string]metricStat
}

func TestUnrollMetrics(t *testing.T) {
	tests := []unrollTest{{
		name: "empty",
		in:   []metricStat{},
		out:  map[string]metricStat{},
	}, {
		name: "simple",
		in: []metricStat{
			{
				statistic: "Sum",
				cloudwatchMetric: &cloudwatch.Metric{
					Dimensions: []*cloudwatch.Dimension{
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
		out: map[string]metricStat{"i0": {
			statistic: "Sum",
			cloudwatchMetric: &cloudwatch.Metric{
				Dimensions: []*cloudwatch.Dimension{
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
		in: []metricStat{
			{
				statistic: "Maximum",
				cloudwatchMetric: &cloudwatch.Metric{
					Dimensions: []*cloudwatch.Dimension{
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
				cloudwatchMetric: &cloudwatch.Metric{
					Dimensions: []*cloudwatch.Dimension{
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
		out: map[string]metricStat{
			"i0": {
				statistic: "Maximum",
				cloudwatchMetric: &cloudwatch.Metric{
					Dimensions: []*cloudwatch.Dimension{
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
				cloudwatchMetric: &cloudwatch.Metric{
					Dimensions: []*cloudwatch.Dimension{
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
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := unrollMetrics(test.in)

			assert.Equal(t, test.out, got)
		})
	}
}
