package xml

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/mRemoteNG/mremoteng-go/internal/connection"
	"github.com/mRemoteNG/mremoteng-go/internal/security"
)

func TestDeserialize_V28NestedTree_DecodesValuesInheritanceAndSecrets(t *testing.T) {
	password := []byte("Password")
	provider := testProvider(t, 1000)
	secret := testEncrypt(t, provider, "node-secret", password)
	vncSecret := testEncrypt(t, provider, "vnc-secret", password)
	gatewaySecret := testEncrypt(t, provider, "gateway-secret", password)
	token := testEncrypt(t, provider, "token-secret", password)
	inner := fmt.Sprintf(`<Node Type="Container" Id="folder" Name="Folder" Expanded="true" Protocol="SSH2" Port="2222" RdpVersion="rdc11" RedirectDiskDrives="Custom" RedirectDiskDrivesCustom="C:,D:" EnvironmentTags="prod" InheritEnvironmentTags="true">
  <Node Type="Connection" Id="leaf" Name="Leaf" Username="alice" Password="%s" VNCProxyPassword="%s" RDGatewayPassword="%s" RDGatewayAccessToken="%s" ExternalCredentialProvider="OnePassword" ExternalAddressProvider="AmazonWebServices" VaultOpenbaoSecretEngine="LdapDynamic" UseVmId="true" VmId="vm-1" UseRestrictedAdmin="true" InheritUsername="true" InheritRdpVersion="true" />
</Node>`, secret, vncSecret, gatewaySecret, token)

	document, err := Deserialize(testXML(t, "2.8", password, 1000, false, inner), Options{Password: password})
	if err != nil {
		t.Fatal(err)
	}
	if document.Metadata.Name != "Connections" || document.Metadata.ConfVersion != "2.8" {
		t.Errorf("metadata = %+v", document.Metadata)
	}
	folder := document.Root.Children()[0].(*connection.ContainerInfo)
	if folder.ID() != "folder" || !folder.Expanded() {
		t.Errorf("folder = id %q expanded %v", folder.ID(), folder.Expanded())
	}
	if folder.Base().Raw.Protocol != connection.ProtocolSSH2 || folder.Base().Raw.RedirectDiskDrives != connection.RDPDiskDrivesCustom {
		t.Errorf("folder values = protocol %q drives %q", folder.Base().Raw.Protocol, folder.Base().Raw.RedirectDiskDrives)
	}
	if folder.Base().Raw.RedirectDiskDrivesCustom != "C:,D:" || folder.Base().Raw.EnvironmentTags != "prod" || !folder.Base().Inheritance.EnvironmentTags {
		t.Error("v28-only values were not decoded")
	}
	leaf := folder.Children()[0].Base()
	if leaf.Raw.Password != "node-secret" || leaf.Raw.VNCProxyPassword != "vnc-secret" || leaf.Raw.RDGatewayPassword != "gateway-secret" || leaf.Raw.RDGatewayAccessToken != "token-secret" {
		t.Errorf("decrypted secrets = %q/%q/%q/%q", leaf.Raw.Password, leaf.Raw.VNCProxyPassword, leaf.Raw.RDGatewayPassword, leaf.Raw.RDGatewayAccessToken)
	}
	if leaf.Raw.ExternalCredentialProvider != connection.CredentialProviderOnePassword || leaf.Raw.ExternalAddressProvider != connection.AddressProviderAmazonWebServices {
		t.Error("provider enums were not decoded")
	}
	if !leaf.Inheritance.Username || !leaf.Inheritance.RDPVersion || !leaf.Raw.UseVMID || leaf.Raw.VMID != "vm-1" {
		t.Error("v27 value/inheritance additions were not decoded")
	}
}

