package xml

import "github.com/mRemoteNG/mremoteng-go/internal/connection"

func (d nodeDecoder) decode26(info *connection.ConnectionInfo, a attributes) error {
	password, err := d.decryptAttribute(a.string("Password"))
	if err != nil {
		return err
	}
	vncPassword, err := d.decryptAttribute(a.string("VNCProxyPassword"))
	if err != nil {
		return err
	}
	gatewayPassword, err := d.decryptAttribute(a.string("RDGatewayPassword"))
	if err != nil {
		return err
	}
	gatewayToken, err := d.decryptAttribute(a.string("RDGatewayAccessToken"))
	if err != nil {
		return err
	}

	raw := &info.Raw
	raw.Name = a.string("Name")
	raw.Description = a.string("Descr")
	raw.Hostname = a.string("Hostname")
	raw.DisplayWallpaper = a.boolean("DisplayWallpaper")
	raw.DisplayThemes = a.boolean("DisplayThemes")
	raw.CacheBitmaps = a.boolean("CacheBitmaps")
	raw.Username = a.string("Username")
	raw.Password = password
	raw.Domain = a.string("Domain")
	raw.UseConsoleSession = a.boolean("ConnectToConsole")
	raw.RedirectPrinters = a.boolean("RedirectPrinters")
	raw.RedirectPorts = a.boolean("RedirectPorts")
	raw.RedirectSmartCards = a.boolean("RedirectSmartCards")
	raw.Protocol = protocolValue(a, "Protocol")
	raw.Port = a.integer("Port")
	raw.RedirectKeys = a.boolean("RedirectKeys")
	raw.PuttySession = a.string("PuttySession")
	raw.Colors = colorsValue(a, "Colors")
	raw.Resolution = resolutionValue(a, "Resolution")
	raw.RedirectSound = soundsValue(a, "RedirectSound")
	raw.RedirectAudioCapture = a.boolean("RedirectAudioCapture")
	raw.Icon = a.string("Icon")
	raw.Panel = a.string("Panel")
	raw.TabColor = a.string("TabColor")
	raw.ConnectionFrameColor = frameColorValue(a, "ConnectionFrameColor")
	info.PleaseConnect = a.boolean("Connected")
	raw.PreExtApp = a.string("PreExtApp")
	raw.PostExtApp = a.string("PostExtApp")
	raw.VNCCompression = vncCompressionValue(a, "VNCCompression")
	raw.VNCEncoding = vncEncodingValue(a, "VNCEncoding")
	raw.VNCAuthMode = vncAuthValue(a, "VNCAuthMode")
	raw.VNCProxyType = vncProxyValue(a, "VNCProxyType")
	raw.VNCProxyIP = a.string("VNCProxyIP")
	raw.VNCProxyPort = a.integer("VNCProxyPort")
	raw.VNCProxyUsername = a.string("VNCProxyUsername")
	raw.VNCProxyPassword = vncPassword
	raw.VNCColors = vncColorsValue(a, "VNCColors")
	raw.VNCSmartSizeMode = vncSmartSizeValue(a, "VNCSmartSizeMode")
	raw.VNCViewOnly = a.boolean("VNCViewOnly")
	raw.RDPAuthenticationLevel = authenticationValue(a, "RDPAuthenticationLevel")
	raw.RenderingEngine = renderingValue(a, "RenderingEngine")
	raw.MacAddress = a.string("MacAddress")
	raw.UserField = a.string("UserField")
	raw.ExtApp = a.string("ExtApp")
	raw.RDGatewayUsageMethod = gatewayUsageValue(a, "RDGatewayUsageMethod")
	raw.RDGatewayHostname = a.string("RDGatewayHostname")
	raw.RDGatewayUseConnectionCredentials = gatewayCredentialValue(a, "RDGatewayUseConnectionCredentials")
	raw.RDGatewayUsername = a.string("RDGatewayUsername")
	raw.RDGatewayPassword = gatewayPassword
	raw.RDGatewayAccessToken = gatewayToken
	raw.RDGatewayDomain = a.string("RDGatewayDomain")
	raw.EnableFontSmoothing = a.boolean("EnableFontSmoothing")
	raw.EnableDesktopComposition = a.boolean("EnableDesktopComposition")
	raw.UseCredSsp = a.boolean("UseCredSsp")
	raw.LoadBalanceInfo = a.string("LoadBalanceInfo")
	raw.AutomaticResize = a.boolean("AutomaticResize")
	raw.SoundQuality = soundQualityValue(a, "SoundQuality")
	raw.RDPMinutesToIdleTimeout = a.integer("RDPMinutesToIdleTimeout")
	raw.RDPAlertIdleTimeout = a.boolean("RDPAlertIdleTimeout")
	if a.boolean("RedirectDiskDrives") {
		raw.RedirectDiskDrives = connection.RDPDiskDrivesLocal
	} else {
		raw.RedirectDiskDrives = connection.RDPDiskDrivesNone
	}
	decodeInheritance26(&info.Inheritance, a)
	return nil
}

