package ui

import (
	"testing"

	"fyne.io/fyne/v2/widget"

	"github.com/mRemoteNG/mremoteng-go/internal/connection"
)

func findRow(t *testing.T, p *PropertiesPanel, fieldName string) *propertyRow {
	t.Helper()
	for _, row := range p.rows {
		if row.field.name == fieldName {
			return row
		}
	}
	t.Fatalf("no property row for field %q", fieldName)
	return nil
}

func TestNewPropertiesPanel_BuildsARowPerField(t *testing.T) {
	p := NewPropertiesPanel()

	if len(p.rows) != len(propertyFields) {
		t.Fatalf("row count = %d, want %d", len(p.rows), len(propertyFields))
	}

	// Username is inheritable in the original model; Hostname and Name are
	// not (a hostname or a name rarely makes sense inherited across
	// unrelated connections) — confirmed against
	// internal/connection/inheritance.go's actual field list rather than
	// assumed, since it's easy to guess wrong about which fields the
	// original model considers inheritable.
	if findRow(t, p, "Username").inherit == nil {
		t.Error("Username row has no Inherit checkbox, want one")
	}
	if findRow(t, p, "Hostname").inherit != nil {
		t.Error("Hostname row has an Inherit checkbox, want none (Hostname has no InheritanceFlags field)")
	}
	if findRow(t, p, "Name").inherit != nil {
		t.Error("Name row has an Inherit checkbox, want none (Name has no InheritanceFlags field)")
	}
}

func TestNewPropertiesPanel_BoolFieldUsesCheckWidget(t *testing.T) {
	p := NewPropertiesPanel()
	row := findRow(t, p, "Favorite")
	if _, ok := row.value.(*widget.Check); !ok {
		t.Errorf("Favorite's value widget = %T, want *widget.Check", row.value)
	}
}

func TestNewPropertiesPanel_StringFieldUsesEntryWidget(t *testing.T) {
	p := NewPropertiesPanel()
	row := findRow(t, p, "Hostname")
	if _, ok := row.value.(*widget.Entry); !ok {
		t.Errorf("Hostname's value widget = %T, want *widget.Entry", row.value)
	}
}

// grandchildBelowRoot builds root -> folder -> leaf: the shallowest tree
// where inheritance actually activates, per
// connection.ConnectionInfo.InheritanceActive's documented rule that root
// nodes and root's *direct* children always keep local values regardless
// of their flags.
func grandchildBelowRoot(t *testing.T) (folder *connection.ContainerInfo, leaf *connection.ConnectionInfo) {
	t.Helper()
	root, err := connection.NewRootInfo()
	if err != nil {
		t.Fatalf("NewRootInfo: %v", err)
	}
	folder, err = connection.NewContainerInfo()
	if err != nil {
		t.Fatalf("NewContainerInfo: %v", err)
	}
	if err := root.AddChild(folder); err != nil {
		t.Fatalf("AddChild(folder): %v", err)
	}
	leaf, err = connection.NewConnectionInfo()
	if err != nil {
		t.Fatalf("NewConnectionInfo: %v", err)
	}
	if err := folder.AddChild(leaf); err != nil {
		t.Fatalf("AddChild(leaf): %v", err)
	}
	return folder, leaf
}

func TestPropertiesPanel_SetTarget_PopulatesRawValue(t *testing.T) {
	_, leaf := grandchildBelowRoot(t)
	leaf.Raw.Hostname = "example.com"

	p := NewPropertiesPanel()
	p.SetTarget(leaf)

	entry := findRow(t, p, "Hostname").value.(*widget.Entry)
	if entry.Text != "example.com" {
		t.Errorf("Hostname entry = %q, want %q", entry.Text, "example.com")
	}
	if entry.Disabled() {
		t.Error("Hostname entry is disabled, want enabled (not inherited)")
	}
}

