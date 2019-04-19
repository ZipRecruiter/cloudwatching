package main

import (
	"regexp"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/stretchr/testify/assert"
)

type includeMetricTest struct {
	name string

	exportConfig
	cloudwatchMetric *cloudwatch.Metric

	result bool
}

func TestIncludeMetric(t *testing.T) {
	tests := []includeMetricTest{
		{
			name:   "Dimensions (missing QueueName)",
			result: false,
			exportConfig: exportConfig{
				Dimensions: []string{"QueueName"},
				Statistics: []string{"Sum"},
			},
			cloudwatchMetric: &cloudwatch.Metric{
				Dimensions: []*cloudwatch.Dimension{{
					Name:  aws.String("Bonk"),
					Value: aws.String("bar"),
				}},
			},
		},
		{
			name:   "Dimensions (extra Bonk)",
			result: false,
			exportConfig: exportConfig{
				Dimensions: []string{"QueueName"},
				Statistics: []string{"Sum"},
			},
			cloudwatchMetric: &cloudwatch.Metric{
				Dimensions: []*cloudwatch.Dimension{{
					Name:  aws.String("Bonk"),
					Value: aws.String("bar"),
				}, {
					Name:  aws.String("QueueName"),
					Value: aws.String("bar"),
				}},
			},
		},
		{
			name:   "!DimensionsMatch",
			result: false,
			exportConfig: exportConfig{
				Dimensions: []string{"QueueName"},
				Statistics: []string{"Sum"},
				dimensionsMatch: map[string]*regexp.Regexp{
					"QueueName": regexp.MustCompile("^foo"),
				},
			},
			cloudwatchMetric: &cloudwatch.Metric{
				Dimensions: []*cloudwatch.Dimension{{
					Name:  aws.String("QueueName"),
					Value: aws.String("bar"),
				}},
			},
		},
		{
			name:   "!DimensionsNoMatch",
			result: false,
			exportConfig: exportConfig{
				Dimensions: []string{"QueueName"},
				Statistics: []string{"Sum"},
				dimensionsNoMatch: map[string]*regexp.Regexp{
					"QueueName": regexp.MustCompile("^foo"),
				},
			},
			cloudwatchMetric: &cloudwatch.Metric{
				Dimensions: []*cloudwatch.Dimension{{
					Name:  aws.String("QueueName"),
					Value: aws.String("foo"),
				}},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.result, includeMetric(test.exportConfig, test.cloudwatchMetric))
		})
	}
}
