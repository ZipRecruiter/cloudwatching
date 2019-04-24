package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type testSleepRange struct {
	name                  string
	got, min, max, expect time.Duration
}

func TestSleepRange(t *testing.T) {
	tests := []testSleepRange{
		{
			name:   "basic",
			got:    10 * (time.Minute*5 + time.Second),
			min:    time.Minute * 5,
			max:    time.Hour,
			expect: 10 * (time.Minute*5 + time.Second),
		},
		{
			name:   "max",
			got:    10 * (time.Minute*6 + time.Second),
			min:    time.Minute * 5,
			max:    time.Hour,
			expect: time.Hour,
		},
		{
			name:   "min",
			got:    10 * (time.Second * 6),
			min:    time.Minute * 5,
			max:    time.Hour,
			expect: time.Minute * 5,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expect, sleepRange(test.got, test.min, test.max))
		})
	}
}
