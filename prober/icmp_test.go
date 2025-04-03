package prober

import (
	"context"
	"github.com/prometheus/blackbox_exporter/config"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/promslog"
	"net"
	"testing"
	"time"
)

func TestICMPSourceAddress(t *testing.T) {
	type testdata struct {
		sourceAddr    string // 设置的源地址
		shouldSucceed bool   // 是否通信失败
	}

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		t.Fatalf("Failed to get local addresses: %v", err)
	}
	var validAddr string
	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if ok && !ipNet.IP.IsLoopback() && ipNet.IP.To4() != nil {
			validAddr = ipNet.IP.String()
			break
		}
	}
	testcases := map[string]testdata{
		"validAddr": {
			sourceAddr:    validAddr, // 使用第一个非回环地址
			shouldSucceed: true,
		},
		"invalidAddr": {
			sourceAddr:    "11.11.11.1", // 使用一个无效的地址
			shouldSucceed: false,
		},
	}

	for i, test := range testcases {
		//ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//}))
		//defer ts.Close()
		registry := prometheus.NewRegistry()
		testCTX, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		testCTX = context.WithValue(testCTX, SourceAddressKey, test.sourceAddr)
		result := ProbeICMP(testCTX, "baidu.com", config.Module{Prober: "icmp", Timeout: time.Second, ICMP: config.ICMPProbe{IPProtocolFallback: true}}, registry, promslog.NewNopLogger())
		// test.shouldSucceed = true, result = false
		if test.shouldSucceed && !result {
			t.Fatalf("Test %s had unexpected result: %s, should success but fail", i, test.sourceAddr)
		}
		// test.shouldSucceed = false, result = true
		if !test.shouldSucceed && result {
			t.Fatalf("Test %s had unexpected result: %s, should fail but success", i, test.sourceAddr)
		}
		mfs, err := registry.Gather()
		if err != nil {
			t.Fatal(err)
		}
		boolToFloat := func(v bool) float64 {
			if v {
				return 1
			}
			return 0
		}
		expectedResults := map[string]float64{
			"probe_failed_due_to_source": boolToFloat(!test.shouldSucceed),
		}
		checkRegistryResults(expectedResults, mfs, t)
	}
}
