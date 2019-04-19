package main

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
)

func initDependencies(config configuration) (*cloudwatch.CloudWatch, error) {
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(config.Region),
	}))

	return cloudwatch.New(sess), nil
}
