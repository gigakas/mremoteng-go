package connection

// NodeKind is the serialized kind of a connection-tree node.
type NodeKind string

const (
	NodeKindConnection NodeKind = "Connection"
	NodeKindContainer  NodeKind = "Container"
)

// ProtocolType identifies the connection protocol. Values match the names
// persisted by mRemoteNG. Serial is a Go-port extension required by Phase 2.
type ProtocolType string

const (
	ProtocolRDP        ProtocolType = "RDP"
	ProtocolVNC        ProtocolType = "VNC"
	ProtocolSSH1       ProtocolType = "SSH1"
	ProtocolSSH2       ProtocolType = "SSH2"
	ProtocolTelnet     ProtocolType = "Telnet"
	ProtocolRlogin     ProtocolType = "Rlogin"
	ProtocolRAW        ProtocolType = "RAW"
	ProtocolHTTP       ProtocolType = "HTTP"
	ProtocolHTTPS      ProtocolType = "HTTPS"
	ProtocolPowerShell ProtocolType = "PowerShell"
	ProtocolARD        ProtocolType = "ARD"
	ProtocolTerminal   ProtocolType = "Terminal"
	ProtocolWSL        ProtocolType = "WSL"
	ProtocolAnyDesk    ProtocolType = "AnyDesk"
	ProtocolIntApp     ProtocolType = "IntApp"
	ProtocolSerial     ProtocolType = "Serial"
)

type ConnectionFrameColor string

const (
	FrameColorNone   ConnectionFrameColor = "None"
	FrameColorRed    ConnectionFrameColor = "Red"
	FrameColorYellow ConnectionFrameColor = "Yellow"
	FrameColorGreen  ConnectionFrameColor = "Green"
	FrameColorBlue   ConnectionFrameColor = "Blue"
	FrameColorPurple ConnectionFrameColor = "Purple"
)

type ExternalAddressProvider string

const (
	AddressProviderNone              ExternalAddressProvider = "None"
	AddressProviderAmazonWebServices ExternalAddressProvider = "AmazonWebServices"
)

type ExternalCredentialProvider string

const (
	CredentialProviderNone                      ExternalCredentialProvider = "None"
	CredentialProviderDelineaSecretServer       ExternalCredentialProvider = "DelineaSecretServer"
	CredentialProviderClickstudiosPasswordState ExternalCredentialProvider = "ClickstudiosPasswordState"
	CredentialProviderOnePassword               ExternalCredentialProvider = "OnePassword"
	CredentialProviderVaultOpenbao              ExternalCredentialProvider = "VaultOpenbao"
)

type VaultOpenbaoSecretEngine string

const (
	VaultEngineKV          VaultOpenbaoSecretEngine = "Kv"
	VaultEngineLDAPDynamic VaultOpenbaoSecretEngine = "LdapDynamic"
	VaultEngineLDAPStatic  VaultOpenbaoSecretEngine = "LdapStatic"
	VaultEngineSSHOTP      VaultOpenbaoSecretEngine = "SSHOTP"
)

type RDPVersion string

const (
	RDPVersion6       RDPVersion = "Rdc6"
	RDPVersion7       RDPVersion = "Rdc7"
	RDPVersion8       RDPVersion = "Rdc8"
	RDPVersion9       RDPVersion = "Rdc9"
	RDPVersion10      RDPVersion = "Rdc10"
	RDPVersion11      RDPVersion = "Rdc11"
	RDPVersionHighest RDPVersion = "Highest"
)

type AuthenticationLevel string

const (
	AuthenticationNone     AuthenticationLevel = "NoAuth"
	AuthenticationRequired AuthenticationLevel = "AuthRequired"
	AuthenticationWarn     AuthenticationLevel = "WarnOnFailedAuth"
)

type RenderingEngine string

const (
	RenderingEngineIE           RenderingEngine = "IE"
	RenderingEngineEdgeChromium RenderingEngine = "EdgeChromium"
)

type RDGatewayUsageMethod string

const (
	RDGatewayNever  RDGatewayUsageMethod = "Never"
	RDGatewayAlways RDGatewayUsageMethod = "Always"
	RDGatewayDetect RDGatewayUsageMethod = "Detect"
)

type RDGatewayCredentialMode string

