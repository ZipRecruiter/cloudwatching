package exportcloudwatch

import (
	"bytes"
	"log"
	"testing"
)

type stringTest struct {
	input          string
	expectedOutput string
}

var stringTests []stringTest = []stringTest{
	{input: "AllErrors", expectedOutput: "all_errors"},
	{input: "2xx", expectedOutput: "2xx"},
	{input: "4xxErrors", expectedOutput: "4xx_errors"},
	{input: "DescribeDeliveryStream.Requests", expectedOutput: "describe_delivery_stream_requests"},
	{input: "DesyncMitigationMode_NonCompliant_Request_Count", expectedOutput: "desync_mitigation_mode_non_compliant_request_count"},
	{input: "ClusterStatus.red", expectedOutput: "cluster_status_red"},
	{input: "ActiveFlowCount_TCP", expectedOutput: "active_flow_count_tcp"},
	{input: "ADAnomalyDetectorsIndexStatusIndexExists", expectedOutput: "ad_anomaly_detectors_index_status_index_exists"},
	{input: "ADAnomalyDetectorsIndexStatus.red", expectedOutput: "ad_anomaly_detectors_index_status_red"},
	{input: "ConsumedLCUs", expectedOutput: "consumed_lcus"},
	{input: "ConsumedLCUs_TCP", expectedOutput: "consumed_lcus_tcp"},
	{input: "AuroraDMLRejectedMasterFull", expectedOutput: "aurora_dml_rejected_master_full"},
	{input: "CPUAllocated", expectedOutput: "cpu_allocated"},
	{input: "dfs.FSNamesystem.PendingReplicationBlocks", expectedOutput: "dfs_fs_namesystem_pending_replication_blocks"},
	{input: "HTTPCode_ELB_5XX", expectedOutput: "http_code_elb_5xx"},
}

type testWriter struct {
	t *testing.T
}

func (tw *testWriter) Write(payload []byte) (int, error) {
	tw.t.Log(string(bytes.TrimSuffix(payload, []byte("\n"))))
	return len(payload), nil
}

func TestCloudWatchToPrometheusName(t *testing.T) {
	for _, test := range stringTests {
		test := test

		t.Run(test.input, func(t *testing.T) {
			tw := &testWriter{t: t}
			log.SetOutput(tw)

			gotOutput := cloudWatchToPrometheusName(test.input)
			if gotOutput != test.expectedOutput {
				t.Fatalf("cloudWatchToPrometheusName(%q) != %q (got %q instead)", test.input, test.expectedOutput, gotOutput)
			}
		})
	}
}
