package csv

import (
	"strings"
	"testing"

	"github.com/mRemoteNG/mremoteng-go/internal/connection"
)

const csharpHeader = "Name;Id;Parent;NodeType;Description;Icon;Panel;TabColor;ConnectionFrameColor;Username;Password;Domain;Hostname;Port;VmId;Protocol;SSHTunnelConnectionName;OpeningCommand;SSHOptions;PuttySession;ConnectToConsole;UseCredSsp;UseRestrictedAdmin;UseRCG;UseRedirectionServerName;UseVmId;UseEnhancedMode;RenderingEngine;RDPAuthenticationLevel;LoadBalanceInfo;Colors;Resolution;AutomaticResize;DisplayWallpaper;DisplayThemes;EnableFontSmoothing;EnableDesktopComposition;DisableFullWindowDrag;DisableMenuAnimations;DisableCursorShadow;DisableCursorBlinking;CacheBitmaps;RedirectDiskDrives;RedirectDiskDrivesCustomRedirectPorts;RedirectPrinters;RedirectClipboard;RedirectSmartCards;RedirectSound;RedirectKeys;PreExtApp;PostExtApp;MacAddress;UserField;EnvironmentTags;ExtApp;Favorite;VNCCompression;VNCEncoding;VNCAuthMode;VNCProxyType;VNCProxyIP;VNCProxyPort;VNCProxyUsername;VNCProxyPassword;VNCColors;VNCSmartSizeMode;VNCViewOnly;RDGatewayUsageMethod;RDGatewayHostname;RDGatewayUseConnectionCredentials;RDGatewayUsername;RDGatewayPassword;RDGatewayDomain;RDGatewayExternalCredentialProvider;RDGatewayUserViaAPI;RedirectAudioCapture;RdpVersion;RDPStartProgram;RDPStartProgramWorkDir;UserViaAPI;EC2InstanceId;EC2Region;ExternalCredentialProvider;ExternalAddressProvider;InheritCacheBitmaps;InheritColors;InheritDescription;InheritDisplayThemes;InheritDisplayWallpaper;InheritEnableFontSmoothing;InheritEnableDesktopComposition;InheritDisableFullWindowDrag;InheritDisableMenuAnimations;InheritDisableCursorShadow;InheritDisableCursorBlinking;InheritDomain;InheritIcon;InheritPanel;InheritTabColor;InheritConnectionFrameColor;InheritPassword;InheritPort;InheritProtocol;InheritSSHTunnelConnectionName;InheritOpeningCommand;InheritSSHOptions;InheritPuttySession;InheritRedirectDiskDrives;InheritRedirectDiskDrivesCustom;InheritRedirectKeys;InheritRedirectPorts;InheritRedirectPrinters;InheritRedirectClipboard;InheritRedirectSmartCards;InheritRedirectSound;InheritResolution;InheritAutomaticResize;InheritUseConsoleSession;InheritUseCredSsp;InheritUseRestrictedAdmin;InheritUseRCG;InheritUseRedirectionServerName;InheritUseVmId;InheritUseEnhancedMode;InheritVmId;InheritRenderingEngine;InheritUsername;InheritRDPAuthenticationLevel;InheritLoadBalanceInfo;InheritPreExtApp;InheritPostExtApp;InheritMacAddress;InheritUserField;InheritEnvironmentTags;InheritFavorite;InheritExtApp;InheritVNCCompression;InheritVNCEncoding;InheritVNCAuthMode;InheritVNCProxyType;InheritVNCProxyIP;InheritVNCProxyPort;InheritVNCProxyUsername;InheritVNCProxyPassword;InheritVNCColors;InheritVNCSmartSizeMode;InheritVNCViewOnly;InheritRDGatewayUsageMethod;InheritRDGatewayHostname;InheritRDGatewayUseConnectionCredentials;InheritRDGatewayUsername;InheritRDGatewayPassword;InheritRDGatewayDomain;InheritRDGatewayExternalCredentialProvider;InheritRDGatewayUserViaAPI;InheritRDPAlertIdleTimeout;InheritRDPMinutesToIdleTimeout;InheritSoundQuality;InheritUserViaAPI;InheritRedirectAudioCapture;InheritRdpVersion;InheritExternalCredentialProvider"

