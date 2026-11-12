package ui

import (
	"github.com/mRemoteNG/mremoteng-go/internal/profiles"
	"github.com/mRemoteNG/mremoteng-go/internal/transfer"
	"strings"
	"testing"
)

func TestApplyImport_MergeSkipsDuplicatesAndAddsFolders(t *testing.T) {
	existing := sampleProfile(t, "web01", "Production", "SSH", "web01.example.com", 22)
	store := &fakeStore{profiles: []profiles.Profile{existing}, folders: []string{"Production"}}
	s := newTestScreen(t, store)
	incoming := sampleProfile(t, "web01", "Production", "SSH", "other.example.com", 22)
	fresh := sampleProfile(t, "db01", "Production/Data", "SSH", "db01.example.com", 22)
	preview := transfer.Preview{Format: transfer.FormatXML, Profiles: []profiles.Profile{incoming, fresh},
		Folders: []string{"Production", "Production/Data"}}

	s.applyImport(preview, false)

	if len(store.profiles) != 2 {
		t.Fatalf("profiles = %d, want the existing plus the fresh one", len(store.profiles))
	}
	if len(store.folders) != 2 {
		t.Fatalf("folders = %v, want the union", store.folders)
	}
	if !strings.Contains(s.status.Text, "1 connection(s) added") || !strings.Contains(s.status.Text, "1 skipped") {
		t.Errorf("status = %q, want added/skipped counts", s.status.Text)
	}
}

func TestApplyImport_ReplaceDropsExistingConnections(t *testing.T) {
	existing := sampleProfile(t, "old", "", "SSH", "old.example.com", 22)
	store := &fakeStore{profiles: []profiles.Profile{existing}, folders: []string{"Legacy"}}
	s := newTestScreen(t, store)
	preview := transfer.Preview{Format: transfer.FormatCSV, Profiles: []profiles.Profile{
		sampleProfile(t, "new", "Fresh", "RDP", "new.example.com", 3389),
	}, Folders: []string{"Fresh"}}

	s.applyImport(preview, true)

	if len(store.profiles) != 1 || store.profiles[0].Name != "new" {
		t.Fatalf("profiles = %+v, want only the imported one", store.profiles)
	}
	if len(store.folders) != 1 || store.folders[0] != "Fresh" {
		t.Fatalf("folders = %v, want only the imported folder", store.folders)
	}
}

func TestImportSummary_ReportsSkippedAndDropped(t *testing.T) {
	summary := importSummary(transfer.Preview{
		Profiles:    []profiles.Profile{{Name: "a"}},
		Unsupported: 2,
		Dropped:     3,
		Encrypted:   true,
	}, 1)
	for _, want := range []string{"2 row(s) skipped", "3 password(s) were dropped", "protected", "1 already exist"} {
		if !strings.Contains(summary, want) {
			t.Errorf("summary = %q, missing %q", summary, want)
		}
	}
}

func TestUnionFolders_ParentsFirstWithoutDuplicates(t *testing.T) {
	got := unionFolders([]string{"A"}, []string{"A", "A/B", "C"})
	if len(got) != 3 || got[0] != "A" || got[1] != "A/B" || got[2] != "C" {
		t.Fatalf("union = %v", got)
	}
}
