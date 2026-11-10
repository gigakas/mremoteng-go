package integration_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/mRemoteNG/mremoteng-go/internal/connection"
	connectionxml "github.com/mRemoteNG/mremoteng-go/internal/serialize/xml"
)

const minimumCorpusSize = 20

type corpusManifest struct {
	MinimumRequired int                       `json:"minimum_required"`
	Complete        bool                      `json:"complete"`
	Profiles        map[string][]expectedNode `json:"profiles"`
	Fixtures        []corpusFixture           `json:"fixtures"`
}

type corpusFixture struct {
	File               string `json:"file"`
	Source             string `json:"source"`
	FileSHA256         string `json:"file_sha256"`
	Password           string `json:"password"`
	Profile            string `json:"profile"`
	SourceIDs          bool   `json:"source_ids"`
	IDSHA256           string `json:"id_sha256"`
	Version            string `json:"version"`
	KDFIterations      int    `json:"kdf_iterations"`
	FullFileEncryption bool   `json:"full_file_encryption"`
}

type expectedNode struct {
	Path            string `json:"path"`
	Kind            string `json:"kind"`
	Username        string `json:"username"`
	Domain          string `json:"domain"`
	Password        string `json:"password"`
	Protocol        string `json:"protocol"`
	Port            int    `json:"port"`
	InheritUsername bool   `json:"inherit_username"`
	InheritDomain   bool   `json:"inherit_domain"`
	InheritPassword bool   `json:"inherit_password"`
}

type actualNode struct {
	Path string
	Node connection.Node
}

type modelSnapshot struct {
	ID          string
	ParentID    string
	Kind        connection.NodeKind
	Expanded    bool
	Raw         connection.ConnectionValues
	Inheritance connection.InheritanceFlags
}

func TestCSharpCorpus_DeserializeAndRoundTrip_MatchesManifest(t *testing.T) {
	manifest := readCorpusManifest(t)
	validateManifest(t, manifest)
	if !manifest.Complete {
		t.Logf("corpus is incomplete: %d/%d C# fixtures", len(manifest.Fixtures), manifest.MinimumRequired)
	}

	for _, fixture := range manifest.Fixtures {
		fixture := fixture
		t.Run(fixture.File, func(t *testing.T) {
			t.Parallel()
			data := readFixture(t, fixture.File)
			validateFixtureDigest(t, data, fixture.FileSHA256)
			document := deserializeFixture(t, data, fixture.Password)
			if _, err := connectionxml.Deserialize(data, connectionxml.Options{Password: []byte(fixture.Password + "-wrong")}); !errors.Is(err, connectionxml.ErrAuthentication) {
				t.Errorf("wrong password error = %v, want ErrAuthentication", err)
			}
			validateMetadata(t, document.Metadata, fixture)
			validateExpectedNodes(t, document.Root, manifest.Profiles[fixture.Profile])
			validateIDs(t, document.Root, fixture.SourceIDs, fixture.IDSHA256)

			before := snapshotTree(document.Root)
			serialized, err := connectionxml.Serialize(document, connectionxml.SerializeOptions{
				Password:           []byte(fixture.Password),
				KDFIterations:      fixture.KDFIterations,
				FullFileEncryption: fixture.FullFileEncryption,
				Export:             document.Metadata.Export,
			})
			if err != nil {
				t.Fatalf("serialize C# fixture as v2.8: %v", err)
			}
			if !fixture.FullFileEncryption && bytes.Contains(serialized, []byte(`Password="rootpassword"`)) {
				t.Error("round-trip output contains a plaintext password")
			}
			roundTripped := deserializeFixture(t, serialized, fixture.Password)
			if roundTripped.Metadata.ConfVersion != "2.8" {
				t.Errorf("round-trip version = %q, want 2.8", roundTripped.Metadata.ConfVersion)
			}
			if after := snapshotTree(roundTripped.Root); !reflect.DeepEqual(after, before) {
				t.Errorf("round-trip model differs\nbefore: %#v\nafter:  %#v", before, after)
			}
		})
	}
}

