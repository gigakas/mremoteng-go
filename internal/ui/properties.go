<<<<<<< HEAD
=======
// Package ui's PropertiesPanel is a reflection-driven editor for
// connection.ConnectionValues, matching every field to its
// connection.InheritanceFlags counterpart by name. Hand-writing a widget
// binding for each of the ~75 inheritable fields (see
// blueprint/phase-3-ui.md: "3.4 must expose per-field inheritance toggles
// exactly like the original property grid") would be enormous, repetitive
// and drift-prone as the model grows; reflection keeps this one generic
// implementation in sync with internal/connection automatically. This
// mirrors how the original C# app's own property grid worked too (.NET's
// PropertyGrid control is itself reflection-based) — not a novel choice
// for this kind of UI.
>>>>>>> 876e6198187aa7ca8eeacb02d85b84d8328d4e68
package ui

import (
	"fmt"
<<<<<<< HEAD
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/mRemoteNG/mremoteng-go/internal/profiles"
)

// propertiesPanel is the bottom "Config" area of the mRemoteNG layout. It
// edits the selected connection; folders are structural and read-only here.
type propertiesPanel struct {
	box    *fyne.Container
	title  *widget.Label
	form   fyne.CanvasObject
	name   *widget.Entry
	folder *widget.Entry
	proto  *widget.Select
	host   *widget.Entry
	port   *widget.Entry
	user   *widget.Entry

	inheritUser *widget.Check
	inheritPort *widget.Check

	save   *widget.Button
	revert *widget.Button
	delete *widget.Button
}

