package services

import (
	"strings"
	"testing"
)

func TestParse_StdinYAML(t *testing.T) {
	p := &YAMLJSONParser{Stdin: strings.NewReader("variables:\n  - key: A\n    value: '1'\n")}
	got, err := p.Parse("-")
	if err != nil {
		t.Fatalf("parse stdin: %v", err)
	}
	if len(got.Variables) != 1 || got.Variables[0].Key != "A" || got.Variables[0].Value != "1" {
		t.Fatalf("unexpected: %+v", got.Variables)
	}
}

func TestParse_StdinJSON(t *testing.T) {
	p := &YAMLJSONParser{Stdin: strings.NewReader(`{"variables":[{"key":"B","value":"2"}]}`)}
	got, err := p.Parse("-")
	if err != nil {
		t.Fatalf("parse stdin json: %v", err)
	}
	if len(got.Variables) != 1 || got.Variables[0].Key != "B" {
		t.Fatalf("unexpected: %+v", got.Variables)
	}
}
