package services

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/AxeForging/yankrun/domain"

	"gopkg.in/yaml.v3"
)

type ReplacementParser interface {
	Parse(filePath string) (domain.InputReplacement, error)
}

type YAMLJSONParser struct {
	FileSystem FileSystem
	// Stdin is the reader used when the input path is "-". It defaults to
	// os.Stdin; tests override it.
	Stdin io.Reader
}

func (p *YAMLJSONParser) Parse(filePath string) (domain.InputReplacement, error) {
	var patterns domain.InputReplacement

	// "-" reads values from stdin so agents can pipe values without a temp
	// file. Format is auto-detected (YAML is a JSON superset).
	if filePath == "-" {
		in := p.Stdin
		if in == nil {
			in = os.Stdin
		}
		data, err := io.ReadAll(in)
		if err != nil {
			return patterns, err
		}
		if err := yaml.Unmarshal(data, &patterns); err != nil {
			return patterns, fmt.Errorf("parse stdin values: %w", err)
		}
		return patterns, nil
	}

	data, err := p.FileSystem.ReadFile(filePath)
	if err != nil {
		return patterns, err
	}

	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".json":
		err = json.Unmarshal(data, &patterns)
	case ".yaml", ".yml":
		err = yaml.Unmarshal(data, &patterns)
	default:
		return patterns, fmt.Errorf("unsupported file format: %s", ext)
	}

	if err != nil {
		return patterns, err
	}

	return patterns, nil
}