func TestDeserialize_FullFileEncryptionAndCustomIterations_DecodesPayload(t *testing.T) {
	password := []byte("custom")
	inner := `<Node Type="Connection" Id="leaf" Name="Full File" Protocol="RDP" Port="3389" />`
	data := testXML(t, "2.6", password, 5000, true, inner)
	document, err := Deserialize(data, Options{Password: password})
	if err != nil {
		t.Fatal(err)
	}
	if !document.Metadata.FullFileEncryption || document.Metadata.KDFIterations != 5000 {
		t.Errorf("metadata = %+v", document.Metadata)
	}
	children := document.Root.Children()
	if len(children) != 1 || children[0].Base().ID() != "leaf" || children[0].Base().Raw.Name != "Full File" {
		t.Errorf("children = %+v", children)
	}
}

func TestDeserialize_VersionSpecificDiskDriveHandling(t *testing.T) {
	cases := []struct {
		version string
		value   string
		want    connection.RDPDiskDrives
	}{
		{"2.6", "true", connection.RDPDiskDrivesLocal},
		{"2.7", "true", connection.RDPDiskDrivesLocal},
		{"2.8", "All", connection.RDPDiskDrivesAll},
	}
	for _, c := range cases {
		inner := fmt.Sprintf(`<Node Type="Connection" Id="leaf" RedirectDiskDrives="%s" />`, c.value)
		document, err := Deserialize(testXML(t, c.version, nil, 1000, false, inner), Options{})
		if err != nil {
			t.Fatalf("version %s: %v", c.version, err)
		}
		if got := document.Root.Children()[0].Base().Raw.RedirectDiskDrives; got != c.want {
			t.Errorf("version %s: drives = %q, want %q", c.version, got, c.want)
		}
	}
}

func TestDeserialize_MissingAndMalformedAttributes_UseCSharpDefaults(t *testing.T) {
	inner := `<Node Type="Connection" Port="bad" UseCredSsp="bad" Colors="bad" VNCCompression="bad" />`
	document, err := Deserialize(testXML(t, "2.6", nil, 1000, false, inner), Options{})
	if err != nil {
		t.Fatal(err)
	}
	leaf := document.Root.Children()[0].Base()
	if leaf.ID() == "" {
		t.Error("missing ID was not generated")
	}
	if leaf.Raw.Port != 0 || leaf.Raw.UseCredSsp || leaf.Raw.Colors != "" {
		t.Errorf("malformed values = port %d credssp %v colors %q", leaf.Raw.Port, leaf.Raw.UseCredSsp, leaf.Raw.Colors)
	}
	if leaf.Raw.Protocol != connection.ProtocolRDP || leaf.Raw.Resolution != connection.RDPResolutionSmartSize || leaf.Raw.VNCCompression != connection.VNCCompression0 {
		t.Errorf("enum defaults = protocol %q resolution %q compression %q", leaf.Raw.Protocol, leaf.Raw.Resolution, leaf.Raw.VNCCompression)
	}
}

func TestDeserialize_InvalidDocumentMetadata_ReturnsTypedError(t *testing.T) {
	valid := testXML(t, "2.8", nil, 1000, false, ``)
	cases := []struct {
		name string
		data []byte
		want error
	}{
		{"empty", nil, ErrEmptyDocument},
		{"malformed", []byte(`<Connections>`), ErrMalformedXML},
		{"missing version", []byte(strings.Replace(string(valid), ` ConfVersion="2.8"`, ``, 1)), ErrUnsupportedVersion},
		{"newer version", testXML(t, "2.9", nil, 1000, false, ``), ErrUnsupportedVersion},
		{"unsupported cipher", []byte(strings.Replace(string(valid), `EncryptionEngine="AES"`, `EncryptionEngine="Twofish"`, 1)), ErrUnsupportedCipher},
		{"bad iterations", []byte(strings.Replace(string(valid), `KdfIterations="1000"`, `KdfIterations="999"`, 1)), security.ErrInvalidIterations},
	}
	for _, c := range cases {
		if _, err := Deserialize(c.data, Options{}); !errors.Is(err, c.want) {
			t.Errorf("%s: error = %v, want %v", c.name, err, c.want)
		}
	}
}

