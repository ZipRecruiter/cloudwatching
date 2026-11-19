package exportcloudwatch

import (
	"regexp"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/stretchr/testify/assert"
)

type includeMetricTest struct {
	name string

	ExportConfig
	cloudwatchMetric *types.Metric

	result bool
}

func TestIncludeMetric(t *testing.T) {
	tests := []includeMetricTest{
		{
			name:   "Dimensions (missing QueueName)",
			result: false,
			ExportConfig: ExportConfig{
				Dimensions: []string{"QueueName"},
				Statistics: []string{"Sum"},
			},
			cloudwatchMetric: &types.Metric{
				Dimensions: []types.Dimension{{
					Name:  aws.String("Bonk"),
					Value: aws.String("bar"),
				}},
			},
		},
		{
			name:   "Dimensions (extra Bonk)",
			result: false,
			ExportConfig: ExportConfig{
				Dimensions: []string{"QueueName"},
				Statistics: []string{"Sum"},
			},
			cloudwatchMetric: &types.Metric{
				Dimensions: []types.Dimension{{
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
			ExportConfig: ExportConfig{
				Dimensions: []string{"QueueName"},
				Statistics: []string{"Sum"},
				DimensionsMatch: map[string]*regexp.Regexp{
					"QueueName": regexp.MustCompile("^foo"),
				},
			},
			cloudwatchMetric: &types.Metric{
				Dimensions: []types.Dimension{{
					Name:  aws.String("QueueName"),
					Value: aws.String("bar"),
				}},
			},
		},
		{
			name:   "!DimensionsNoMatch",
			result: false,
			ExportConfig: ExportConfig{
				Dimensions: []string{"QueueName"},
				Statistics: []string{"Sum"},
				DimensionsNoMatch: map[string]*regexp.Regexp{
					"QueueName": regexp.MustCompile("^foo"),
				},
			},
			cloudwatchMetric: &types.Metric{
				Dimensions: []types.Dimension{{
					Name:  aws.String("QueueName"),
					Value: aws.String("foo"),
				}},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.result, includeMetric(test.ExportConfig, test.cloudwatchMetric))
		})
	}
}
