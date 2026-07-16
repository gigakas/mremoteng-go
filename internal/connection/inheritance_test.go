package connection

import (
	"errors"
	"reflect"
	"testing"
)

func TestEffective_InheritanceOff_ReturnsLocalValues(t *testing.T) {
	parent := testContainer(t, "parent")
	child := testConnection(t, "child")
	parent.Base().Raw.Username = "parent-user"
	child.Raw.Username = "local-user"
	if err := parent.AddChild(child); err != nil {
		t.Fatal(err)
	}

	if got := child.Effective().Username; got != "local-user" {
		t.Errorf("Username = %q, want local-user", got)
	}
}

func TestEffective_EnabledFields_InheritByType(t *testing.T) {
	parent := testContainer(t, "parent")
	child := testConnection(t, "child")
	parent.Base().Raw.Username = "parent-user"
	parent.Base().Raw.Port = 2222
	parent.Base().Raw.UseCredSsp = false
	parent.Base().Raw.Protocol = ProtocolSSH2
	child.Raw.Username = "local-user"
	child.Raw.Port = 22
	child.Raw.UseCredSsp = true
	child.Raw.Protocol = ProtocolRDP
	child.Inheritance.Username = true
	child.Inheritance.Port = true
	child.Inheritance.UseCredSsp = true
	child.Inheritance.Protocol = true
	if err := parent.AddChild(child); err != nil {
		t.Fatal(err)
	}

	got := child.Effective()
	if got.Username != "parent-user" || got.Port != 2222 || got.UseCredSsp || got.Protocol != ProtocolSSH2 {
		t.Errorf("effective values = user %q, port %d, credssp %v, protocol %q", got.Username, got.Port, got.UseCredSsp, got.Protocol)
	}
	if child.Raw.Username != "local-user" || child.Raw.Port != 22 {
		t.Error("Effective mutated Raw values")
	}
}

func TestEffective_MultiLevelChain_UsesParentsEffectiveValue(t *testing.T) {
	root := testRoot(t, "root")
	grandparent := testContainer(t, "grandparent")
	parent := testContainer(t, "parent")
	child := testConnection(t, "child")
	grandparent.Base().Raw.Username = "grandparent-user"
	parent.Base().Raw.Username = "parent-user"
	child.Raw.Username = "child-user"
	parent.Base().Inheritance.Username = true
	child.Inheritance.Username = true
	if err := root.AddChild(grandparent); err != nil {
		t.Fatal(err)
	}
	if err := grandparent.AddChild(parent); err != nil {
		t.Fatal(err)
	}
	if err := parent.AddChild(child); err != nil {
		t.Fatal(err)
	}

	if got := child.Effective().Username; got != "grandparent-user" {
		t.Errorf("Username = %q, want grandparent-user", got)
	}
}

func TestEffective_MultiLevelChain_DisabledIntermediateUsesLocalValue(t *testing.T) {
	grandparent := testContainer(t, "grandparent")
	parent := testContainer(t, "parent")
	child := testConnection(t, "child")
	grandparent.Base().Raw.Username = "grandparent-user"
	parent.Base().Raw.Username = "parent-user"
	child.Raw.Username = "child-user"
	child.Inheritance.Username = true
	if err := grandparent.AddChild(parent); err != nil {
		t.Fatal(err)
	}
	if err := parent.AddChild(child); err != nil {
		t.Fatal(err)
	}

	if got := child.Effective().Username; got != "parent-user" {
		t.Errorf("Username = %q, want parent-user", got)
	}
}

func TestEffective_DirectChildOfRoot_DoesNotInherit(t *testing.T) {
	root := testRoot(t, "root")
	child := testConnection(t, "child")
	root.Base().Raw.Username = "root-user"
	child.Raw.Username = "local-user"
	child.Inheritance.Username = true
	if err := root.AddChild(child); err != nil {
		t.Fatal(err)
	}

	if child.InheritanceActive() {
		t.Fatal("inheritance must be inactive for a direct child of root")
	}
	if got := child.Effective().Username; got != "local-user" {
		t.Errorf("Username = %q, want local-user", got)
	}
}

