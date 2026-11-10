package xml

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/mRemoteNG/mremoteng-go/internal/connection"
	"github.com/mRemoteNG/mremoteng-go/internal/security"
)

func TestSerialize_NestedTree_RoundTripsLatestValuesAndSecrets(t *testing.T) {
	document := testSerializableDocument(t)
	password := []byte("Password")
	data, err := Serialize(document, SerializeOptions{Password: password, KDFIterations: 5000, Export: true})
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.HasPrefix(text, `<?xml version="1.0" encoding="utf-8"?>`) {
		t.Errorf("XML declaration is not C# compatible: %q", text[:min(len(text), 64)])
	}
	if !strings.Contains(text, `ConfVersion="2.8"`) || !strings.Contains(text, `KdfIterations="5000"`) || !strings.Contains(text, `Export="true"`) {
		t.Errorf("root metadata missing:\n%s", text)
	}
	if strings.Contains(text, `Password="node-secret"`) {
		t.Error("plaintext password was written")
	}

	decoded, err := Deserialize(data, Options{Password: password})
	if err != nil {
		t.Fatal(err)
	}
	folder := decoded.Root.Children()[0].(*connection.ContainerInfo)
	leaf := folder.Children()[0].Base()
	if folder.ID() != "folder" || leaf.ID() != "leaf" || leaf.Raw.Name != `Leaf & "Quoted"` {
		t.Errorf("identity did not round trip: folder=%q leaf=%q name=%q", folder.ID(), leaf.ID(), leaf.Raw.Name)
	}
	if leaf.Raw.Password != "node-secret" || leaf.Raw.VNCProxyPassword != "vnc-secret" || leaf.Raw.RDGatewayPassword != "gateway-secret" || leaf.Raw.RDGatewayAccessToken != "token-secret" {
		t.Errorf("secrets = %q/%q/%q/%q", leaf.Raw.Password, leaf.Raw.VNCProxyPassword, leaf.Raw.RDGatewayPassword, leaf.Raw.RDGatewayAccessToken)
	}
	if leaf.Raw.RedirectDiskDrives != connection.RDPDiskDrivesCustom || leaf.Raw.RedirectDiskDrivesCustom != "C:,D:" || leaf.Raw.EnvironmentTags != "prod" {
		t.Error("v28 values did not round trip")
	}
	if !leaf.Inheritance.Username || !leaf.Inheritance.EnvironmentTags || leaf.Inheritance.Port {
		t.Errorf("inheritance flags = %+v", leaf.Inheritance)
	}
}

