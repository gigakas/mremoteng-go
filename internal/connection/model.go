// Package connection defines the protocol-neutral connection model and its
// homogeneous connection/container tree. The flat model mirrors mRemoteNG's
// AbstractConnectionRecord so serializers and protocol implementations share
// one stable source of connection data.
package connection

import (
	"crypto/rand"
	"errors"
	"fmt"
	"strings"
)

var ErrInvalidID = errors.New("connection: ID must not be empty")

// ConnectionInfo is the flat superset of fields used by every supported
// protocol. Runtime tree links are private so parent/child invariants can only
// be changed through ContainerInfo operations.
type ConnectionInfo struct {
	id             string
	parent         *ContainerInfo
	containerOwner *ContainerInfo

	// Raw contains the local values persisted for this node. Stage 1.2 adds
	// Effective to resolve inherited values without making local/effective
	// reads ambiguous to serializers and protocol consumers.
	Raw ConnectionValues

	// Runtime-only state, excluded from connection attributes.
	IsDefault      bool
	IsQuickConnect bool
	PleaseConnect  bool
}

// ConnectionValues is the flat superset corresponding to the original
// AbstractConnectionRecord. It deliberately contains values only: identity,
// tree links and runtime state belong to ConnectionInfo.
type ConnectionValues struct {
	// Display.
	Name                 string
	Description          string
	Icon                 string
	Panel                string
	Color                string
	TabColor             string
	ConnectionFrameColor ConnectionFrameColor

	// Connection and credentials.
	Hostname                   string
	Port                       int
	ExternalCredentialProvider ExternalCredentialProvider
	UserViaAPI                 string
	Username                   string
	Password                   string
	VaultOpenbaoMount          string
	VaultOpenbaoRole           string
	VaultOpenbaoSecretEngine   VaultOpenbaoSecretEngine
	Domain                     string
	ExternalAddressProvider    ExternalAddressProvider
	EC2InstanceID              string
	EC2Region                  string
	VMID                       string
	SSHTunnelConnectionName    string
	OpeningCommand             string

	// Protocol.
	Protocol                 ProtocolType
	RDPVersion               RDPVersion
	ExtApp                   string
	PuttySession             string
	SSHOptions               string
	UseConsoleSession        bool
	RDPAuthenticationLevel   AuthenticationLevel
	RDPMinutesToIdleTimeout  int
	RDPAlertIdleTimeout      bool
	LoadBalanceInfo          string
	RenderingEngine          RenderingEngine
	UseCredSsp               bool
	UseRestrictedAdmin       bool
	UseRCG                   bool
	UseRedirectionServerName bool
	UseVMID                  bool
	UseEnhancedMode          bool

	// Remote Desktop Gateway.
	RDGatewayUsageMethod                RDGatewayUsageMethod
	RDGatewayHostname                   string
	RDGatewayUseConnectionCredentials   RDGatewayCredentialMode
	RDGatewayUsername                   string
	RDGatewayPassword                   string
	RDGatewayAccessToken                string
	RDGatewayDomain                     string
	RDGatewayExternalCredentialProvider ExternalCredentialProvider
	RDGatewayUserViaAPI                 string

	// Appearance.
	Resolution               RDPResolution
	AutomaticResize          bool
	Colors                   RDPColors
	CacheBitmaps             bool
	DisplayWallpaper         bool
	DisplayThemes            bool
	EnableFontSmoothing      bool
	EnableDesktopComposition bool
	DisableFullWindowDrag    bool
	DisableMenuAnimations    bool
	DisableCursorShadow      bool
	DisableCursorBlinking    bool

	// Redirection.
	RedirectKeys             bool
	RedirectDiskDrives       RDPDiskDrives
	RedirectDiskDrivesCustom string
	RedirectPrinters         bool
	RedirectClipboard        bool
	RedirectPorts            bool
	RedirectSmartCards       bool
	RedirectSound            RDPSounds
	SoundQuality             RDPSoundQuality
	RedirectAudioCapture     bool

	// Miscellaneous.
	PreExtApp              string
	PostExtApp             string
	MacAddress             string
	UserField              string
	EnvironmentTags        string
	Favorite               bool
	RDPStartProgram        string
	RDPStartProgramWorkDir string

	// VNC.
	VNCCompression   VNCCompression
	VNCEncoding      VNCEncoding
	VNCAuthMode      VNCAuthMode
	VNCProxyType     VNCProxyType
	VNCProxyIP       string
	VNCProxyPort     int
	VNCProxyUsername string
	VNCProxyPassword string
	VNCColors        VNCColors
	VNCSmartSizeMode VNCSmartSizeMode
	VNCViewOnly      bool
}