func validateManifest(t *testing.T, manifest corpusManifest) {
	t.Helper()
	if len(manifest.Fixtures) == 0 {
		t.Fatal("corpus manifest contains no fixtures")
	}
	if manifest.MinimumRequired < minimumCorpusSize {
		t.Errorf("minimum_required = %d, must not be below %d", manifest.MinimumRequired, minimumCorpusSize)
	}
	files := make(map[string]bool)
	digests := make(map[string]bool)
	for _, fixture := range manifest.Fixtures {
		if fixture.File == "" || fixture.Source == "" || fixture.FileSHA256 == "" || fixture.Profile == "" || fixture.Password == "" {
			t.Errorf("fixture %q has incomplete provenance or options", fixture.File)
		}
		decodedDigest, err := hex.DecodeString(fixture.FileSHA256)
		if err != nil || len(decodedDigest) != sha256.Size {
			t.Errorf("fixture %q has invalid file_sha256 %q", fixture.File, fixture.FileSHA256)
		}
		if files[fixture.File] {
			t.Errorf("duplicate fixture file %q", fixture.File)
		}
		if digests[fixture.FileSHA256] {
			t.Errorf("fixture %q duplicates another fixture's content digest", fixture.File)
		}
		files[fixture.File] = true
		digests[fixture.FileSHA256] = true
	}
	if manifest.Complete {
		validateCompleteCorpus(t, manifest)
	}
}

func validateCompleteCorpus(t *testing.T, manifest corpusManifest) {
	t.Helper()
	if len(manifest.Fixtures) < minimumCorpusSize {
		t.Errorf("complete corpus has %d fixtures, want at least %d", len(manifest.Fixtures), minimumCorpusSize)
	}
	versions := make(map[string]bool)
	var fullFile, normal, defaultPassword, customPassword bool
	var nested, credentials, inheritance bool
	for _, fixture := range manifest.Fixtures {
		versions[fixture.Version] = true
		fullFile = fullFile || fixture.FullFileEncryption
		normal = normal || !fixture.FullFileEncryption
		defaultPassword = defaultPassword || fixture.Password == "mR3m"
		customPassword = customPassword || fixture.Password != "mR3m"
		for _, node := range manifest.Profiles[fixture.Profile] {
			nested = nested || strings.Count(node.Path, "/") >= 2
			credentials = credentials || node.Username != "" && node.Password != ""
			inheritance = inheritance || node.InheritUsername || node.InheritDomain || node.InheritPassword
		}
	}
	for _, version := range []string{"2.6", "2.7", "2.8"} {
		if !versions[version] {
			t.Errorf("complete corpus does not cover version %s", version)
		}
	}
	if !fullFile || !normal || !defaultPassword || !customPassword || !nested || !credentials || !inheritance {
		t.Errorf("complete corpus coverage is insufficient: full-file=%v normal=%v default-password=%v custom-password=%v nested=%v credentials=%v inheritance=%v", fullFile, normal, defaultPassword, customPassword, nested, credentials, inheritance)
	}
}

func readCorpusManifest(t *testing.T) corpusManifest {
	t.Helper()
	file, err := os.Open(filepath.Join("..", "testdata", "corpus", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			t.Errorf("close corpus manifest: %v", err)
		}
	}()
	var manifest corpusManifest
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		t.Fatalf("decode corpus manifest: %v", err)
	}
	return manifest
}

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	if filepath.Base(name) != name {
		t.Fatalf("fixture name must not contain a path: %q", name)
	}
	data, err := os.ReadFile(filepath.Join("..", "testdata", "corpus", name))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func validateFixtureDigest(t *testing.T, data []byte, want string) {
	t.Helper()
	data = bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
	digest := sha256.Sum256(data)
	got := hex.EncodeToString(digest[:])
	if got != want {
		t.Fatalf("fixture SHA-256 = %s, want %s", got, want)
	}
}

func deserializeFixture(t *testing.T, data []byte, password string) *connectionxml.Document {
	t.Helper()
	document, err := connectionxml.Deserialize(data, connectionxml.Options{Password: []byte(password)})
	if err != nil {
		t.Fatalf("deserialize fixture: %v", err)
	}
	return document
}

func validateMetadata(t *testing.T, got connectionxml.Metadata, want corpusFixture) {
	t.Helper()
	if got.ConfVersion != want.Version || got.KDFIterations != want.KDFIterations || got.FullFileEncryption != want.FullFileEncryption {
		t.Errorf("metadata = version %q, iterations %d, full-file %v; want %q, %d, %v", got.ConfVersion, got.KDFIterations, got.FullFileEncryption, want.Version, want.KDFIterations, want.FullFileEncryption)
	}
	if got.Name != "Connections" || got.EncryptionEngine != "AES" || got.BlockCipherMode != "GCM" {
		t.Errorf("unexpected C# metadata: %+v", got)
	}
}