func newPropertiesPanel(s *screen) *propertiesPanel {
	p := &propertiesPanel{}

	p.title = widget.NewLabelWithStyle("Connection details", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	p.name = widget.NewEntry()
	p.name.SetPlaceHolder("Production web server")
	p.folder = widget.NewEntry()
	p.folder.SetPlaceHolder("Production/Web")
	p.proto = widget.NewSelect([]string{
		string(profiles.SSH), string(profiles.RDP), string(profiles.VNC),
		string(profiles.TELNET), string(profiles.RAW), string(profiles.RLOGIN), string(profiles.HTTP),
	}, s.protocolChanged)
	p.host = widget.NewEntry()
	p.host.SetPlaceHolder("server.example.com")
	p.port = widget.NewEntry()
	p.port.SetPlaceHolder("22")
	p.user = widget.NewEntry()
	p.user.SetPlaceHolder("deploy")
	p.inheritUser = widget.NewCheck("From folder", nil)
	p.inheritPort = widget.NewCheck("From folder", nil)
	p.inheritUser.OnChanged = func(bool) { s.refreshInheritance() }
	p.inheritPort.OnChanged = func(bool) { s.refreshInheritance() }

	p.save = widget.NewButtonWithIcon("Save", theme.DocumentSaveIcon(), s.save)
	p.revert = widget.NewButtonWithIcon("Revert", theme.CancelIcon(), s.revert)
	p.delete = widget.NewButtonWithIcon("", theme.DeleteIcon(), s.requestDelete)

	// Delete lives next to the connection name so the toolbar stays free of
	// write actions, matching how mRemoteNG ties object actions to the object.
	nameRow := container.NewBorder(nil, nil, nil, p.delete, p.name)

	// One form, one field per row: a single form layout aligns every label and
	// field, while two side-by-side forms computed their columns independently
	// and looked ragged.
	p.form = widget.NewForm(
		widget.NewFormItem("Name", nameRow),
		widget.NewFormItem("Folder", p.folder),
		widget.NewFormItem("Protocol", p.proto),
		widget.NewFormItem("Host", p.host),
		widget.NewFormItem("Port", p.port),
		widget.NewFormItem("Username", container.NewBorder(nil, nil, nil, p.inheritUser, p.user)),
		widget.NewFormItem("Port inherited", p.inheritPort),
	)
	buttons := container.NewHBox(p.save, p.revert)
	p.box = container.NewBorder(
		container.NewVBox(p.title, widget.NewSeparator()),
		nil, nil, nil,
		container.NewBorder(nil, container.NewPadded(buttons), nil, nil, container.NewPadded(p.form)),
	)
	return p
}

func (p *propertiesPanel) populate(profile profiles.Profile) {
	p.inheritUser.SetChecked(profile.InheritUsername)
	p.inheritPort.SetChecked(profile.InheritPort)
	p.name.SetText(profile.Name)
	p.folder.SetText(profile.Folder)
	p.proto.SetSelected(string(profile.Protocol))
	p.host.SetText(profile.Host)
	p.port.SetText(strconv.Itoa(profile.Port))
	p.user.SetText(profile.Username)
}

func (p *propertiesPanel) rawForm(id string, protocol profiles.Protocol) profiles.Profile {
	port, err := strconv.Atoi(strings.TrimSpace(p.port.Text))
	if err != nil {
		port = 0
	}
	return profiles.Profile{
		ID:              id,
		Name:            strings.TrimSpace(p.name.Text),
		Folder:          strings.TrimSpace(p.folder.Text),
		Protocol:        protocol,
		Host:            strings.TrimSpace(p.host.Text),
		Port:            port,
		Username:        strings.TrimSpace(p.user.Text),
		InheritUsername: p.inheritUser.Checked,
		InheritPort:     p.inheritPort.Checked,
	}
}

func (p *propertiesPanel) formProfile(id string, protocol profiles.Protocol) (profiles.Profile, error) {
	if _, err := strconv.Atoi(strings.TrimSpace(p.port.Text)); err != nil {
		return profiles.Profile{}, fmt.Errorf("Port must be a whole number between 1 and 65535.")
	}
	profile := p.rawForm(id, protocol)
	if err := profile.Validate(); err != nil {
		return profiles.Profile{}, err
	}
	return profile, nil
}

func (p *propertiesPanel) setEnabled(enabled bool) {
	for _, button := range []*widget.Button{p.save, p.delete} {
		if enabled {
			button.Enable()
			continue
		}
		button.Disable()
	}
}

// welcomeContent is the start page shown when no session tab is open.
func welcomeContent() fyne.CanvasObject {
	heading := widget.NewLabelWithStyle("Welcome to mRemoteNG Go", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	intro := widget.NewLabel("A Go implementation of the mRemoteNG connection manager, built for Linux first.")
	intro.Alignment = fyne.TextAlignCenter
	steps := widget.NewLabel("1. Choose File > New connection (or the New button).\n" +
		"2. Fill in name, optional folder, protocol, host and port in the Config panel.\n" +
		"3. Choose Save: the connection appears in the tree and is stored locally.\n" +
		"4. Select it in the tree to open its session tab.")
	steps.Alignment = fyne.TextAlignLeading
	notice := widget.NewLabel("Session support is not implemented yet: this preview manages connection profiles only. " +
		"SSH and RDP sessions are the next delivery stage.")
	notice.Wrapping = fyne.TextWrapWord
	notice.Importance = widget.WarningImportance

	card := widget.NewCard("Getting started", "", container.NewVBox(steps, notice))
	return container.NewBorder(
		container.NewVBox(container.NewPadded(heading), intro, widget.NewSeparator()),
		nil, nil, nil,
		container.NewPadded(card),
	)
=======
	"reflect"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/mRemoteNG/mremoteng-go/internal/connection"
)

// propertyField describes one editable field, resolved once at package
// init by matching connection.ConnectionValues' and
// connection.InheritanceFlags' field names — not per PropertiesPanel
// instance, since the model's shape is fixed at compile time.
type propertyField struct {
	name       string
	index      int // field index into ConnectionValues
	inheritIdx int // field index into InheritanceFlags, or -1 if this field has no inheritance flag
	kind       reflect.Kind
}

var propertyFields = buildPropertyFields()

func buildPropertyFields() []propertyField {
	valuesType := reflect.TypeOf(connection.ConnectionValues{})
	flagsType := reflect.TypeOf(connection.InheritanceFlags{})

	flagIndex := make(map[string]int, flagsType.NumField())
	for i := 0; i < flagsType.NumField(); i++ {
		flagIndex[flagsType.Field(i).Name] = i
	}

	fields := make([]propertyField, 0, valuesType.NumField())
	for i := 0; i < valuesType.NumField(); i++ {
		f := valuesType.Field(i)
		inheritIdx := -1
		if idx, ok := flagIndex[f.Name]; ok {
			inheritIdx = idx
		}
		fields = append(fields, propertyField{
			name:       f.Name,
			index:      i,
			inheritIdx: inheritIdx,
			kind:       f.Type.Kind(),
		})
	}
	// Deterministic order: NumField order is already declaration order,
	// which is already grouped by category (Display/Connection/
	// Protocol/...) in model.go — sorting by name would scramble that
	// grouping, so this intentionally leaves declaration order alone
	// rather than re-sorting alphabetically.
	return fields
}

// propertyRow is one rendered field: name, its value editor, and (for
// fields the original model makes inheritable) an "Inherit" checkbox.
type propertyRow struct {
	field   propertyField
	value   fyne.CanvasObject // *widget.Entry or *widget.Check, depending on field.kind
	inherit *widget.Check     // nil if field.inheritIdx == -1
}

// PropertiesPanel edits one connection.Node's local values, with
// per-field inheritance toggles. v1 renders every field as a flat list in
// model declaration order (see buildPropertyFields) rather than the
// original app's categorized/tabbed grid — a real gap, not attempted here
// given this phase's stated visual-verification limitation makes
// iterating on a multi-tab layout hard to get right blind; a flat
// scrollable list is at least fully functional. Every field is a plain
// text Entry (numeric fields parse/reformat as integers, bool fields use
// a Check) rather than the enum-aware dropdowns a polished version would
// have — also deferred, same reasoning.
type PropertiesPanel struct {
	widget.BaseWidget

	scroll *container.Scroll
	rows   []*propertyRow

	node connection.Node

	// refreshing guards against a feedback loop: refresh calls
	// widget.Entry.SetText/widget.Check.SetChecked to display the current
	// value, but Fyne's SetText/SetChecked fire OnChanged the same as a
	// real user edit would — without this guard, refresh's own
	// programmatic update would re-trigger commit, which would write
	// whatever refresh just *displayed* (e.g. an inherited value) back
	// into Raw, silently overwriting a real local override. Found by a
	// failing test (TestPropertiesPanel_TogglingInherit_UpdatesFlagsAndDisplay),
	// not spotted by inspection.
	refreshing bool
}

// NewPropertiesPanel builds an empty panel. Call SetTarget to point it at
// a connection.Node.
func NewPropertiesPanel() *PropertiesPanel {
	p := &PropertiesPanel{}
	p.buildRows()
	p.ExtendBaseWidget(p)
	return p
}

func (p *PropertiesPanel) buildRows() {
	form := container.New(layout.NewFormLayout())
	p.rows = make([]*propertyRow, 0, len(propertyFields))

	for _, f := range propertyFields {
		row := &propertyRow{field: f}

		switch f.kind {
		case reflect.Bool:
			check := widget.NewCheck("", func(bool) { p.commit(row) })
			row.value = check
		default:
			entry := widget.NewEntry()
			entry.OnChanged = func(string) { p.commit(row) }
			row.value = entry
		}

		form.Add(widget.NewLabel(f.name))
		if f.inheritIdx >= 0 {
			inherit := widget.NewCheck("Inherit", func(bool) { p.commit(row) })
			row.inherit = inherit
			form.Add(container.NewBorder(nil, nil, nil, inherit, row.value))
		} else {
			form.Add(row.value)
		}

		p.rows = append(p.rows, row)
	}

	p.scroll = container.NewVScroll(form)
}

// CreateRenderer implements fyne.Widget.
func (p *PropertiesPanel) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(p.scroll)
}

// SetTarget points the panel at node (a connection or container — both
// embed the same ConnectionValues/InheritanceFlags shape via
// connection.ConnectionInfo) and (re)populates every row from it. Pass
// nil to clear the panel.
func (p *PropertiesPanel) SetTarget(node connection.Node) {
	p.node = node
	p.refresh()
}

// refresh repaints every row's displayed value from the current target:
// the raw (locally-set) value when inheritance is off for that field, the
// resolved effective value when it's on — matching how the original
// property grid greys out and shows the inherited value rather than the
// local one, without actually overwriting the local value underneath.
func (p *PropertiesPanel) refresh() {
	p.refreshing = true
	defer func() { p.refreshing = false }()

	if p.node == nil {
		for _, row := range p.rows {
			setFieldText(row.value, "")
			enableField(row.value, false)
			if row.inherit != nil {
				row.inherit.SetChecked(false)
				row.inherit.Disable()
			}
		}
		return
	}

	info := p.node.Base()
	raw := reflect.ValueOf(info.Raw)
	effective := reflect.ValueOf(info.Effective())
	flags := reflect.ValueOf(info.Inheritance)

	for _, row := range p.rows {
		inherited := row.field.inheritIdx >= 0 && flags.Field(row.field.inheritIdx).Bool()
		if row.inherit != nil {
			row.inherit.SetChecked(inherited)
			row.inherit.Enable()
		}

		shown := raw.Field(row.field.index)
		if inherited {
			shown = effective.Field(row.field.index)
		}
		setFieldValue(row.value, shown)
		enableField(row.value, !inherited)
	}
}

// commit writes row's currently-displayed value back into the target
// node — the raw value if present (a widget.Entry/Check always holds
// something typeable/checkable even while showing an inherited value;
// see the note below) and the inheritance flag if this field has one.
//
// Editing a field currently showing its inherited value would, taken
// literally, overwrite Raw with whatever the effective value happened to
// be — but the value widget is disabled while inherited (see refresh), so
// OnChanged/toggled can't actually fire from user interaction in that
// state. commit only runs from real callbacks, so this is safe without
// needing an extra "was this inherited" guard here too.
func (p *PropertiesPanel) commit(row *propertyRow) {
	if p.node == nil || p.refreshing {
		return
	}
	info := p.node.Base()

	if row.inherit != nil {
		flags := reflect.ValueOf(&info.Inheritance).Elem()
		flags.Field(row.field.inheritIdx).SetBool(row.inherit.Checked)
	}

	raw := reflect.ValueOf(&info.Raw).Elem()
	target := raw.Field(row.field.index)
	if err := setReflectFromWidget(target, row.value); err != nil {
		// Malformed numeric input, most likely — leave the underlying
		// value untouched rather than panic or silently zero it.
		return
	}

	p.refresh()
}

func setFieldValue(obj fyne.CanvasObject, v reflect.Value) {
	switch w := obj.(type) {
	case *widget.Check:
		w.SetChecked(v.Bool())
	case *widget.Entry:
		w.SetText(fmt.Sprint(v.Interface()))
	}
}

func setFieldText(obj fyne.CanvasObject, s string) {
	if entry, ok := obj.(*widget.Entry); ok {
		entry.SetText(s)
	}
}

func enableField(obj fyne.CanvasObject, enabled bool) {
	type disabler interface {
		Enable()
		Disable()
	}
	d, ok := obj.(disabler)
	if !ok {
		return
	}
	if enabled {
		d.Enable()
	} else {
		d.Disable()
	}
}

func setReflectFromWidget(target reflect.Value, obj fyne.CanvasObject) error {
	switch w := obj.(type) {
	case *widget.Check:
		target.SetBool(w.Checked)
		return nil
	case *widget.Entry:
		switch target.Kind() {
		case reflect.String:
			target.SetString(w.Text)
			return nil
		case reflect.Int:
			n, err := strconv.Atoi(w.Text)
			if err != nil {
				return err
			}
			target.SetInt(int64(n))
			return nil
		}
	}
	return fmt.Errorf("ui: unsupported field kind %s", target.Kind())
>>>>>>> 876e6198187aa7ca8eeacb02d85b84d8328d4e68
}
