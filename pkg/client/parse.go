package client

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/guidewire-oss/fern-junit-client/pkg/models/fern"
	"github.com/guidewire-oss/fern-junit-client/pkg/models/junit"
	"github.com/guidewire-oss/fern-junit-client/pkg/util"
)

func parseReports(testRun *fern.TestRun, filePattern string, tags string, verbose bool) error {
	files, err := filepath.Glob(filePattern)
	if err != nil {
		return fmt.Errorf("failed to parse file pattern %s: %w", filePattern, err)
	}
	if len(files) == 0 {
		return fmt.Errorf("no files found for pattern %s", filePattern)
	}
	for _, file := range files {
		suiteRun, err := parseReport(file, tags, verbose)
		if err != nil {
			return fmt.Errorf("failed to parse report %s: %w", file, err)
		}
		testRun.SuiteRuns = append(testRun.SuiteRuns, suiteRun...)
	}
	for _, suiteRun := range testRun.SuiteRuns {
		// Set testRun.StartTime to the earliest suite start time
		if testRun.StartTime.IsZero() || suiteRun.StartTime.Compare(testRun.StartTime) < 0 {
			testRun.StartTime = suiteRun.StartTime
		}
		// Set testRun.EndTime to the latest suite end time
		if testRun.EndTime.IsZero() || suiteRun.EndTime.Compare(testRun.EndTime) > 0 {
			testRun.EndTime = suiteRun.EndTime
		}
	}
	if verbose {
		log.Default().Printf("TestRun start time: %s\n", testRun.StartTime.String())
		log.Default().Printf("TestRun end time: %s\n", testRun.EndTime.String())
	}
	return nil
}

func parseReport(filePath string, tags string, verbose bool) ([]fern.SuiteRun, error) {
	var testSuites junit.TestSuites
	var testSuite junit.TestSuite
	var suiteRuns []fern.SuiteRun

	if verbose {
		log.Default().Printf("Reading %s\n", filePath)
	}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	byteValue, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	if verbose {
		log.Default().Printf("Unmarshaling %s\n", filePath)
	}

	if err := xml.Unmarshal(byteValue, &testSuites); err != nil {
		if err = xml.Unmarshal(byteValue, &testSuite); err != nil {
			return nil, err
		} else {
			testSuites.TestSuites = append(testSuites.TestSuites, testSuite)
		}
	}

	for _, suite := range testSuites.TestSuites {
		run, err := parseTestSuite(suite, tags, verbose)
		if err != nil {
			return nil, err
		}
		suiteRuns = append(suiteRuns, run)
	}
	return suiteRuns, err
}

func parseTestSuite(testSuite junit.TestSuite, tags string, verbose bool) (suiteRun fern.SuiteRun, err error) {
	if verbose {
		log.Default().Printf("Parsing TestSuite %s\n", testSuite.Name)
	}

	suiteRun.SuiteName = testSuite.Name

	if testSuite.Timestamp == "" {
		suiteRun.StartTime = util.GlobalClock.Now()
	} else {
		suiteRun.StartTime, err = time.Parse(time.RFC3339, testSuite.Timestamp)
		if err != nil {
			// Attempt to parse with a "Z" suffix for UTC time if the initial parsing fails
			suiteRun.StartTime, err = time.Parse(time.RFC3339, testSuite.Timestamp+"Z")
			if err != nil {
				err = fmt.Errorf("failed to parse suite start time: %w", err)
				return
			}
		}
	}

	suiteRun.EndTime, err = getEndTime(suiteRun.StartTime, testSuite.Time)
	if err != nil {
		err = fmt.Errorf("failed to calculate suite end time: %w", err)
		return
	}

	if verbose {
		log.Default().Printf("Resulting SuiteRun: %#v\n", suiteRun)
	}

	startTime := suiteRun.StartTime
	var endTime time.Time
	for _, testCase := range testSuite.TestCases {
		if verbose {
			log.Default().Printf("Parsing TestCase %s\n", testCase.Name)
		}

		status := ""
		message := ""
		description := ""
		if len(testCase.Failures) > 0 {
			status = "failed"
			message = testCase.Failures[0].Message + "\n" + testCase.Failures[0].Content
		} else if len(testCase.Errors) > 0 {
			status = "failed"
			message = testCase.Errors[0].Message + "\n" + testCase.Errors[0].Content
		} else if len(testCase.Skips) > 0 {
			status = "skipped"
			message = skipReason(testCase.Skips[0])
			description = message
		} else {
			status = "passed"
		}

		// A case that carries its own start (parallel subtests, concurrent
		// workers) keeps it; otherwise it follows the previous case.
		if testCase.Timestamp != "" {
			if ts, perr := time.Parse(time.RFC3339Nano, testCase.Timestamp); perr == nil {
				startTime = ts
			}
		}
		endTime, err = getEndTime(startTime, testCase.Time)
		if err != nil {
			err = fmt.Errorf("failed to calculate test end time: %w", err)
			return
		}

		specRun := fern.SpecRun{
			SpecDescription: testCase.Name,
			Status:          status,
			Message:         message,
			Description:     description,
			Tags:            convertToTags(tags),
			StartTime:       startTime,
			EndTime:         endTime,
		}
		if steps := parseSteps(testCase.Properties); steps != nil {
			specRun.Metadata = map[string]interface{}{"steps": steps}
		} else if verbose && hasProperty(testCase.Properties, stepsProperty) {
			log.Default().Printf("TestCase %s: %s is not a JSON list; steps dropped\n", testCase.Name, stepsProperty)
		}
		suiteRun.SpecRuns = append(suiteRun.SpecRuns, specRun)

		startTime = endTime

		if verbose {
			log.Default().Printf("Resulting SpecRun: %#v\n", specRun)
		}
	}
	return
}

