package bdu

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"mephi_vkr_asoc/services/reference-data-service/internal/models"
)

const miniVULXML = `<?xml version="1.0" encoding="utf-8"?>
<vulnerabilities>
<vul><identifier>BDU:2099-00001</identifier><name>t1</name><description>d1</description><vul_state>x</vul_state></vul>
<vul><identifier>BDU:2099-00002</identifier><name>t2</name><description>d2</description><vul_state>y</vul_state></vul>
</vulnerabilities>
`

func TestPlainVULXMLStreaming(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	p := filepath.Join(dir, "vulxml.xml")
	if err := os.WriteFile(p, []byte(miniVULXML), 0o644); err != nil {
		t.Fatal(err)
	}
	b := &BulkImporter{BatchSize: 10}
	var seen int
	err := b.streamVulXMLFromPlainFile(context.Background(), p, func(batch []models.SourceRecord) error {
		for _, rec := range batch {
			switch rec.ExternalID {
			case "BDU:2099-00001", "BDU:2099-00002":
			default:
				t.Errorf("unexpected id %q", rec.ExternalID)
			}
		}
		seen += len(batch)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if seen != 2 {
		t.Fatalf("expected 2 records, got %d", seen)
	}
}

func TestIsPlainVulXMLFile(t *testing.T) {
	t.Parallel()
	if !isPlainVulXMLFile("/tmp/vulxml.xml") {
		t.Fatal("expected .xml")
	}
	if isPlainVulXMLFile("/tmp/vulxml.zip") {
		t.Fatal("zip must not be plain xml")
	}
}