func TestSerialize_CSharpSchema_EmitsExactPublishedHeader(t *testing.T) {
	root := mustRoot(t, "root")
	if err := root.AddChild(mustConnection(t, "node", "Node")); err != nil {
		t.Fatal(err)
	}
	data, err := Serialize(root, SerializeOptions{})
	if err != nil {
		t.Fatal(err)
	}
	header := strings.Split(string(data), "\n")[0]
	if header != csharpHeader {
		t.Fatalf("header differs from C# fixture\ngot:  %s\nwant: %s", header, csharpHeader)
	}
	if strings.HasSuffix(header, ";") {
		t.Error("C# inheritance header must not have a final semicolon")
	}
	for _, goOnly := range []string{";Color;", "RDGatewayAccessToken", "VaultOpenbao", ";SoundQuality;"} {
		if strings.Contains(header, goOnly) {
			t.Errorf("header contains non-C# column %q", goOnly)
		}
	}
}

func TestSerializeDeserialize_CSharpDefectiveSchema_RoundTripsNestedTreeAndShiftedFields(t *testing.T) {
	root := mustRoot(t, "root-id")
	folder := mustContainer(t, "folder-id", "Folder")
	folder.Base().Raw.UserViaAPI = "api-user"
	child := mustConnection(t, "child-id", "Child")
	child.Raw.RedirectDiskDrivesCustom = "C,D"
	child.Raw.RedirectPorts = true
	child.Raw.RedirectPrinters = true
	child.Raw.RDPVersion = connection.RDPVersion11
	child.Inheritance.UserViaAPI = true
	child.Inheritance.RedirectAudioCapture = true
	if err := folder.AddChild(child); err != nil {
		t.Fatal(err)
	}
	if err := root.AddChild(folder); err != nil {
		t.Fatal(err)
	}
	data, err := Serialize(root, SerializeOptions{})
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := Deserialize(data)
	if err != nil {
		t.Fatalf("Deserialize(C# output) error = %v", err)
	}
	if decoded.ID() != "root-id" {
		t.Errorf("root ID = %q", decoded.ID())
	}
	decodedFolder := decoded.Children()[0].(*connection.ContainerInfo)
	got := decodedFolder.Children()[0].Base()
	if got.Raw.RedirectDiskDrivesCustom != "C,D" || !got.Raw.RedirectPorts || !got.Raw.RedirectPrinters {
		t.Errorf("fields around merged header = custom %q ports %v printers %v", got.Raw.RedirectDiskDrivesCustom, got.Raw.RedirectPorts, got.Raw.RedirectPrinters)
	}
	if got.Raw.RDPVersion != connection.RDPVersion11 || got.Raw.UserViaAPI != "api-user" {
		t.Errorf("tail fields = RDPVersion %q UserViaAPI %q", got.Raw.RDPVersion, got.Raw.UserViaAPI)
	}
	if !got.Inheritance.UserViaAPI || !got.Inheritance.RedirectAudioCapture {
		t.Errorf("shifted inheritance flags were not restored: %+v", got.Inheritance)
	}
}

func TestSerialize_SaveFilter_OnlyConditionsPrimaryCredentials(t *testing.T) {
	node := mustConnection(t, "node", "Node")
	node.Raw.Username = "primary-user"
	node.Raw.Password = "primary-password"
	node.Raw.Domain = "primary-domain"
	node.Raw.VNCProxyUsername = "proxy-user"
	node.Raw.VNCProxyPassword = "proxy-password"
	node.Raw.RDGatewayUsername = "gateway-user"
	node.Raw.RDGatewayPassword = "gateway-password"
	node.Raw.RDGatewayDomain = "gateway-domain"
	data, err := Serialize(node, SerializeOptions{SaveFilter: &SaveFilter{}})
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(string(data), "\n")
	for _, primary := range []string{";Username;", ";Password;", ";Domain;", "InheritCacheBitmaps"} {
		if strings.Contains(lines[0], primary) {
			t.Errorf("filtered header contains %q", primary)
		}
	}
	for _, secondary := range []string{"proxy-user", "proxy-password", "gateway-user", "gateway-password", "gateway-domain"} {
		if !strings.Contains(lines[1], secondary) {
			t.Errorf("C# row omitted secondary credential %q", secondary)
		}
	}
	if !strings.HasSuffix(lines[0], ";") {
		t.Error("base-only C# header must retain final semicolon")
	}
}

