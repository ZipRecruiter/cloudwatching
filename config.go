package main

import (
	"github.com/ZipRecruiter/monitoring--cloudwatch/pkg/exportcloudwatch"
)

type configuration struct {
	Region string
	Debug  bool

	ExportConfigs []exportcloudwatch.ExportConfig
}

func (c *configuration) Validate() error {
	// for _, e := range c.ExportConfigs {
	// 	if err := e.Validate(); err != nil {
	// 		return err
	// 	}
	// }

	return nil
}
