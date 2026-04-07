package eval

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func sampleReports() map[string]*Report {
	return map[string]*Report{
		"accuracy": {
			Evaluator: "accuracy",
			PassRate:  0.75,
			Metrics:   map[string]float64{"pass_rate": 0.75},
			Failures: []*Failure{
				{Input: "q1", Expected: "yes", Actual: "no", Reason: "mismatch"},
			},
			Timestamp: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		"performance": {
			Evaluator: "performance",
			PassRate:  1.0,
			Metrics:   map[string]float64{"latency_p50_ms": 10, "latency_p99_ms": 50},
			Timestamp: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}
}

func TestWriteJSON_Structure(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteJSON(&buf, sampleReports()); err != nil {
		t.Fatalf("WriteJSON error: %v", err)
	}

	raw := buf.Bytes()
	var parsed map[string]json.RawMessage
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, raw)
	}

	for _, key := range []string{"accuracy", "performance"} {
		if _, ok := parsed[key]; !ok {
			t.Errorf("missing key %q in JSON output", key)
		}
	}
}

func TestWriteJSON_Deterministic(t *testing.T) {
	var buf1, buf2 bytes.Buffer
	reports := sampleReports()
	if err := WriteJSON(&buf1, reports); err != nil {
		t.Fatal(err)
	}
	if err := WriteJSON(&buf2, reports); err != nil {
		t.Fatal(err)
	}
	if buf1.String() != buf2.String() {
		t.Error("WriteJSON output is not deterministic")
	}
}

func TestWriteJUnit_Structure(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteJUnit(&buf, sampleReports(), "my-suite"); err != nil {
		t.Fatalf("WriteJUnit error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "<?xml") {
		t.Error("missing XML declaration")
	}
	if !strings.Contains(out, "<testsuites") {
		t.Error("missing <testsuites> element")
	}
	if !strings.Contains(out, "<testsuite") {
		t.Error("missing <testsuite> element")
	}
	if !strings.Contains(out, "<failure") {
		t.Error("missing <failure> element for failed case")
	}
	if !strings.Contains(out, "accuracy") {
		t.Error("missing evaluator name 'accuracy'")
	}
}

func TestWriteJUnit_Deterministic(t *testing.T) {
	var buf1, buf2 bytes.Buffer
	reports := sampleReports()
	if err := WriteJUnit(&buf1, reports, "suite"); err != nil {
		t.Fatal(err)
	}
	if err := WriteJUnit(&buf2, reports, "suite"); err != nil {
		t.Fatal(err)
	}
	if buf1.String() != buf2.String() {
		t.Error("WriteJUnit output is not deterministic")
	}
}