func validateExpectedNodes(t *testing.T, root *connection.ContainerInfo, expected []expectedNode) {
	t.Helper()
	if expected == nil {
		t.Fatal("fixture references an unknown expectation profile")
	}
	actual := flattenTree(root)
	if len(actual) != len(expected) {
		t.Fatalf("node count = %d, want %d", len(actual), len(expected))
	}
	for i, want := range expected {
		got := actual[i]
		values := got.Node.Base().Raw
		inheritance := got.Node.Base().Inheritance
		if got.Path != want.Path || string(got.Node.Kind()) != want.Kind {
			t.Errorf("node %d = %s (%s), want %s (%s)", i, got.Path, got.Node.Kind(), want.Path, want.Kind)
		}
		if values.Username != want.Username || values.Domain != want.Domain || values.Password != want.Password {
			t.Errorf("%s credentials = %q/%q/%q, want %q/%q/%q", got.Path, values.Username, values.Domain, values.Password, want.Username, want.Domain, want.Password)
		}
		if string(values.Protocol) != want.Protocol || values.Port != want.Port {
			t.Errorf("%s endpoint = %s/%d, want %s/%d", got.Path, values.Protocol, values.Port, want.Protocol, want.Port)
		}
		if inheritance.Username != want.InheritUsername || inheritance.Domain != want.InheritDomain || inheritance.Password != want.InheritPassword {
			t.Errorf("%s credential inheritance = %v/%v/%v, want %v/%v/%v", got.Path, inheritance.Username, inheritance.Domain, inheritance.Password, want.InheritUsername, want.InheritDomain, want.InheritPassword)
		}
	}
}

func flattenTree(root *connection.ContainerInfo) []actualNode {
	var flattened []actualNode
	var visit func(*connection.ContainerInfo, string)
	visit = func(container *connection.ContainerInfo, prefix string) {
		for _, node := range container.Children() {
			path := node.Base().Raw.Name
			if prefix != "" {
				path = prefix + "/" + path
			}
			flattened = append(flattened, actualNode{Path: path, Node: node})
			if child, ok := node.(*connection.ContainerInfo); ok {
				visit(child, path)
			}
		}
	}
	visit(root, "")
	return flattened
}

func validateIDs(t *testing.T, root *connection.ContainerInfo, sourceIDs bool, want string) {
	t.Helper()
	var ids strings.Builder
	seen := make(map[string]bool)
	for _, item := range flattenTree(root) {
		id := item.Node.Base().ID()
		if id == "" {
			t.Errorf("%s has an empty ID", item.Path)
		}
		if seen[id] {
			t.Errorf("%s repeats ID %q", item.Path, id)
		}
		seen[id] = true
		fmt.Fprintln(&ids, id)
	}
	if !sourceIDs {
		if want != "" {
			t.Error("fixture without source IDs must not define id_sha256")
		}
		return
	}
	if want == "" {
		t.Fatal("fixture with source IDs must define id_sha256")
	}
	digest := sha256.Sum256([]byte(ids.String()))
	got := hex.EncodeToString(digest[:])
	if got != want {
		t.Errorf("ordered ID digest = %s, want %s\nIDs:\n%s", got, want, ids.String())
	}
}

func snapshotTree(root *connection.ContainerInfo) []modelSnapshot {
	items := flattenTree(root)
	snapshots := make([]modelSnapshot, 0, len(items))
	for _, item := range items {
		node := item.Node
		parentID := ""
		if parent := node.Base().Parent(); parent != nil && !parent.IsRoot() {
			parentID = parent.ID()
		}
		expanded := false
		if container, ok := node.(*connection.ContainerInfo); ok {
			expanded = container.Expanded()
		}
		snapshots = append(snapshots, modelSnapshot{
			ID:          node.Base().ID(),
			ParentID:    parentID,
			Kind:        node.Kind(),
			Expanded:    expanded,
			Raw:         node.Base().Raw,
			Inheritance: node.Base().Inheritance,
		})
	}
	return snapshots
}
