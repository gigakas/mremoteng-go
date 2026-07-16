package connection

// InheritanceFlags mirrors ConnectionInfoInheritance. A true field means the
// corresponding value resolves from the parent when inheritance is active.
type InheritanceFlags struct {
	Description          bool
	Icon                 bool
	Panel                bool
	Color                bool
	TabColor             bool
	ConnectionFrameColor bool

	ExternalCredentialProvider bool
	UserViaAPI                 bool
	Username                   bool
	VMID                       bool
	Password                   bool
	Domain                     bool
	Port                       bool
	SSHTunnelConnectionName    bool
	OpeningCommand             bool

	Protocol                 bool
	RDPVersion               bool
	ExtApp                   bool
	PuttySession             bool
	SSHOptions               bool
	RDPAuthenticationLevel   bool
	RDPMinutesToIdleTimeout  bool
	RDPAlertIdleTimeout      bool
	LoadBalanceInfo          bool
	RenderingEngine          bool
	UseConsoleSession        bool
	UseCredSsp               bool
	UseRestrictedAdmin       bool
	UseRCG                   bool
	UseRedirectionServerName bool
	UseVMID                  bool
	UseEnhancedMode          bool

	RDGatewayUsageMethod                bool
	RDGatewayHostname                   bool
	RDGatewayUseConnectionCredentials   bool
	RDGatewayUsername                   bool
	RDGatewayPassword                   bool
	RDGatewayAccessToken                bool
	RDGatewayDomain                     bool
	RDGatewayExternalCredentialProvider bool
	RDGatewayUserViaAPI                 bool

	Resolution               bool
	AutomaticResize          bool
	Colors                   bool
	CacheBitmaps             bool
	DisplayWallpaper         bool
	DisplayThemes            bool
	EnableFontSmoothing      bool
	EnableDesktopComposition bool
	DisableFullWindowDrag    bool
	DisableMenuAnimations    bool
	DisableCursorShadow      bool
	DisableCursorBlinking    bool

	RedirectKeys             bool
	RedirectDiskDrives       bool
	RedirectDiskDrivesCustom bool
	RedirectPrinters         bool
	RedirectClipboard        bool
	RedirectPorts            bool
	RedirectSmartCards       bool
	RedirectSound            bool
	SoundQuality             bool
	RedirectAudioCapture     bool

	PreExtApp       bool
	PostExtApp      bool
	MacAddress      bool
	UserField       bool
	EnvironmentTags bool
	Favorite        bool

	VNCCompression   bool
	VNCEncoding      bool
	VNCAuthMode      bool
	VNCProxyType     bool
	VNCProxyIP       bool
	VNCProxyPort     bool
	VNCProxyUsername bool
	VNCProxyPassword bool
	VNCColors        bool
	VNCSmartSizeMode bool
	VNCViewOnly      bool
}

// SetAll enables or disables every inheritance flag.
func (f *InheritanceFlags) SetAll(value bool) {
	if f != nil {
		*f = allInheritanceFlags(value)
	}
}

// EverythingInherited reports whether every field is enabled.
func (f InheritanceFlags) EverythingInherited() bool {
	return f == allInheritanceFlags(true)
}

// Clone returns an independent value copy suitable for another node.
func (f InheritanceFlags) Clone() InheritanceFlags { return f }

// InheritanceActive matches the C# rule: root nodes and direct children of a
// root retain local values even when individual flags are true.
func (c *ConnectionInfo) InheritanceActive() bool {
	if c == nil || c.containerOwner != nil && c.containerOwner.IsRoot() {
		return false
	}
	return c.parent == nil || !c.parent.IsRoot()
}

// Effective returns the values visible to protocol/UI consumers after
// recursively resolving enabled fields through Parent. Raw is never mutated.
func (c *ConnectionInfo) Effective() ConnectionValues {
	if c == nil {
		return ConnectionValues{}
	}
	values := c.Raw
	if !c.InheritanceActive() || c.parent == nil {
		return values
	}
	parentValues := c.parent.Base().Effective()
	c.Inheritance.apply(&values, parentValues)
	return values
}

// ApplyInheritanceToChildren clones this container's flag template to every
// descendant, matching ContainerInfo.ApplyInheritancePropertiesToChildren.
func (c *ContainerInfo) ApplyInheritanceToChildren() {
	if c == nil {
		return
	}
	template := c.Base().Inheritance
	for _, child := range c.Descendants() {
		child.Base().Inheritance = template.Clone()
	}
}

