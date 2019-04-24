package exportcloudwatch

import (
	"regexp"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

type validateTest struct {
	name string

	in, out []ExportConfig

	err error
}

func xxxAxeCollectors(c []ExportConfig) {
	for i := range c {
		c[i].collectors = nil
	}
}

func TestValidate(t *testing.T) {
	tests := []validateTest{
		{
			name: "basic",

			in: []ExportConfig{
				{
					Namespace:  "AWS/SQS",
					Name:       "ApproximateAgeOfOldestMessage",
					Dimensions: []string{"QueueName", "Alpha"},
					Statistics: []string{"Maximum"},
					DimensionsMatch: map[string]string{
						"QueueName": "^foo",
					},
					DimensionsNoMatch: map[string]string{
						"QueueName": "^bar",
					},
				},
			},

			out: []ExportConfig{
				{
					Namespace:  "AWS/SQS",
					Name:       "ApproximateAgeOfOldestMessage",
					Dimensions: []string{"Alpha", "QueueName"},
					Statistics: []string{"Maximum"},
					DimensionsMatch: map[string]string{
						"QueueName": "^foo",
					},
					DimensionsNoMatch: map[string]string{
						"QueueName": "^bar",
					},
					dimensionsMatch: map[string]*regexp.Regexp{
						"QueueName": regexp.MustCompile("^foo"),
					},
					dimensionsNoMatch: map[string]*regexp.Regexp{
						"QueueName": regexp.MustCompile("^bar"),
					},
				},
			},
		},
		{
			name: "Unknown Match Dimension",

			in: []ExportConfig{
				{
					Namespace:  "AWS/SQS",
					Name:       "ApproximateAgeOfOldestMessage",
					Statistics: []string{"Maximum"},
					Dimensions: []string{"QueueName", "Alpha"},
					DimensionsMatch: map[string]string{
						"TableName": "^foo",
					},
				},
			},

			err: errors.New("DimensionsMatch name not in Dimensions"),
		},
		{
			name: "Unknown NoMatch Dimension",

			in: []ExportConfig{
				{
					Namespace:  "AWS/SQS",
					Name:       "ApproximateAgeOfOldestMessage",
					Statistics: []string{"Maximum"},
					Dimensions: []string{"QueueName", "Alpha"},
					DimensionsNoMatch: map[string]string{
						"TableName": "^foo",
					},
				},
			},

			err: errors.New("DimensionsNoMatch name not in Dimensions"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var err error
			for i := range test.in {
				err = test.in[i].Validate()
				if err != nil && test.err == nil {
					t.Fatal("Couldn't validate supposedly correct config: " + err.Error())
				}
			}

			xxxAxeCollectors(test.in)
			if err != nil || test.err != nil {
				assert.Equal(t, test.err.Error(), err.Error())
				return
			}

			assert.Equal(t, test.out, test.in, "config was correctly modified")
		})
	}
}
