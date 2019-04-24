package main

import (
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/prometheus/client_golang/prometheus"
)

var awsRequestSeconds = prometheus.NewSummaryVec(prometheus.SummaryOpts{
	Name: "monitoring_cloudwatch_aws_request_seconds",
	Help: "Duration of requests we've made to the AWS service",
}, []string{"service", "call"})

var awsErrorsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
	Name: "monitoring_cloudwatch_aws_errors_total",
	Help: "Count of errors we've gotten from the AWS service",
}, []string{"service", "call"})

func init() {
	prometheus.MustRegister(awsRequestSeconds, awsErrorsTotal)
}

func initDependencies(config configuration) (*cloudwatch.CloudWatch, error) {
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(config.Region),
	}))

	sess.Handlers.Send.SetBackNamed(request.NamedHandler{
		Name: "Counters",
		Fn: func(r *request.Request) {
			service := r.ClientInfo.ServiceName
			call := r.Operation.Name
			duration := time.Now().Sub(r.AttemptTime).Seconds()
			awsRequestSeconds.WithLabelValues(service, call).Observe(duration)
		},
	})

	sess.Handlers.UnmarshalError.SetBackNamed(request.NamedHandler{
		Name: "Counters",
		Fn: func(r *request.Request) {
			service := r.ClientInfo.ServiceName
			call := r.Operation.Name
			awsErrorsTotal.WithLabelValues(service, call).Inc()
		},
	})

	return cloudwatch.New(sess), nil
}