func (f InheritanceFlags) apply(dst *ConnectionValues, src ConnectionValues) {
	if f.Description {
		dst.Description = src.Description
	}
	if f.Icon {
		dst.Icon = src.Icon
	}
	if f.Panel {
		dst.Panel = src.Panel
	}
	if f.Color {
		dst.Color = src.Color
	}
	if f.TabColor {
		dst.TabColor = src.TabColor
	}
	if f.ConnectionFrameColor {
		dst.ConnectionFrameColor = src.ConnectionFrameColor
	}
	if f.ExternalCredentialProvider {
		dst.ExternalCredentialProvider = src.ExternalCredentialProvider
	}
	if f.UserViaAPI {
		dst.UserViaAPI = src.UserViaAPI
	}
	if f.Username {
		dst.Username = src.Username
	}
	if f.VMID {
		dst.VMID = src.VMID
	}
	if f.Password {
		dst.Password = src.Password
	}
	if f.Domain {
		dst.Domain = src.Domain
	}
	if f.Port {
		dst.Port = src.Port
	}
	if f.SSHTunnelConnectionName {
		dst.SSHTunnelConnectionName = src.SSHTunnelConnectionName
	}
	if f.OpeningCommand {
		dst.OpeningCommand = src.OpeningCommand
	}
	if f.Protocol {
		dst.Protocol = src.Protocol
	}
	if f.RDPVersion {
		dst.RDPVersion = src.RDPVersion
	}
	if f.ExtApp {
		dst.ExtApp = src.ExtApp
	}
	if f.PuttySession {
		dst.PuttySession = src.PuttySession
	}
	if f.SSHOptions {
		dst.SSHOptions = src.SSHOptions
	}
	if f.RDPAuthenticationLevel {
		dst.RDPAuthenticationLevel = src.RDPAuthenticationLevel
	}
	if f.RDPMinutesToIdleTimeout {
		dst.RDPMinutesToIdleTimeout = src.RDPMinutesToIdleTimeout
	}
	if f.RDPAlertIdleTimeout {
		dst.RDPAlertIdleTimeout = src.RDPAlertIdleTimeout
	}
	if f.LoadBalanceInfo {
		dst.LoadBalanceInfo = src.LoadBalanceInfo
	}
	if f.RenderingEngine {
		dst.RenderingEngine = src.RenderingEngine
	}
	if f.UseConsoleSession {
		dst.UseConsoleSession = src.UseConsoleSession
	}
	if f.UseCredSsp {
		dst.UseCredSsp = src.UseCredSsp
	}
	if f.UseRestrictedAdmin {
		dst.UseRestrictedAdmin = src.UseRestrictedAdmin
	}
	if f.UseRCG {
		dst.UseRCG = src.UseRCG
	}
	if f.UseRedirectionServerName {
		dst.UseRedirectionServerName = src.UseRedirectionServerName
	}
	if f.UseVMID {
		dst.UseVMID = src.UseVMID
	}
	if f.UseEnhancedMode {
		dst.UseEnhancedMode = src.UseEnhancedMode
	}
	if f.RDGatewayUsageMethod {
		dst.RDGatewayUsageMethod = src.RDGatewayUsageMethod
	}
	if f.RDGatewayHostname {
		dst.RDGatewayHostname = src.RDGatewayHostname
	}
	if f.RDGatewayUseConnectionCredentials {
		dst.RDGatewayUseConnectionCredentials = src.RDGatewayUseConnectionCredentials
	}
	if f.RDGatewayUsername {
		dst.RDGatewayUsername = src.RDGatewayUsername
	}
	if f.RDGatewayPassword {
		dst.RDGatewayPassword = src.RDGatewayPassword
	}
	if f.RDGatewayAccessToken {
		dst.RDGatewayAccessToken = src.RDGatewayAccessToken
	}
	if f.RDGatewayDomain {
		dst.RDGatewayDomain = src.RDGatewayDomain
	}
	if f.RDGatewayExternalCredentialProvider {
		dst.RDGatewayExternalCredentialProvider = src.RDGatewayExternalCredentialProvider
	}
	if f.RDGatewayUserViaAPI {
		dst.RDGatewayUserViaAPI = src.RDGatewayUserViaAPI
	}
	if f.Resolution {
		dst.Resolution = src.Resolution
	}
	if f.AutomaticResize {
		dst.AutomaticResize = src.AutomaticResize
	}
	if f.Colors {
		dst.Colors = src.Colors
	}
	if f.CacheBitmaps {
		dst.CacheBitmaps = src.CacheBitmaps
	}
	if f.DisplayWallpaper {
		dst.DisplayWallpaper = src.DisplayWallpaper
	}
	if f.DisplayThemes {
		dst.DisplayThemes = src.DisplayThemes
	}
	if f.EnableFontSmoothing {
		dst.EnableFontSmoothing = src.EnableFontSmoothing
	}
	if f.EnableDesktopComposition {
		dst.EnableDesktopComposition = src.EnableDesktopComposition
	}
	if f.DisableFullWindowDrag {
		dst.DisableFullWindowDrag = src.DisableFullWindowDrag
	}
	if f.DisableMenuAnimations {
		dst.DisableMenuAnimations = src.DisableMenuAnimations
	}
	if f.DisableCursorShadow {
		dst.DisableCursorShadow = src.DisableCursorShadow
	}
	if f.DisableCursorBlinking {
		dst.DisableCursorBlinking = src.DisableCursorBlinking
	}
	if f.RedirectKeys {
		dst.RedirectKeys = src.RedirectKeys
	}
	if f.RedirectDiskDrives {
		dst.RedirectDiskDrives = src.RedirectDiskDrives
	}
	if f.RedirectDiskDrivesCustom {
		dst.RedirectDiskDrivesCustom = src.RedirectDiskDrivesCustom
	}
	if f.RedirectPrinters {
		dst.RedirectPrinters = src.RedirectPrinters
	}
	if f.RedirectClipboard {
		dst.RedirectClipboard = src.RedirectClipboard
	}
	if f.RedirectPorts {
		dst.RedirectPorts = src.RedirectPorts
	}
	if f.RedirectSmartCards {
		dst.RedirectSmartCards = src.RedirectSmartCards
	}
	if f.RedirectSound {
		dst.RedirectSound = src.RedirectSound
	}
	if f.SoundQuality {
		dst.SoundQuality = src.SoundQuality
	}
	if f.RedirectAudioCapture {
		dst.RedirectAudioCapture = src.RedirectAudioCapture
	}
	if f.PreExtApp {
		dst.PreExtApp = src.PreExtApp
	}
	if f.PostExtApp {
		dst.PostExtApp = src.PostExtApp
	}
	if f.MacAddress {
		dst.MacAddress = src.MacAddress
	}
	if f.UserField {
		dst.UserField = src.UserField
	}
	if f.EnvironmentTags {
		dst.EnvironmentTags = src.EnvironmentTags
	}
	if f.Favorite {
		dst.Favorite = src.Favorite
	}
	if f.VNCCompression {
		dst.VNCCompression = src.VNCCompression
	}
	if f.VNCEncoding {
		dst.VNCEncoding = src.VNCEncoding
	}
	if f.VNCAuthMode {
		dst.VNCAuthMode = src.VNCAuthMode
	}
	if f.VNCProxyType {
		dst.VNCProxyType = src.VNCProxyType
	}
	if f.VNCProxyIP {
		dst.VNCProxyIP = src.VNCProxyIP
	}
	if f.VNCProxyPort {
		dst.VNCProxyPort = src.VNCProxyPort
	}
	if f.VNCProxyUsername {
		dst.VNCProxyUsername = src.VNCProxyUsername
	}
	if f.VNCProxyPassword {
		dst.VNCProxyPassword = src.VNCProxyPassword
	}
	if f.VNCColors {
		dst.VNCColors = src.VNCColors
	}
	if f.VNCSmartSizeMode {
		dst.VNCSmartSizeMode = src.VNCSmartSizeMode
	}
	if f.VNCViewOnly {
		dst.VNCViewOnly = src.VNCViewOnly
	}
}