const (
	RDGatewayCredentialsDifferent RDGatewayCredentialMode = "No"
	RDGatewayCredentialsSame      RDGatewayCredentialMode = "Yes"
	RDGatewayCredentialsSmartCard RDGatewayCredentialMode = "SmartCard"
	RDGatewayCredentialsExternal  RDGatewayCredentialMode = "ExternalCredentialProvider"
	RDGatewayCredentialsToken     RDGatewayCredentialMode = "AccessToken"
)

type RDPResolution string

const (
	RDPResolutionSmartSize   RDPResolution = "SmartSize"
	RDPResolutionFitToWindow RDPResolution = "FitToWindow"
	RDPResolutionFullscreen  RDPResolution = "Fullscreen"
)

type RDPColors string

const (
	RDPColors256   RDPColors = "Colors256"
	RDPColors15Bit RDPColors = "Colors15Bit"
	RDPColors16Bit RDPColors = "Colors16Bit"
	RDPColors24Bit RDPColors = "Colors24Bit"
	RDPColors32Bit RDPColors = "Colors32Bit"
)

type RDPDiskDrives string

const (
	RDPDiskDrivesNone   RDPDiskDrives = "None"
	RDPDiskDrivesLocal  RDPDiskDrives = "Local"
	RDPDiskDrivesAll    RDPDiskDrives = "All"
	RDPDiskDrivesCustom RDPDiskDrives = "Custom"
)

type RDPSounds string

const (
	RDPSoundsLocal  RDPSounds = "BringToThisComputer"
	RDPSoundsRemote RDPSounds = "LeaveAtRemoteComputer"
	RDPSoundsNone   RDPSounds = "DoNotPlay"
)

type RDPSoundQuality string

const (
	RDPSoundQualityDynamic RDPSoundQuality = "Dynamic"
	RDPSoundQualityMedium  RDPSoundQuality = "Medium"
	RDPSoundQualityHigh    RDPSoundQuality = "High"
)

type VNCCompression string

const (
	VNCCompressionNone VNCCompression = "CompNone"
	VNCCompression0    VNCCompression = "Comp0"
	VNCCompression1    VNCCompression = "Comp1"
	VNCCompression2    VNCCompression = "Comp2"
	VNCCompression3    VNCCompression = "Comp3"
	VNCCompression4    VNCCompression = "Comp4"
	VNCCompression5    VNCCompression = "Comp5"
	VNCCompression6    VNCCompression = "Comp6"
	VNCCompression7    VNCCompression = "Comp7"
	VNCCompression8    VNCCompression = "Comp8"
	VNCCompression9    VNCCompression = "Comp9"
)

type VNCEncoding string

const (
	VNCEncodingRaw     VNCEncoding = "EncRaw"
	VNCEncodingRRE     VNCEncoding = "EncRRE"
	VNCEncodingCorre   VNCEncoding = "EncCorre"
	VNCEncodingHextile VNCEncoding = "EncHextile"
	VNCEncodingZlib    VNCEncoding = "EncZlib"
	VNCEncodingTight   VNCEncoding = "EncTight"
	VNCEncodingZLibHex VNCEncoding = "EncZLibHex"
	VNCEncodingZRLE    VNCEncoding = "EncZRLE"
)

type VNCAuthMode string

const (
	VNCAuthVNC     VNCAuthMode = "AuthVNC"
	VNCAuthWindows VNCAuthMode = "AuthWin"
)

type VNCProxyType string

const (
	VNCProxyNone   VNCProxyType = "ProxyNone"
	VNCProxyHTTP   VNCProxyType = "ProxyHTTP"
	VNCProxySocks5 VNCProxyType = "ProxySocks5"
	VNCProxyUltra  VNCProxyType = "ProxyUltra"
)

type VNCColors string

const (
	VNCColorsNormal VNCColors = "ColNormal"
	VNCColors8Bit   VNCColors = "Col8Bit"
)

type VNCSmartSizeMode string

const (
	VNCSmartSizeNone   VNCSmartSizeMode = "SmartSNo"
	VNCSmartSizeFree   VNCSmartSizeMode = "SmartSFree"
	VNCSmartSizeAspect VNCSmartSizeMode = "SmartSAspect"
)