func TestInheritanceActive_DetachedConnection_IsActiveButEffectiveIsLocal(t *testing.T) {
	connection := testConnection(t, "detached")
	connection.Raw.Username = "local-user"
	connection.Inheritance.Username = true
	if !connection.InheritanceActive() {
		t.Fatal("detached normal connection must report active inheritance")
	}
	if got := connection.Effective().Username; got != "local-user" {
		t.Errorf("Username = %q, want local-user", got)
	}
}

func TestInheritanceFlags_SetAllAndClone(t *testing.T) {
	var flags InheritanceFlags
	flags.SetAll(true)
	if !flags.EverythingInherited() {
		t.Fatal("SetAll(true) did not enable every flag")
	}
	flagsValue := reflect.ValueOf(flags)
	for i := 0; i < flagsValue.NumField(); i++ {
		if !flagsValue.Field(i).Bool() {
			t.Errorf("SetAll(true) left %s disabled", flagsValue.Type().Field(i).Name)
		}
	}
	clone := flags.Clone()
	clone.Username = false
	if !flags.Username {
		t.Error("mutating clone changed source flags")
	}
	flags.SetAll(false)
	if flags != (InheritanceFlags{}) {
		t.Error("SetAll(false) did not clear every flag")
	}
}

func TestEffective_EveryFlag_MapsToMatchingRawField(t *testing.T) {
	flagsType := reflect.TypeFor[InheritanceFlags]()
	valuesType := reflect.TypeFor[ConnectionValues]()
	for i := 0; i < flagsType.NumField(); i++ {
		name := flagsType.Field(i).Name
		if _, ok := valuesType.FieldByName(name); !ok {
			t.Errorf("inheritance flag %s has no matching ConnectionValues field", name)
			continue
		}

		parent := testContainer(t, "parent-"+name)
		child := testConnection(t, "child-"+name)
		parentField := reflect.ValueOf(&parent.Base().Raw).Elem().FieldByName(name)
		childField := reflect.ValueOf(&child.Raw).Elem().FieldByName(name)
		setDistinctInheritanceValues(parentField, childField)
		reflect.ValueOf(&child.Inheritance).Elem().FieldByName(name).SetBool(true)
		if err := parent.AddChild(child); err != nil {
			t.Fatal(err)
		}

		effectiveField := reflect.ValueOf(child.Effective()).FieldByName(name)
		if !reflect.DeepEqual(effectiveField.Interface(), parentField.Interface()) {
			t.Errorf("flag %s did not inherit: got %v, want %v", name, effectiveField.Interface(), parentField.Interface())
		}
	}
}

func TestContainerInfo_ApplyInheritanceToChildren_ClonesTemplateRecursively(t *testing.T) {
	container := testContainer(t, "container")
	nested := testContainer(t, "nested")
	leaf := testConnection(t, "leaf")
	container.Base().Inheritance.Username = true
	container.Base().Inheritance.Port = true
	_ = container.AddChild(nested)
	_ = nested.AddChild(leaf)

	container.ApplyInheritanceToChildren()
	for _, node := range []Node{nested, leaf} {
		if !node.Base().Inheritance.Username || !node.Base().Inheritance.Port {
			t.Errorf("node %s did not receive template", node.Base().ID())
		}
	}
	leaf.Inheritance.Username = false
	if !nested.Base().Inheritance.Username || !container.Base().Inheritance.Username {
		t.Error("descendants share mutable inheritance state")
	}
}

func TestContainerInfo_AddChild_Root_ReturnsError(t *testing.T) {
	container := testContainer(t, "container")
	root := testRoot(t, "root")
	if err := container.AddChild(root); !errors.Is(err, ErrRootChild) {
		t.Fatalf("error = %v, want ErrRootChild", err)
	}
}

func testRoot(t *testing.T, id string) *ContainerInfo {
	t.Helper()
	root, err := NewRootInfoWithID(id)
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func setDistinctInheritanceValues(parent, child reflect.Value) {
	switch parent.Kind() {
	case reflect.Bool:
		parent.SetBool(true)
		child.SetBool(false)
	case reflect.Int:
		parent.SetInt(42)
		child.SetInt(7)
	case reflect.String:
		parent.SetString("inherited")
		child.SetString("local")
	default:
		panic("unsupported ConnectionValues field kind: " + parent.Kind().String())
	}
}