func TestDeserialize_WrongPassword_ReturnsAuthenticationError(t *testing.T) {
	data := testXML(t, "2.8", []byte("right"), 1000, false, ``)
	if _, err := Deserialize(data, Options{Password: []byte("wrong")}); !errors.Is(err, ErrAuthentication) {
		t.Errorf("error = %v, want ErrAuthentication", err)
	}
}

func TestDeserialize_CSharpProtectedVectors_Authenticate(t *testing.T) {
	cases := []struct {
		name       string
		iterations int
		password   string
		protected  string
	}{
		{"default password", 1000, "mR3m", "8LmIO3+MWBY0zTmfjfOEdCGxhTAwnlohb1veTGNZFt6lAYvY2UOzWyjVzkx6V93smpbP0ZOuexN15u7rvwJEjawC"},
		{"5000 iterations", 5000, "mR3m", "Z1IOT8h7neJ5V7es5Iv63A2WsDG6QWl10F/Rb9ljKxvCseEITty1BfMNgiaVPfm7w61uabQKqu2waDCXUpLo1OZW"},
		{"custom password", 1000, "Password", "e/T6ajrPtNNlHreSeD4QBqToTuiqtNACKiPJv7vU+l6TWCu9JNsmL+Y8lJ4aTl5YVcstXpQjxsZ9i8+YV4Gs"},
	}
	for _, c := range cases {
		data := []byte(fmt.Sprintf(`<Connections Name="C#" EncryptionEngine="AES" BlockCipherMode="GCM" KdfIterations="%d" FullFileEncryption="false" Protected="%s" ConfVersion="2.6" />`, c.iterations, c.protected))
		if _, err := Deserialize(data, Options{Password: []byte(c.password)}); err != nil {
			t.Errorf("%s: %v", c.name, err)
		}
	}
}

func TestDeserialize_NamespacedAttributeCannotOverrideMetadata(t *testing.T) {
	data := testXML(t, "2.8", nil, 1000, false, ``)
	data = []byte(strings.Replace(string(data), `ConfVersion="2.8"`, `ConfVersion="2.8" xmlns:x="urn:test" x:ConfVersion="2.9"`, 1))
	if _, err := Deserialize(data, Options{}); err != nil {
		t.Fatal(err)
	}
}

func TestDeserialize_WhitespaceWrappedCiphertext_IsAccepted(t *testing.T) {
	data := testXML(t, "2.8", nil, 1000, false, ``)
	marker := `Protected="`
	start := strings.Index(string(data), marker) + len(marker)
	end := start + strings.Index(string(data[start:]), `"`)
	wrapped := string(data[:start]) + string(data[start:start+12]) + " \n\t" + string(data[start+12:end]) + string(data[end:])
	if _, err := Deserialize([]byte(wrapped), Options{}); err != nil {
		t.Fatal(err)
	}
}

func testXML(t *testing.T, version string, password []byte, iterations int, full bool, inner string) []byte {
	t.Helper()
	if len(password) == 0 {
		password = []byte(defaultPassword)
	}
	provider := testProvider(t, iterations)
	protected := testEncrypt(t, provider, "ThisIsProtected", password)
	body := inner
	if full {
		body = testEncrypt(t, provider, inner, password)
	}
	return []byte(fmt.Sprintf(`<Connections xmlns="http://mremoteng.org" Name=" Connections " Export="false" EncryptionEngine="AES" BlockCipherMode="GCM" KdfIterations="%d" FullFileEncryption="%t" Protected="%s" ConfVersion="%s">%s</Connections>`, iterations, full, protected, version, body))
}

func testProvider(t *testing.T, iterations int) *security.AEAD {
	t.Helper()
	provider, err := security.NewAEADWithIterations(iterations)
	if err != nil {
		t.Fatal(err)
	}
	return provider
}

func testEncrypt(t *testing.T, provider security.Provider, plaintext string, password []byte) string {
	t.Helper()
	ciphertext, err := provider.Encrypt(plaintext, password)
	if err != nil {
		t.Fatal(err)
	}
	return ciphertext
}
