package client

import (
	"encoding/json"
	"encoding/xml"
	"reflect"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/guidewire-oss/fern-junit-client/pkg/models/fern"
	"github.com/guidewire-oss/fern-junit-client/pkg/models/junit"
	"github.com/guidewire-oss/fern-junit-client/pkg/util"
)

func Test_parseReports(t *testing.T) {
	type args struct {
		testRun     *fern.TestRun
		filePattern string
		tags        string
		verbose     bool
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "combined reports",
			args: args{
				testRun:     &fern.TestRun{},
				filePattern: reportsCombinedPattern,
				tags:        exampleTags,
				verbose:     true,
			},
			wantErr: false,
		},
		{
			name: "failed report",
			args: args{
				testRun:     &fern.TestRun{},
				filePattern: reportFailedPath,
				tags:        exampleTags,
				verbose:     true,
			},
			wantErr: false,
		},
		{
			name: "passed report",
			args: args{
				testRun:     &fern.TestRun{},
				filePattern: reportPassedPath,
				tags:        exampleTags,
				verbose:     true,
			},
			wantErr: false,
		},
		{
			name: "passed report without timestamp",
			args: args{
				testRun:     &fern.TestRun{},
				filePattern: reportPassedWithoutTimestampPath,
				tags:        exampleTags,
				verbose:     true,
			},
			wantErr: false,
		},
		{
			name: "no reports",
			args: args{
				testRun:     &fern.TestRun{},
				filePattern: nonExistentFilePath,
				tags:        exampleTags,
				verbose:     true,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := parseReports(tt.args.testRun, tt.args.filePattern, tt.args.tags, tt.args.verbose); (err != nil) != tt.wantErr {
				t.Errorf("parseReports() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_parseReport(t *testing.T) {
	type args struct {
		filePath string
		tags     string
		verbose  bool
	}
	tests := []struct {
		name    string
		args    args
		want    []fern.SuiteRun
		wantErr bool
	}{
		{
			name: "failed report",
			args: args{
				filePath: reportFailedPath,
				tags:     exampleTags,
				verbose:  true,
			},
			want:    fernTestRunFailed.SuiteRuns,
			wantErr: false,
		},
		{
			name: "passed report",
			args: args{
				filePath: reportPassedPath,
				tags:     exampleTags,
				verbose:  true,
			},
			want:    fernTestRunPassed.SuiteRuns,
			wantErr: false,
		},
		{
			name: "non-existent report",
			args: args{
				filePath: nonExistentFilePath,
				tags:     exampleTags,
				verbose:  true,
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseReport(tt.args.filePath, tt.args.tags, tt.args.verbose)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseReport() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseReport() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_parseTestSuite(t *testing.T) {
	type args struct {
		testSuite junit.TestSuite
		tags      string
		verbose   bool
	}
	util.GlobalClock = util.NewMockClock()
	tests := []struct {
		name         string
		args         args
		wantSuiteRun fern.SuiteRun
		wantErr      bool
	}{
		{
			name: "failed suite",
			args: args{
				testSuite: junitTestSuiteFailed,
				tags:      exampleTags,
				verbose:   true,
			},
			wantSuiteRun: fernTestRunFailed.SuiteRuns[0],
			wantErr:      false,
		},
		{
			name: "passed suite",
			args: args{
				testSuite: junitTestSuitePassed,
				tags:      exampleTags,
				verbose:   true,
			},
			wantSuiteRun: fernTestRunPassed.SuiteRuns[0],
			wantErr:      false,
		},
		{
			name: "empty suite",
			args: args{
				testSuite: junit.TestSuite{},
				tags:      exampleTags,
				verbose:   true,
			},
			wantSuiteRun: fern.SuiteRun{StartTime: util.GlobalClock.Now(), EndTime: util.GlobalClock.Now()},
			wantErr:      true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSuiteRun, err := parseTestSuite(tt.args.testSuite, tt.args.tags, tt.args.verbose)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseTestSuite() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotSuiteRun, tt.wantSuiteRun) {
				t.Errorf("parseTestSuite() = %v, want %v", gotSuiteRun, tt.wantSuiteRun)
			}
		})
	}
}

func Test_getEndTime(t *testing.T) {
	type args struct {
		startTime       time.Time
		durationSeconds string
	}
	tests := []struct {
		name        string
		args        args
		wantEndTime time.Time
		wantErr     bool
	}{
		{
			name: "10 seconds",
			args: args{
				startTime:       time.Unix(0, 0),
				durationSeconds: "10",
			},
			wantEndTime: time.Unix(10, 0),
			wantErr:     false,
		},
		{
			name: "10.5 seconds",
			args: args{
				startTime:       time.Unix(0, 0),
				durationSeconds: "10.5",
			},
			wantEndTime: time.Unix(10, 500000000),
			wantErr:     false,
		},
		{
			name: "invalid duration",
			args: args{
				startTime:       time.Unix(0, 0),
				durationSeconds: "foo",
			},
			wantEndTime: time.Unix(0, 0),
			wantErr:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotEndTime, err := getEndTime(tt.args.startTime, tt.args.durationSeconds)
			if (err != nil) != tt.wantErr {
				t.Errorf("getEndTime() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotEndTime, tt.wantEndTime) {
				t.Errorf("getEndTime() = %v, want %v", gotEndTime, tt.wantEndTime)
			}
		})
	}
}

func Test_convertToTags(t *testing.T) {
	type args struct {
		tagString string
	}
	tests := []struct {
		name     string
		args     args
		wantTags []fern.Tag
	}{
		{
			name: "single tag",
			args: args{
				tagString: "test",
			},
			wantTags: []fern.Tag{
				{
					Name: "test",
				},
			},
		},
		{
			name: "simple string",
			args: args{
				tagString: exampleTags,
			},
			wantTags: []fern.Tag{
				{
					Name: "test",
				},
				{
					Name: "tagtest",
				},
				{
					Name: "9=-+_",
				},
			},
		},
		{
			name: "unicode string",
			args: args{
				tagString: "🤔😯😲🤯,あなたこれを読んだ!",
			},
			wantTags: []fern.Tag{
				{
					Name: "🤔😯😲🤯",
				},
				{
					Name: "あなたこれを読んだ!",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tags := convertToTags(tt.args.tagString)
			if !reflect.DeepEqual(tags, tt.wantTags) {
				t.Errorf("getEndTime() = %v, want %v", tags, tt.wantTags)
			}
		})
	}
}

func Test_parseTestSuite_TestCaseTimestamps(t *testing.T) {
	suite := junit.TestSuite{
		Name:      "pkg",
		Timestamp: "2026-09-29T08:00:00Z",
		Time:      "10",
		TestCases: []junit.TestCase{
			{Name: "TestA", Time: "4", Timestamp: "2026-09-29T08:00:01.5Z"},
			{Name: "TestB", Time: "2", Timestamp: "2026-09-29T08:00:01.5Z"}, // ran alongside TestA
			{Name: "TestC", Time: "1"},                                      // no timestamp: follows TestB
		},
	}
	got, err := parseTestSuite(suite, "", false)
	if err != nil {
		t.Fatal(err)
	}
	at := func(s string) time.Time { v, _ := time.Parse(time.RFC3339Nano, s); return v }
	want := []struct{ start, end time.Time }{
		{at("2026-09-29T08:00:01.5Z"), at("2026-09-29T08:00:05.5Z")},
		{at("2026-09-29T08:00:01.5Z"), at("2026-09-29T08:00:03.5Z")},
		{at("2026-09-29T08:00:03.5Z"), at("2026-09-29T08:00:04.5Z")},
	}
	for i, w := range want {
		s := got.SpecRuns[i]
		if !s.StartTime.Equal(w.start) || !s.EndTime.Equal(w.end) {
			t.Errorf("spec %d (%s): got %s..%s, want %s..%s", i, s.SpecDescription, s.StartTime, s.EndTime, w.start, w.end)
		}
	}
}

// A skip's reason (the t.Skip text gotestsum writes into <skipped message>)
// reaches Fern as the spec run's description, which Fern keeps for every
// status, so a "KNOWN GAP ..." skip can be told from a plain one.
func Test_parseTestSuite_SkipReason(t *testing.T) {
	xmlDoc := `<testsuite name="e2e" timestamp="2026-10-01T08:00:00Z" time="1">
	  <testcase name="TestX/S08_no_handover" time="0.01"><skipped message="=== RUN   TestX/S08_no_handover&#xA;    x_test.go:41: KNOWN GAP ENG-465 S08: no handover route&#xA;--- SKIP: TestX/S08_no_handover (0.01s)"></skipped></testcase>
	  <testcase name="TestX/S09" time="0.01"><skipped>body only</skipped></testcase>
	  <testcase name="TestX/S10" time="0.01"><skipped/></testcase>
	  <testcase name="TestX/S11" time="0.01"><failure message="boom">trace</failure></testcase>
	</testsuite>`
	var suite junit.TestSuite
	if err := xml.Unmarshal([]byte(xmlDoc), &suite); err != nil {
		t.Fatal(err)
	}
	got, err := parseTestSuite(suite, "", false)
	if err != nil {
		t.Fatal(err)
	}
	s := got.SpecRuns
	if s[0].Status != "skipped" || !strings.Contains(s[0].Description, "KNOWN GAP ENG-465 S08: no handover route") || s[0].Message != s[0].Description {
		t.Errorf("skip with message: %+v", s[0])
	}
	if s[1].Description != "body only" {
		t.Errorf("skip with body: %q", s[1].Description)
	}
	if s[2].Status != "skipped" || s[2].Description != "" {
		t.Errorf("bare skip: %+v", s[2])
	}
	if s[3].Status != "failed" || s[3].Message != "boom\ntrace" || s[3].Description != "" {
		t.Errorf("failure: %+v", s[3])
	}
}

func Test_skipReason(t *testing.T) {
	if got := skipReason(junit.Skip{Message: " same ", Content: "same"}); got != "same" {
		t.Errorf("duplicate body: %q", got)
	}
	if got := skipReason(junit.Skip{Message: "a", Content: "b"}); got != "a\nb" {
		t.Errorf("message and body: %q", got)
	}
	long := strings.Repeat("é", maxSkipReason) // 2 bytes each
	got := skipReason(junit.Skip{Message: long})
	if len(got) > maxSkipReason || !utf8.ValidString(got) {
		t.Errorf("long reason: %d bytes, valid %v", len(got), utf8.ValidString(got))
	}
}

func TestParseStepsProperty(t *testing.T) {
	xmlDoc := `<testsuites><testsuite name="pkg" timestamp="2026-10-02T10:00:00Z" time="3">
	<testcase classname="pkg" name="TestA" time="2" timestamp="2026-10-02T10:00:01Z">
	  <properties><property name="other" value="x"/><property name="fern.steps" value='[{"title":"member PTY-1 policy POL-2","kind":"log","mode":"technical","start":"2026-10-02T10:00:01.5Z","duration_ms":null,"status":null,"error":null,"depth":0,"detail":"a_test.go:12"},{"title":"S01 works","kind":"subtest","mode":"technical","start":"2026-10-02T10:00:02Z","duration_ms":300,"status":"passed","error":null,"depth":0,"detail":null}]'/></properties>
	</testcase>
	<testcase classname="pkg" name="TestB" time="1"><properties><property name="fern.steps">[{"title":"open","kind":"step","mode":"plain","depth":0}]</property></properties></testcase>
	<testcase classname="pkg" name="TestC" time="1"><properties><property name="fern.steps" value="not json"/></properties></testcase>
	<testcase classname="pkg" name="TestD" time="1"/>
	</testsuite></testsuites>`
	var suites junit.TestSuites
	if err := xml.Unmarshal([]byte(xmlDoc), &suites); err != nil {
		t.Fatal(err)
	}
	run, err := parseTestSuite(suites.TestSuites[0], "", false)
	if err != nil {
		t.Fatal(err)
	}
	specs := run.SpecRuns
	steps, ok := specs[0].Metadata["steps"].([]interface{})
	if !ok || len(steps) != 2 {
		t.Fatalf("TestA steps = %#v", specs[0].Metadata)
	}
	if s := steps[1].(map[string]interface{}); s["kind"] != "subtest" || s["duration_ms"].(float64) != 300 {
		t.Fatalf("TestA step 2 = %v", s)
	}
	if steps, _ := specs[1].Metadata["steps"].([]interface{}); len(steps) != 1 {
		t.Fatalf("TestB (steps in the element body) = %#v", specs[1].Metadata)
	}
	if specs[2].Metadata != nil || specs[3].Metadata != nil {
		t.Fatalf("bad or no steps must send no metadata: %#v %#v", specs[2].Metadata, specs[3].Metadata)
	}
}

func TestParseStepsCaps(t *testing.T) {
	var b strings.Builder
	b.WriteString("[")
	for i := 0; i < 800; i++ {
		if i > 0 {
			b.WriteString(",")
		}
		b.WriteString(`{"title":"log","kind":"log","depth":0}`)
	}
	b.WriteString("]")
	steps := parseSteps([]junit.Property{{Name: "fern.steps", Value: b.String()}})
	if len(steps) != maxSteps {
		t.Fatalf("%d steps, want %d", len(steps), maxSteps)
	}
	if last := steps[len(steps)-1].(map[string]interface{}); last["title"] != "301 more steps not shown" {
		t.Fatalf("last step %v", last)
	}
	big := `[` + strings.Repeat(`{"title":"`+strings.Repeat("x", 3000)+`","kind":"log"},`, 40) + `{"title":"end","kind":"log"}]`
	steps = parseSteps([]junit.Property{{Name: "fern.steps", Value: big}})
	if raw, _ := json.Marshal(steps); len(raw) > maxStepBytes {
		t.Fatalf("byte cap: %d", len(raw))
	}
}