func TestSerialize_InheritedPassword_WritesEffectiveValueAndFlag(t *testing.T) {
	root := mustRoot(t, "root")
	folder := mustContainer(t, "folder", "Folder")
	folder.Base().Raw.Password = "effective-secret"
	child := mustConnection(t, "child", "Child")
	child.Raw.Password = "local-secret"
	child.Inheritance.Password = true
	if err := folder.AddChild(child); err != nil {
		t.Fatal(err)
	}
	if err := root.AddChild(folder); err != nil {
		t.Fatal(err)
	}
	data, err := Serialize(root, SerializeOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Split(string(data), "\n")[1], "effective-secret;") {
		t.Error("inherited password did not use effective value")
	}
	decoded, err := Deserialize(data)
	if err != nil {
		t.Fatal(err)
	}
	got := decoded.Children()[0].(*connection.ContainerInfo).Children()[0].Base()
	if !got.Inheritance.Password {
		t.Error("password inheritance flag was not preserved")
	}
}

func TestSerialize_NonRootContainer_IncludesContainerAfterChildren(t *testing.T) {
	folder := mustContainer(t, "folder", "Folder")
	if err := folder.AddChild(mustConnection(t, "child", "Child")); err != nil {
		t.Fatal(err)
	}
	data, err := Serialize(folder, SerializeOptions{SaveFilter: &SaveFilter{}})
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(string(data), "\n")
	if len(lines) != 3 || !strings.HasPrefix(lines[1], "Child;child;folder;Connection;") || !strings.HasPrefix(lines[2], "Folder;folder;;Container;") {
		t.Fatalf("non-root serialization order:\n%s", data)
	}
}

func TestDeserialize_HeaderDrivenCompatibility_ToleratesUnknownAndMalformedValues(t *testing.T) {
	data := []byte("Unknown;Protocol;Port;UseCredSsp;NodeType;Id;Name;Parent;\nignored;future-protocol;bad-port;bad-bool;Connection;child;Child;leaf-parent;\nignored;RDP;22;True;Connection;leaf-parent;Not a container;root;\n")
	root, err := Deserialize(data)
	if err != nil {
		t.Fatalf("Deserialize() error = %v", err)
	}
	if root.ID() != "root" || len(root.Children()) != 2 {
		t.Fatalf("fallback root = %q children = %d", root.ID(), len(root.Children()))
	}
	child := root.Children()[0].Base()
	if child.Raw.Protocol != connection.ProtocolRDP || child.Raw.Port != 3389 || !child.Raw.UseCredSsp {
		t.Errorf("malformed values replaced defaults: protocol %q port %d credssp %v", child.Raw.Protocol, child.Raw.Port, child.Raw.UseCredSsp)
	}
}

func TestDeserialize_SplitCorrectedColumns_AcceptsCompatibilityAliases(t *testing.T) {
	data := []byte("Name;Id;NodeType;RedirectDiskDrivesCustom;RedirectPorts;\nNode;id;Connection;C,D;True;\n")
	root, err := Deserialize(data)
	if err != nil {
		t.Fatal(err)
	}
	got := root.Children()[0].Base().Raw
	if got.RedirectDiskDrivesCustom != "C,D" || !got.RedirectPorts {
		t.Errorf("split aliases = %q/%v", got.RedirectDiskDrivesCustom, got.RedirectPorts)
	}
}

func TestSerialize_SchemaValidationAndSemicolonSanitization_PreventsReflectionFailure(t *testing.T) {
	if err := validateSchema(); err != nil {
		t.Fatalf("validateSchema() error = %v", err)
	}
	node := mustConnection(t, "id", "na;me")
	data, err := Serialize(node, SerializeOptions{SaveFilter: &SaveFilter{}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(strings.Split(string(data), "\n")[1], "name;id;;Connection;") {
		t.Fatalf("semicolon was not removed:\n%s", data)
	}
}

func mustRoot(t *testing.T, id string) *connection.ContainerInfo {
	t.Helper()
	root, err := connection.NewRootInfoWithID(id)
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func mustContainer(t *testing.T, id, name string) *connection.ContainerInfo {
	t.Helper()
	container, err := connection.NewContainerInfoWithID(id)
	if err != nil {
		t.Fatal(err)
	}
	container.Base().Raw.Name = name
	return container
}

func mustConnection(t *testing.T, id, name string) *connection.ConnectionInfo {
	t.Helper()
	node, err := connection.NewConnectionInfoWithID(id)
	if err != nil {
		t.Fatal(err)
	}
	node.Raw.Name = name
	return node
}
