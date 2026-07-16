package xml

import (
	stdxml "encoding/xml"

	"github.com/mRemoteNG/mremoteng-go/internal/connection"
)

func appendInheritanceAttributes(attrs []stdxml.Attr, f connection.InheritanceFlags) []stdxml.Attr {
	flags := []struct {
		name  string
		value bool
	}{
		{"InheritCacheBitmaps", f.CacheBitmaps},
		{"InheritColors", f.Colors},
		{"InheritDescription", f.Description},
		{"InheritDisplayThemes", f.DisplayThemes},
		{"InheritDisplayWallpaper", f.DisplayWallpaper},
		{"InheritEnableFontSmoothing", f.EnableFontSmoothing},
		{"InheritEnableDesktopComposition", f.EnableDesktopComposition},
		{"InheritDisableFullWindowDrag", f.DisableFullWindowDrag},
		{"InheritDisableMenuAnimations", f.DisableMenuAnimations},
		{"InheritDisableCursorShadow", f.DisableCursorShadow},
		{"InheritDisableCursorBlinking", f.DisableCursorBlinking},
		{"InheritDomain", f.Domain},
		{"InheritIcon", f.Icon},
		{"InheritPanel", f.Panel},
		{"InheritTabColor", f.TabColor},
		{"InheritConnectionFrameColor", f.ConnectionFrameColor},
		{"InheritPassword", f.Password},
		{"InheritPort", f.Port},
		{"InheritProtocol", f.Protocol},
		{"InheritRdpVersion", f.RDPVersion},
		{"InheritSSHTunnelConnectionName", f.SSHTunnelConnectionName},
		{"InheritOpeningCommand", f.OpeningCommand},
		{"InheritSSHOptions", f.SSHOptions},
		{"InheritPuttySession", f.PuttySession},
		{"InheritRedirectDiskDrives", f.RedirectDiskDrives},
		{"InheritRedirectDiskDrivesCustom", f.RedirectDiskDrivesCustom},
		{"InheritRedirectKeys", f.RedirectKeys},
		{"InheritRedirectPorts", f.RedirectPorts},
		{"InheritRedirectPrinters", f.RedirectPrinters},
		{"InheritRedirectClipboard", f.RedirectClipboard},
		{"InheritRedirectSmartCards", f.RedirectSmartCards},
		{"InheritRedirectSound", f.RedirectSound},
		{"InheritSoundQuality", f.SoundQuality},
		{"InheritRedirectAudioCapture", f.RedirectAudioCapture},
		{"InheritResolution", f.Resolution},
		{"InheritAutomaticResize", f.AutomaticResize},
		{"InheritUseConsoleSession", f.UseConsoleSession},
		{"InheritUseCredSsp", f.UseCredSsp},
		{"InheritRenderingEngine", f.RenderingEngine},
		{"InheritUsername", f.Username},
		{"InheritRDPAuthenticationLevel", f.RDPAuthenticationLevel},
		{"InheritRDPMinutesToIdleTimeout", f.RDPMinutesToIdleTimeout},
		{"InheritRDPAlertIdleTimeout", f.RDPAlertIdleTimeout},
		{"InheritLoadBalanceInfo", f.LoadBalanceInfo},
		{"InheritPreExtApp", f.PreExtApp},
		{"InheritPostExtApp", f.PostExtApp},
		{"InheritMacAddress", f.MacAddress},
		{"InheritUserField", f.UserField},
		{"InheritEnvironmentTags", f.EnvironmentTags},
		{"InheritFavorite", f.Favorite},
		{"InheritExtApp", f.ExtApp},
		{"InheritVNCCompression", f.VNCCompression},
		{"InheritVNCEncoding", f.VNCEncoding},
		{"InheritVNCAuthMode", f.VNCAuthMode},
		{"InheritVNCProxyType", f.VNCProxyType},
		{"InheritVNCProxyIP", f.VNCProxyIP},
		{"InheritVNCProxyPort", f.VNCProxyPort},
		{"InheritVNCProxyUsername", f.VNCProxyUsername},
		{"InheritVNCProxyPassword", f.VNCProxyPassword},
		{"InheritVNCColors", f.VNCColors},
		{"InheritVNCSmartSizeMode", f.VNCSmartSizeMode},
		{"InheritVNCViewOnly", f.VNCViewOnly},
		{"InheritRDGatewayUsageMethod", f.RDGatewayUsageMethod},
		{"InheritRDGatewayHostname", f.RDGatewayHostname},
		{"InheritRDGatewayUseConnectionCredentials", f.RDGatewayUseConnectionCredentials},
		{"InheritRDGatewayUsername", f.RDGatewayUsername},
		{"InheritRDGatewayPassword", f.RDGatewayPassword},
		{"InheritRDGatewayAccessToken", f.RDGatewayAccessToken},
		{"InheritRDGatewayDomain", f.RDGatewayDomain},
		{"InheritRDGatewayExternalCredentialProvider", f.RDGatewayExternalCredentialProvider},
		{"InheritRDGatewayUserViaAPI", f.RDGatewayUserViaAPI},
		{"InheritVmId", f.VMID},
		{"InheritUseVmId", f.UseVMID},
		{"InheritUseEnhancedMode", f.UseEnhancedMode},
		{"InheritExternalCredentialProvider", f.ExternalCredentialProvider},
		{"InheritUserViaAPI", f.UserViaAPI},
		{"InheritUseRCG", f.UseRCG},
		{"InheritUseRestrictedAdmin", f.UseRestrictedAdmin},
		{"InheritUseRedirectionServerName", f.UseRedirectionServerName},
	}
	for _, flag := range flags {
		if flag.value {
			attrs = append(attrs, boolXMLAttr(flag.name, true))
		}
	}
	return attrs
}
