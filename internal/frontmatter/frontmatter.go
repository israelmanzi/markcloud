package frontmatter

import (
	"bytes"
	"fmt"

	"gopkg.in/yaml.v3"
)

type Metadata struct {
	Tags   []string `yaml:"tags"`
	Public bool     `yaml:"public"`
}

var separator = []byte("---")

func Parse(data []byte) (*Metadata, []byte, error) {
	meta := &Metadata{}

	trimmed := bytes.TrimSpace(data)
	if !bytes.HasPrefix(trimmed, separator) {
		return meta, data, nil
	}

	rest := trimmed[len(separator):]
	end := bytes.Index(rest, separator)
	if end == -1 {
		return meta, data, nil
	}

	yamlBlock := rest[:end]
	body := bytes.TrimSpace(rest[end+len(separator):])

	if err := yaml.Unmarshal(yamlBlock, meta); err != nil {
		return nil, nil, fmt.Errorf("parsing frontmatter: %w", err)
	}

	if meta.Tags == nil {
		meta.Tags = []string{}
	}

	return meta, body, nil
}

func Serialize(meta *Metadata, body []byte) []byte {
	var buf bytes.Buffer
	buf.Write(separator)
	buf.WriteByte('\n')

	yamlBytes, _ := yaml.Marshal(meta)
	buf.Write(yamlBytes)

	buf.Write(separator)
	buf.WriteByte('\n')
	buf.WriteByte('\n')
	buf.Write(body)

	return buf.Bytes()
}