func TestPropertiesPanel_SetTarget_ShowsEffectiveValueWhenInherited(t *testing.T) {
	folder, leaf := grandchildBelowRoot(t)
	folder.Base().Raw.Username = "parent-user"
	leaf.Inheritance.Username = true

	p := NewPropertiesPanel()
	p.SetTarget(leaf)

	row := findRow(t, p, "Username")
	entry := row.value.(*widget.Entry)
	if entry.Text != "parent-user" {
		t.Errorf("Username entry = %q, want the inherited %q", entry.Text, "parent-user")
	}
	if !entry.Disabled() {
		t.Error("Username entry is enabled, want disabled (inherited fields aren't directly editable)")
	}
	if !row.inherit.Checked {
		t.Error("Inherit checkbox is unchecked, want checked")
	}
}

func TestPropertiesPanel_EditingValue_CommitsToRaw(t *testing.T) {
	_, leaf := grandchildBelowRoot(t)

	p := NewPropertiesPanel()
	p.SetTarget(leaf)

	entry := findRow(t, p, "Hostname").value.(*widget.Entry)
	entry.SetText("new-host")

	if leaf.Raw.Hostname != "new-host" {
		t.Errorf("Raw.Hostname = %q, want %q", leaf.Raw.Hostname, "new-host")
	}
}

func TestPropertiesPanel_TogglingInherit_UpdatesFlagsAndDisplay(t *testing.T) {
	folder, leaf := grandchildBelowRoot(t)
	folder.Base().Raw.Username = "parent-user"
	leaf.Raw.Username = "local-user"

	p := NewPropertiesPanel()
	p.SetTarget(leaf)

	row := findRow(t, p, "Username")
	row.inherit.SetChecked(true)

	if !leaf.Inheritance.Username {
		t.Error("Inheritance.Username = false, want true after checking Inherit")
	}
	entry := row.value.(*widget.Entry)
	if entry.Text != "parent-user" {
		t.Errorf("Username entry after inheriting = %q, want %q", entry.Text, "parent-user")
	}
	// The local override must survive underneath, unmodified, so
	// unchecking Inherit later would restore it rather than losing it.
	if leaf.Raw.Username != "local-user" {
		t.Errorf("Raw.Username was overwritten to %q, want it preserved as %q", leaf.Raw.Username, "local-user")
	}
}

func TestPropertiesPanel_IntField_ParsesAndCommits(t *testing.T) {
	_, leaf := grandchildBelowRoot(t)

	p := NewPropertiesPanel()
	p.SetTarget(leaf)

	entry := findRow(t, p, "Port").value.(*widget.Entry)
	entry.SetText("2222")

	if leaf.Raw.Port != 2222 {
		t.Errorf("Raw.Port = %d, want 2222", leaf.Raw.Port)
	}
}

func TestPropertiesPanel_IntField_InvalidInputLeavesValueUnchanged(t *testing.T) {
	_, leaf := grandchildBelowRoot(t)
	leaf.Raw.Port = 22

	p := NewPropertiesPanel()
	p.SetTarget(leaf)

	entry := findRow(t, p, "Port").value.(*widget.Entry)
	entry.SetText("not-a-number")

	if leaf.Raw.Port != 22 {
		t.Errorf("Raw.Port = %d after invalid input, want it unchanged at 22", leaf.Raw.Port)
	}
}

func TestPropertiesPanel_SetTarget_Nil_ClearsAndDisablesRows(t *testing.T) {
	_, leaf := grandchildBelowRoot(t)
	leaf.Raw.Hostname = "example.com"

	p := NewPropertiesPanel()
	p.SetTarget(leaf)
	p.SetTarget(nil)

	entry := findRow(t, p, "Hostname").value.(*widget.Entry)
	if entry.Text != "" {
		t.Errorf("Hostname entry after SetTarget(nil) = %q, want empty", entry.Text)
	}
	if !entry.Disabled() {
		t.Error("Hostname entry after SetTarget(nil) is enabled, want disabled")
	}
}
