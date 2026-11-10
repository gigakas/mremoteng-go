package xml

import (
	"bytes"
	stdxml "encoding/xml"
	"fmt"

	"github.com/mRemoteNG/mremoteng-go/internal/connection"
	"github.com/mRemoteNG/mremoteng-go/internal/security"
)

type nodeSerializer struct {
	provider security.Provider
	password []byte
	filter   SaveFilter
}

func (s nodeSerializer) encodeChildren(container *connection.ContainerInfo, indent bool) ([]byte, error) {
	var output bytes.Buffer
	encoder := stdxml.NewEncoder(&output)
	if indent {
		encoder.Indent("", "  ")
	}
	for _, child := range container.Children() {
		if err := s.encodeNode(encoder, child, true); err != nil {
			return nil, err
		}
	}
	if err := encoder.Flush(); err != nil {
		return nil, fmt.Errorf("xml: flush nodes: %w", err)
	}
	return output.Bytes(), nil
}

func (s nodeSerializer) encodeNode(encoder *stdxml.Encoder, node connection.Node, resetNamespace bool) error {
	attrs, err := s.nodeAttributes(node)
	if err != nil {
		return err
	}
	if resetNamespace {
		attrs = append([]stdxml.Attr{stringXMLAttr("xmlns", "")}, attrs...)
	}
	start := stdxml.StartElement{Name: stdxml.Name{Local: "Node"}, Attr: attrs}
	if err := encoder.EncodeToken(start); err != nil {
		return fmt.Errorf("xml: encode node %q: %w", node.Base().Raw.Name, err)
	}
	if container, ok := node.(*connection.ContainerInfo); ok {
		for _, child := range container.Children() {
			if err := s.encodeNode(encoder, child, false); err != nil {
				return err
			}
		}
	}
	if err := encoder.EncodeToken(start.End()); err != nil {
		return fmt.Errorf("xml: close node %q: %w", node.Base().Raw.Name, err)
	}
	return nil
}