func decodeInheritance26(flags *connection.InheritanceFlags, a attributes) {
	flags.CacheBitmaps = a.boolean("InheritCacheBitmaps")
	flags.Colors = a.boolean("InheritColors")
	flags.Description = a.boolean("InheritDescription")
	flags.DisplayThemes = a.boolean("InheritDisplayThemes")
	flags.DisplayWallpaper = a.boolean("InheritDisplayWallpaper")
	flags.Icon = a.boolean("InheritIcon")
	flags.Panel = a.boolean("InheritPanel")
	flags.TabColor = a.boolean("InheritTabColor")
	flags.ConnectionFrameColor = a.boolean("InheritConnectionFrameColor")
	flags.Port = a.boolean("InheritPort")
	flags.Protocol = a.boolean("InheritProtocol")
	flags.PuttySession = a.boolean("InheritPuttySession")
	flags.RedirectDiskDrives = a.boolean("InheritRedirectDiskDrives")
	flags.RedirectKeys = a.boolean("InheritRedirectKeys")
	flags.RedirectPorts = a.boolean("InheritRedirectPorts")
	flags.RedirectPrinters = a.boolean("InheritRedirectPrinters")
	flags.RedirectSmartCards = a.boolean("InheritRedirectSmartCards")
	flags.RedirectSound = a.boolean("InheritRedirectSound")
	flags.RedirectAudioCapture = a.boolean("InheritRedirectAudioCapture")
	flags.Resolution = a.boolean("InheritResolution")
	flags.UseConsoleSession = a.boolean("InheritUseConsoleSession")
	flags.Domain = a.boolean("InheritDomain")
	flags.Password = a.boolean("InheritPassword")
	flags.Username = a.boolean("InheritUsername")
	flags.PreExtApp = a.boolean("InheritPreExtApp")
	flags.PostExtApp = a.boolean("InheritPostExtApp")
	flags.VNCCompression = a.boolean("InheritVNCCompression")
	flags.VNCEncoding = a.boolean("InheritVNCEncoding")
	flags.VNCAuthMode = a.boolean("InheritVNCAuthMode")
	flags.VNCProxyType = a.boolean("InheritVNCProxyType")
	flags.VNCProxyIP = a.boolean("InheritVNCProxyIP")
	flags.VNCProxyPort = a.boolean("InheritVNCProxyPort")
	flags.VNCProxyUsername = a.boolean("InheritVNCProxyUsername")
	flags.VNCProxyPassword = a.boolean("InheritVNCProxyPassword")
	flags.VNCColors = a.boolean("InheritVNCColors")
	flags.VNCSmartSizeMode = a.boolean("InheritVNCSmartSizeMode")
	flags.VNCViewOnly = a.boolean("InheritVNCViewOnly")
	flags.RDPAuthenticationLevel = a.boolean("InheritRDPAuthenticationLevel")
	flags.RenderingEngine = a.boolean("InheritRenderingEngine")
	flags.MacAddress = a.boolean("InheritMacAddress")
	flags.UserField = a.boolean("InheritUserField")
	flags.ExtApp = a.boolean("InheritExtApp")
	flags.RDGatewayUsageMethod = a.boolean("InheritRDGatewayUsageMethod")
	flags.RDGatewayHostname = a.boolean("InheritRDGatewayHostname")
	flags.RDGatewayUseConnectionCredentials = a.boolean("InheritRDGatewayUseConnectionCredentials")
	flags.RDGatewayUsername = a.boolean("InheritRDGatewayUsername")
	flags.RDGatewayPassword = a.boolean("InheritRDGatewayPassword")
	flags.RDGatewayAccessToken = a.boolean("InheritRDGatewayAccessToken")
	flags.RDGatewayDomain = a.boolean("InheritRDGatewayDomain")
	flags.EnableFontSmoothing = a.boolean("InheritEnableFontSmoothing")
	flags.EnableDesktopComposition = a.boolean("InheritEnableDesktopComposition")
	flags.UseCredSsp = a.boolean("InheritUseCredSsp")
	flags.LoadBalanceInfo = a.boolean("InheritLoadBalanceInfo")
	flags.AutomaticResize = a.boolean("InheritAutomaticResize")
	flags.SoundQuality = a.boolean("InheritSoundQuality")
	flags.RDPMinutesToIdleTimeout = a.boolean("InheritRDPMinutesToIdleTimeout")
	flags.RDPAlertIdleTimeout = a.boolean("InheritRDPAlertIdleTimeout")
}