// NewConnectionInfo creates a connection with a random RFC 4122 version 4 ID
// and the same shipped defaults as a new mRemoteNG connection.
func NewConnectionInfo() (*ConnectionInfo, error) {
	id, err := newID()
	if err != nil {
		return nil, err
	}
	return NewConnectionInfoWithID(id)
}

// NewConnectionInfoWithID creates a connection preserving an existing ID,
// as required when deserializing mRemoteNG files.
func NewConnectionInfoWithID(id string) (*ConnectionInfo, error) {
	if strings.TrimSpace(id) == "" {
		return nil, ErrInvalidID
	}
	return &ConnectionInfo{
		id: id,
		Raw: ConnectionValues{
			Name:                                "New Connection",
			Icon:                                "mRemoteNG",
			Panel:                               "General",
			ConnectionFrameColor:                FrameColorNone,
			ExternalCredentialProvider:          CredentialProviderNone,
			VaultOpenbaoSecretEngine:            VaultEngineKV,
			ExternalAddressProvider:             AddressProviderNone,
			EC2Region:                           "eu-central-1",
			Protocol:                            ProtocolRDP,
			RDPVersion:                          RDPVersion10,
			Port:                                DefaultPort(ProtocolRDP),
			PuttySession:                        "Default Settings",
			RDPAuthenticationLevel:              AuthenticationNone,
			RenderingEngine:                     RenderingEngineEdgeChromium,
			UseCredSsp:                          true,
			RDGatewayUsageMethod:                RDGatewayNever,
			RDGatewayUseConnectionCredentials:   RDGatewayCredentialsSame,
			RDGatewayExternalCredentialProvider: CredentialProviderNone,
			Resolution:                          RDPResolutionSmartSize,
			AutomaticResize:                     true,
			Colors:                              RDPColors16Bit,
			RedirectDiskDrives:                  RDPDiskDrivesNone,
			RedirectSound:                       RDPSoundsNone,
			SoundQuality:                        RDPSoundQualityDynamic,
			VNCCompression:                      VNCCompressionNone,
			VNCEncoding:                         VNCEncodingHextile,
			VNCAuthMode:                         VNCAuthVNC,
			VNCProxyType:                        VNCProxyNone,
			VNCColors:                           VNCColorsNormal,
			VNCSmartSizeMode:                    VNCSmartSizeAspect,
		},
	}, nil
}

// ID returns the stable identifier persisted as the mRemoteNG Id attribute.
func (c *ConnectionInfo) ID() string {
	if c == nil {
		return ""
	}
	return c.id
}

// Parent returns the node's current parent, or nil when detached.
func (c *ConnectionInfo) Parent() *ContainerInfo {
	if c == nil {
		return nil
	}
	return c.parent
}

// DefaultPort returns mRemoteNG's default for protocol.
func DefaultPort(protocol ProtocolType) int {
	switch protocol {
	case ProtocolRDP:
		return 3389
	case ProtocolVNC, ProtocolARD:
		return 5900
	case ProtocolSSH1, ProtocolSSH2, ProtocolTerminal:
		return 22
	case ProtocolTelnet, ProtocolRAW:
		return 23
	case ProtocolRlogin:
		return 513
	case ProtocolHTTP:
		return 80
	case ProtocolHTTPS:
		return 443
	case ProtocolPowerShell:
		return 5985
	case ProtocolSerial:
		return 9600
	default:
		return 0
	}
}

func newID() (string, error) {
	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		return "", fmt.Errorf("connection: generate ID: %w", err)
	}
	id[6] = id[6]&0x0f | 0x40
	id[8] = id[8]&0x3f | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		id[0:4], id[4:6], id[6:8], id[8:10], id[10:16]), nil
}
