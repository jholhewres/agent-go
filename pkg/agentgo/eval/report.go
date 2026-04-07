package eval

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"sort"
)

// WriteJSON writes all reports as a deterministic JSON object to w.
// Keys are sorted alphabetically.
func WriteJSON(w io.Writer, reports map[string]*Report) error {
	// Sort keys for deterministic output.
	keys := make([]string, 0, len(reports))
	for k := range reports {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	ordered := make([]json.RawMessage, 0, len(keys))
	for _, k := range keys {
		entry, err := json.Marshal(reports[k])
		if err != nil {
			return fmt.Errorf("marshal report %q: %w", k, err)
		}
		ordered = append(ordered, entry)
	}

	// Build a map with sorted keys by marshalling pairs.
	obj := make(map[string]json.RawMessage, len(keys))
	for i, k := range keys {
		obj[k] = ordered[i]
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(obj)
}

// junitTestSuites is the root XML element.
type junitTestSuites struct {
	XMLName    xml.Name         `xml:"testsuites"`
	Name       string           `xml:"name,attr"`
	TestSuites []junitTestSuite `xml:"testsuite"`
}

type junitTestSuite struct {
	XMLName   xml.Name        `xml:"testsuite"`
	Name      string          `xml:"name,attr"`
	Tests     int             `xml:"tests,attr"`
	Failures  int             `xml:"failures,attr"`
	TestCases []junitTestCase `xml:"testcase"`
}

type junitTestCase struct {
	XMLName xml.Name      `xml:"testcase"`
	Name    string        `xml:"name,attr"`
	Failure *junitFailure `xml:"failure,omitempty"`
}

type junitFailure struct {
	XMLName xml.Name `xml:"failure"`
	Message string   `xml:"message,attr"`
	Text    string   `xml:",chardata"`
}

// WriteJUnit writes all reports as JUnit XML to w, suitable for CI tools.
func WriteJUnit(w io.Writer, reports map[string]*Report, suiteName string) error {
	// Sort keys for deterministic output.
	keys := make([]string, 0, len(reports))
	for k := range reports {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	root := junitTestSuites{Name: suiteName}

	for _, k := range keys {
		rep := reports[k]
		suite := junitTestSuite{
			Name:  rep.Evaluator,
			Tests: len(rep.Failures) + int(rep.PassRate*float64(len(rep.Failures)+1)),
		}

		// Each failure becomes a testcase with a failure element.
		for _, f := range rep.Failures {
			suite.Failures++
			suite.TestCases = append(suite.TestCases, junitTestCase{
				Name: fmt.Sprintf("%s/%s", rep.Evaluator, f.Input),
				Failure: &junitFailure{
					Message: f.Reason,
					Text:    fmt.Sprintf("expected: %s\nactual: %s", f.Expected, f.Actual),
				},
			})
		}

		// Add a pass testcase to represent the overall result.
		passCase := junitTestCase{Name: fmt.Sprintf("%s/pass_rate=%.2f", rep.Evaluator, rep.PassRate)}
		suite.TestCases = append(suite.TestCases, passCase)
		suite.Tests = len(suite.TestCases)

		root.TestSuites = append(root.TestSuites, suite)
	}

	if _, err := fmt.Fprintf(w, xml.Header); err != nil {
		return err
	}
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	if err := enc.Encode(root); err != nil {
		return err
	}
	return enc.Flush()
}