func allInheritanceFlags(value bool) InheritanceFlags {
	return InheritanceFlags{
		Description: value, Icon: value, Panel: value, Color: value, TabColor: value, ConnectionFrameColor: value,
		ExternalCredentialProvider: value, UserViaAPI: value, Username: value, VMID: value, Password: value,
		Domain: value, Port: value, SSHTunnelConnectionName: value, OpeningCommand: value,
		Protocol: value, RDPVersion: value, ExtApp: value, PuttySession: value, SSHOptions: value,
		RDPAuthenticationLevel: value, RDPMinutesToIdleTimeout: value, RDPAlertIdleTimeout: value,
		LoadBalanceInfo: value, RenderingEngine: value, UseConsoleSession: value, UseCredSsp: value,
		UseRestrictedAdmin: value, UseRCG: value, UseRedirectionServerName: value, UseVMID: value, UseEnhancedMode: value,
		RDGatewayUsageMethod: value, RDGatewayHostname: value, RDGatewayUseConnectionCredentials: value,
		RDGatewayUsername: value, RDGatewayPassword: value, RDGatewayAccessToken: value, RDGatewayDomain: value,
		RDGatewayExternalCredentialProvider: value, RDGatewayUserViaAPI: value,
		Resolution: value, AutomaticResize: value, Colors: value, CacheBitmaps: value, DisplayWallpaper: value,
		DisplayThemes: value, EnableFontSmoothing: value, EnableDesktopComposition: value,
		DisableFullWindowDrag: value, DisableMenuAnimations: value, DisableCursorShadow: value, DisableCursorBlinking: value,
		RedirectKeys: value, RedirectDiskDrives: value, RedirectDiskDrivesCustom: value, RedirectPrinters: value,
		RedirectClipboard: value, RedirectPorts: value, RedirectSmartCards: value, RedirectSound: value,
		SoundQuality: value, RedirectAudioCapture: value,
		PreExtApp: value, PostExtApp: value, MacAddress: value, UserField: value, EnvironmentTags: value, Favorite: value,
		VNCCompression: value, VNCEncoding: value, VNCAuthMode: value, VNCProxyType: value, VNCProxyIP: value,
		VNCProxyPort: value, VNCProxyUsername: value, VNCProxyPassword: value, VNCColors: value,
		VNCSmartSizeMode: value, VNCViewOnly: value,
	}
}