func (s nodeSerializer) nodeAttributes(node connection.Node) ([]stdxml.Attr, error) {
	info := node.Base()
	values := info.Effective()
	password := ""
	var err error
	if s.filter.Password && !info.Inheritance.Password {
		password, err = s.provider.Encrypt(values.Password, s.password)
		if err != nil {
			return nil, fmt.Errorf("xml: encrypt Password for %q: %w", info.Raw.Name, err)
		}
	}
	vncPassword, gatewayPassword, gatewayToken := "", "", ""
	if s.filter.Password {
		vncPassword, err = s.provider.Encrypt(values.VNCProxyPassword, s.password)
		if err != nil {
			return nil, fmt.Errorf("xml: encrypt VNCProxyPassword for %q: %w", info.Raw.Name, err)
		}
		gatewayPassword, err = s.provider.Encrypt(values.RDGatewayPassword, s.password)
		if err != nil {
			return nil, fmt.Errorf("xml: encrypt RDGatewayPassword for %q: %w", info.Raw.Name, err)
		}
		gatewayToken, err = s.provider.Encrypt(values.RDGatewayAccessToken, s.password)
		if err != nil {
			return nil, fmt.Errorf("xml: encrypt RDGatewayAccessToken for %q: %w", info.Raw.Name, err)
		}
	}

	attrs := []stdxml.Attr{
		stringXMLAttr("Name", info.Raw.Name),
		stringXMLAttr("VmId", values.VMID),
		boolXMLAttr("UseVmId", values.UseVMID),
		boolXMLAttr("UseEnhancedMode", values.UseEnhancedMode),
		nodeTypeAttr(node),
	}
	if container, ok := node.(*connection.ContainerInfo); ok {
		attrs = append(attrs, boolXMLAttr("Expanded", container.Expanded()))
	}
	attrs = append(attrs,
		stringXMLAttr("Descr", values.Description),
		stringXMLAttr("Icon", values.Icon),
		stringXMLAttr("Panel", values.Panel),
		stringXMLAttr("TabColor", values.TabColor),
		enumXMLAttr("ConnectionFrameColor", values.ConnectionFrameColor),
		stringXMLAttr("Id", info.ID()),
		stringXMLAttr("Username", filteredString(values.Username, s.filter.Username)),
		stringXMLAttr("Domain", filteredString(values.Domain, s.filter.Domain)),
		stringXMLAttr("Password", password),
		stringXMLAttr("Hostname", values.Hostname),
		enumXMLAttr("Protocol", values.Protocol),
		stringXMLAttr("RdpVersion", lowerFirst(string(values.RDPVersion))),
		stringXMLAttr("SSHTunnelConnectionName", values.SSHTunnelConnectionName),
		stringXMLAttr("OpeningCommand", values.OpeningCommand),
		stringXMLAttr("SSHOptions", values.SSHOptions),
		stringXMLAttr("PuttySession", values.PuttySession),
		intXMLAttr("Port", values.Port),
		boolXMLAttr("ConnectToConsole", values.UseConsoleSession),
		boolXMLAttr("UseCredSsp", values.UseCredSsp),
		enumXMLAttr("RenderingEngine", values.RenderingEngine),
		enumXMLAttr("RDPAuthenticationLevel", values.RDPAuthenticationLevel),
		intXMLAttr("RDPMinutesToIdleTimeout", values.RDPMinutesToIdleTimeout),
		boolXMLAttr("RDPAlertIdleTimeout", values.RDPAlertIdleTimeout),
		stringXMLAttr("LoadBalanceInfo", values.LoadBalanceInfo),
		enumXMLAttr("Colors", values.Colors),
		enumXMLAttr("Resolution", values.Resolution),
		boolXMLAttr("AutomaticResize", values.AutomaticResize),
		boolXMLAttr("DisplayWallpaper", values.DisplayWallpaper),
		boolXMLAttr("DisplayThemes", values.DisplayThemes),
		boolXMLAttr("EnableFontSmoothing", values.EnableFontSmoothing),
		boolXMLAttr("EnableDesktopComposition", values.EnableDesktopComposition),
		boolXMLAttr("DisableFullWindowDrag", values.DisableFullWindowDrag),
		boolXMLAttr("DisableMenuAnimations", values.DisableMenuAnimations),
		boolXMLAttr("DisableCursorShadow", values.DisableCursorShadow),
		boolXMLAttr("DisableCursorBlinking", values.DisableCursorBlinking),
		boolXMLAttr("CacheBitmaps", values.CacheBitmaps),
		enumXMLAttr("RedirectDiskDrives", values.RedirectDiskDrives),
		stringXMLAttr("RedirectDiskDrivesCustom", values.RedirectDiskDrivesCustom),
		boolXMLAttr("RedirectPorts", values.RedirectPorts),
		boolXMLAttr("RedirectPrinters", values.RedirectPrinters),
		boolXMLAttr("RedirectClipboard", values.RedirectClipboard),
		boolXMLAttr("RedirectSmartCards", values.RedirectSmartCards),
		enumXMLAttr("RedirectSound", values.RedirectSound),
		enumXMLAttr("SoundQuality", values.SoundQuality),
		boolXMLAttr("RedirectAudioCapture", values.RedirectAudioCapture),
		boolXMLAttr("RedirectKeys", values.RedirectKeys),
		boolXMLAttr("Connected", false),
		stringXMLAttr("PreExtApp", values.PreExtApp),
		stringXMLAttr("PostExtApp", values.PostExtApp),
		stringXMLAttr("MacAddress", values.MacAddress),
		stringXMLAttr("UserField", values.UserField),
		stringXMLAttr("EnvironmentTags", values.EnvironmentTags),
		boolXMLAttr("Favorite", values.Favorite),
		stringXMLAttr("ExtApp", values.ExtApp),
		stringXMLAttr("StartProgram", values.RDPStartProgram),
		stringXMLAttr("StartProgramWorkDir", values.RDPStartProgramWorkDir),
		enumXMLAttr("VNCCompression", values.VNCCompression),
		enumXMLAttr("VNCEncoding", values.VNCEncoding),
		enumXMLAttr("VNCAuthMode", values.VNCAuthMode),
		enumXMLAttr("VNCProxyType", values.VNCProxyType),
		stringXMLAttr("VNCProxyIP", values.VNCProxyIP),
		intXMLAttr("VNCProxyPort", values.VNCProxyPort),
		stringXMLAttr("VNCProxyUsername", filteredString(values.VNCProxyUsername, s.filter.Username)),
		stringXMLAttr("VNCProxyPassword", vncPassword),
		enumXMLAttr("VNCColors", values.VNCColors),
		enumXMLAttr("VNCSmartSizeMode", values.VNCSmartSizeMode),
		boolXMLAttr("VNCViewOnly", values.VNCViewOnly),
		enumXMLAttr("RDGatewayUsageMethod", values.RDGatewayUsageMethod),
		stringXMLAttr("RDGatewayHostname", values.RDGatewayHostname),
		enumXMLAttr("RDGatewayUseConnectionCredentials", values.RDGatewayUseConnectionCredentials),
		enumXMLAttr("RDGatewayExternalCredentialProvider", values.RDGatewayExternalCredentialProvider),
		stringXMLAttr("RDGatewayUserViaAPI", values.RDGatewayUserViaAPI),
		stringXMLAttr("RDGatewayUsername", filteredString(values.RDGatewayUsername, s.filter.Username)),
		stringXMLAttr("RDGatewayPassword", gatewayPassword),
		stringXMLAttr("RDGatewayAccessToken", gatewayToken),
		stringXMLAttr("RDGatewayDomain", filteredString(values.RDGatewayDomain, s.filter.Domain)),
		boolXMLAttr("UseRCG", values.UseRCG),
		boolXMLAttr("UseRestrictedAdmin", values.UseRestrictedAdmin),
		boolXMLAttr("UseRedirectionServerName", values.UseRedirectionServerName),
		stringXMLAttr("UserViaAPI", values.UserViaAPI),
		stringXMLAttr("EC2InstanceId", values.EC2InstanceID),
		stringXMLAttr("EC2Region", values.EC2Region),
		enumXMLAttr("ExternalCredentialProvider", values.ExternalCredentialProvider),
		enumXMLAttr("ExternalAddressProvider", values.ExternalAddressProvider),
		stringXMLAttr("VaultOpenbaoMount", values.VaultOpenbaoMount),
		stringXMLAttr("VaultOpenbaoRole", values.VaultOpenbaoRole),
		enumXMLAttr("VaultOpenbaoSecretEngine", values.VaultOpenbaoSecretEngine),
	)
	if s.filter.Inheritance {
		attrs = appendInheritanceAttributes(attrs, info.Inheritance)
	}
	return attrs, nil
}

func lowerFirst(value string) string {
	if value == "" {
		return ""
	}
	return string(value[0]+('a'-'A')) + value[1:]
}

func filteredString(value string, include bool) string {
	if !include {
		return ""
	}
	return value
}
