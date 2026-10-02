package junit

import (
	"encoding/xml"
)

type TestSuites struct {
	XMLName    xml.Name    `xml:"testsuites"`
	Name       string      `xml:"name,attr"`
	Time       string      `xml:"time,attr"`
	TestSuites []TestSuite `xml:"testsuite"`
}

type TestSuite struct {
	XMLName   xml.Name   `xml:"testsuite"`
	Name      string     `xml:"name,attr"`
	Tests     int        `xml:"tests,attr"`
	Skipped   int        `xml:"skipped,attr"`
	Failures  int        `xml:"failures,attr"`
	Errors    int        `xml:"errors,attr"`
	Timestamp string     `xml:"timestamp,attr"`
	Time      string     `xml:"time,attr"`
	TestCases []TestCase `xml:"testcase"`
}

type TestCase struct {
	XMLName   xml.Name `xml:"testcase"`
	Name      string   `xml:"name,attr"`
	ClassName string   `xml:"classname,attr"`
	Time      string   `xml:"time,attr"`
	// Timestamp is the case's own start (RFC3339), when the producer knows it.
	// Not in every JUnit dialect; without it cases are laid end to end.
	Timestamp string    `xml:"timestamp,attr"`
	Failures  []Failure `xml:"failure"`
	Errors    []Error   `xml:"error"`
	Skips     []Skip    `xml:"skipped"`
	// Properties are the case's own <properties> (pytest record_property,
	// junit-stamp.py). "fern.steps" carries the case's steps as JSON.
	Properties []Property `xml:"properties>property"`
}

type Property struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value,attr"`
	// Content is the value when it is the element's body instead.
	Content string `xml:",chardata"`
}

type Failure struct {
	XMLName xml.Name `xml:"failure"`
	Message string   `xml:"message,attr"`
	Type    string   `xml:"type,attr"`
	Content string   `xml:",chardata"`
}

type Error struct {
	XMLName xml.Name `xml:"error"`
	Message string   `xml:"message,attr"`
	Type    string   `xml:"type,attr"`
	Content string   `xml:",chardata"`
}

type Skip struct {
	XMLName xml.Name `xml:"skipped"`
	// Message is why the case was skipped (go test / gotestsum put the
	// t.Skip output here, e.g. "KNOWN GAP ENG-465 S08: ...").
	Message string `xml:"message,attr"`
	Content string `xml:",chardata"`
}