func getEndTime(startTime time.Time, durationSeconds string) (endTime time.Time, err error) {
	ms, err := time.ParseDuration(durationSeconds + "s")
	endTime = startTime.Add(ms)
	return
}

func convertToTags(tagString string) (tags []fern.Tag) {
	// strings.Split("", ",") yields [""] - an empty tag name the fern server
	// rejects with a 500 ("tag name cannot be empty"), which fails the whole
	// run ingest for callers that pass no -t at all. Skip empty segments.
	for _, tag := range strings.Split(tagString, ",") {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		tags = append(tags, fern.Tag{Name: tag})
	}
	return
}

// maxSkipReason bounds the skip text sent per spec run.
const maxSkipReason = 2000

// skipReason is why a case was skipped: the message attribute and any body,
// trimmed and joined, at most maxSkipReason bytes (cut on a rune boundary).
func skipReason(s junit.Skip) string {
	parts := []string{}
	for _, p := range []string{strings.TrimSpace(s.Message), strings.TrimSpace(s.Content)} {
		if p != "" && (len(parts) == 0 || parts[0] != p) {
			parts = append(parts, p)
		}
	}
	r := strings.Join(parts, "\n")
	if len(r) > maxSkipReason {
		r = r[:maxSkipReason]
		for len(r) > 0 && !utf8.ValidString(r) {
			r = r[:len(r)-1]
		}
	}
	return r
}

// stepsProperty names the testcase property that carries the case's steps:
// a JSON list of {title, kind, mode, start, duration_ms, status, error, depth,
// detail}, written by the producer (olly's junit-stamp.py from test2json).
const stepsProperty = "fern.steps"

// Step limits per case, as Fern enforces them.
const (
	maxSteps     = 500
	maxStepBytes = 64 * 1024
)

func hasProperty(props []junit.Property, name string) bool {
	for _, p := range props {
		if p.Name == name {
			return true
		}
	}
	return false
}

// parseSteps reads the fern.steps property: nil when absent or not a JSON
// list. Over the limits, the first steps are kept and a last one says how
// many were cut.
func parseSteps(props []junit.Property) []interface{} {
	for _, p := range props {
		if p.Name != stepsProperty {
			continue
		}
		raw := p.Value
		if strings.TrimSpace(raw) == "" {
			raw = p.Content
		}
		var steps []interface{}
		if err := json.Unmarshal([]byte(raw), &steps); err != nil || steps == nil {
			return nil
		}
		kept := make([]interface{}, 0, len(steps))
		size := 2
		for _, s := range steps {
			if len(kept) >= maxSteps-1 && len(steps) > maxSteps {
				break
			}
			b, err := json.Marshal(s)
			if err != nil {
				continue
			}
			if size+len(b)+1 > maxStepBytes-200 {
				break
			}
			size += len(b) + 1
			kept = append(kept, s)
		}
		if n := len(steps) - len(kept); n > 0 {
			kept = append(kept, map[string]interface{}{"title": fmt.Sprintf("%d more steps not shown", n), "kind": "log",
				"mode": "technical", "depth": 0, "start": nil, "duration_ms": nil, "status": nil, "error": nil, "detail": nil})
		}
		return kept
	}
	return nil
}
