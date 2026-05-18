package lynis

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseReport(t *testing.T) {
	p := filepath.Join(t.TempDir(), "report.dat")
	body := `# Lynis Report
warning[]=AUTH-9229|Password file consistency check|description||
suggestion[]=BOOT-5122|Set a password on GRUB|tip||
some_other_field=skip
`
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	findings, err := ParseReport(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 2 {
		t.Fatalf("expected 2 findings, got %d", len(findings))
	}
	if findings[0].ID != "lynis.AUTH-9229" || findings[0].Source != "lynis" {
		t.Fatalf("bad finding %+v", findings[0])
	}
	if findings[1].Title != "Set a password on GRUB" {
		t.Fatalf("bad title %q", findings[1].Title)
	}
}