func TestSerialize_FullFileEncryption_HidesNodesAndRoundTrips(t *testing.T) {
	document := testSerializableDocument(t)
	data, err := Serialize(document, SerializeOptions{FullFileEncryption: true})
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(data, []byte(`<Node`)) || !bytes.Contains(data, []byte(`FullFileEncryption="true"`)) {
		t.Errorf("full-file output leaked nodes or metadata is missing:\n%s", data)
	}
	decoded, err := Deserialize(data, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(decoded.Root.Descendants()) != 2 {
		t.Errorf("descendants = %d, want 2", len(decoded.Root.Descendants()))
	}
}

func TestNodeSerializer_FullFilePayload_OmitsWhitespaceNodes(t *testing.T) {
	document := testSerializableDocument(t)
	provider, err := security.NewAEADWithIterations(1000)
	if err != nil {
		t.Fatal(err)
	}
	serializer := nodeSerializer{
		provider: provider,
		password: []byte(defaultPassword),
		filter:   normalizedSaveFilter(nil),
	}
	payload, err := serializer.encodeChildren(document.Root, false)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(payload, []byte("\n")) {
		t.Errorf("full-file payload contains whitespace nodes:\n%s", payload)
	}
}

func TestSerialize_InheritedValues_UsesEffectiveSnapshotAndOmitsInheritedPassword(t *testing.T) {
	root := mustRoot(t, "root")
	parent := mustContainer(t, "parent")
	child := mustConnection(t, "child")
	parent.Base().Raw.Username = "parent-user"
	parent.Base().Raw.Password = "parent-password"
	child.Raw.Username = "local-user"
	child.Raw.Password = "local-password"
	child.Inheritance.Username = true
	child.Inheritance.Password = true
	if err := root.AddChild(parent); err != nil {
		t.Fatal(err)
	}
	if err := parent.AddChild(child); err != nil {
		t.Fatal(err)
	}
	data, err := Serialize(&Document{Root: root}, SerializeOptions{})
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.Contains(text, `Username="parent-user"`) || !strings.Contains(text, `InheritUsername="true"`) || !strings.Contains(text, `InheritPassword="true"`) {
		t.Errorf("effective inheritance output missing:\n%s", text)
	}
	decoded, err := Deserialize(data, Options{})
	if err != nil {
		t.Fatal(err)
	}
	decodedChild := decoded.Root.Children()[0].(*connection.ContainerInfo).Children()[0].Base()
	if decodedChild.Raw.Password != "" {
		t.Errorf("inherited password = %q, want empty persisted value", decodedChild.Raw.Password)
	}
}

func TestSerialize_UseEnhancedMode_WritesActualField(t *testing.T) {
	root := mustRoot(t, "root")
	child := mustConnection(t, "child")
	child.Raw.UseVMID = false
	child.Raw.UseEnhancedMode = true
	if err := root.AddChild(child); err != nil {
		t.Fatal(err)
	}
	data, err := Serialize(&Document{Root: root}, SerializeOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `UseVmId="false" UseEnhancedMode="true"`) {
		t.Errorf("enhanced mode serialized from wrong field:\n%s", data)
	}
}

func TestSerialize_SaveFilter_RedactsCredentialsAndInheritance(t *testing.T) {
	document := testSerializableDocument(t)
	leaf := document.Root.Children()[0].(*connection.ContainerInfo).Children()[0].Base()
	leaf.Raw.Username = "alice"
	leaf.Raw.Domain = "example"
	data, err := Serialize(document, SerializeOptions{Export: true, SaveFilter: &SaveFilter{}})
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, forbidden := range []string{"alice", "example", "node-secret", "vnc-secret", "gateway-secret", "token-secret", "InheritUsername"} {
		if strings.Contains(text, forbidden) {
			t.Errorf("redacted output contains %q", forbidden)
		}
	}
	decoded, err := Deserialize(data, Options{})
	if err != nil {
		t.Fatal(err)
	}
	decodedLeaf := decoded.Root.Children()[0].(*connection.ContainerInfo).Children()[0].Base()
	if decodedLeaf.Raw.Username != "" || decodedLeaf.Raw.Domain != "" || decodedLeaf.Raw.Password != "" ||
		decodedLeaf.Raw.VNCProxyPassword != "" || decodedLeaf.Raw.RDGatewayPassword != "" || decodedLeaf.Raw.RDGatewayAccessToken != "" ||
		decodedLeaf.Inheritance.Username {
		t.Error("redacted output reconstructed credentials or inheritance")
	}
}

func TestSerialize_NodeNamespaceAndConnected_AreCanonical(t *testing.T) {
	root := mustRoot(t, "root")
	child := mustConnection(t, "child")
	child.PleaseConnect = true
	if err := root.AddChild(child); err != nil {
		t.Fatal(err)
	}
	data, err := Serialize(&Document{Root: root}, SerializeOptions{})
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.Contains(text, `<Node xmlns=""`) {
		t.Errorf("node does not reset default namespace:\n%s", text)
	}
	if !strings.Contains(text, `Connected="false"`) {
		t.Errorf("PleaseConnect leaked into Connected:\n%s", text)
	}
}

func TestSerialize_InvalidInput_ReturnsTypedError(t *testing.T) {
	if _, err := Serialize(nil, SerializeOptions{}); !errors.Is(err, ErrInvalidDocument) {
		t.Errorf("nil document error = %v, want ErrInvalidDocument", err)
	}
	root := mustRoot(t, "root")
	if _, err := Serialize(&Document{Root: root}, SerializeOptions{KDFIterations: 999}); !errors.Is(err, security.ErrInvalidIterations) {
		t.Errorf("iterations error = %v, want ErrInvalidIterations", err)
	}
}

func testSerializableDocument(t *testing.T) *Document {
	t.Helper()
	root := mustRoot(t, "root")
	root.Base().Raw.Name = "Connections"
	folder := mustContainer(t, "folder")
	folder.SetExpanded(true)
	folder.Base().Raw.EnvironmentTags = "prod"
	leaf := mustConnection(t, "leaf")
	leaf.Raw.Name = `Leaf & "Quoted"`
	leaf.Raw.Protocol = connection.ProtocolSSH2
	leaf.Raw.Port = 2222
	leaf.Raw.RDPVersion = connection.RDPVersion11
	leaf.Raw.Password = "node-secret"
	leaf.Raw.VNCProxyPassword = "vnc-secret"
	leaf.Raw.RDGatewayPassword = "gateway-secret"
	leaf.Raw.RDGatewayAccessToken = "token-secret"
	leaf.Raw.RedirectDiskDrives = connection.RDPDiskDrivesCustom
	leaf.Raw.RedirectDiskDrivesCustom = "C:,D:"
	leaf.Raw.EnvironmentTags = "local"
	leaf.Raw.UseEnhancedMode = true
	leaf.Inheritance.Username = true
	leaf.Inheritance.EnvironmentTags = true
	if err := root.AddChild(folder); err != nil {
		t.Fatal(err)
	}
	if err := folder.AddChild(leaf); err != nil {
		t.Fatal(err)
	}
	return &Document{Root: root}
}

func mustRoot(t *testing.T, id string) *connection.ContainerInfo {
	t.Helper()
	root, err := connection.NewRootInfoWithID(id)
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func mustContainer(t *testing.T, id string) *connection.ContainerInfo {
	t.Helper()
	container, err := connection.NewContainerInfoWithID(id)
	if err != nil {
		t.Fatal(err)
	}
	return container
}

func mustConnection(t *testing.T, id string) *connection.ConnectionInfo {
	t.Helper()
	info, err := connection.NewConnectionInfoWithID(id)
	if err != nil {
		t.Fatal(err)
	}
	return info
}
