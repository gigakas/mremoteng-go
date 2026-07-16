package xml

import (
	stdxml "encoding/xml"
	"strconv"
	"strings"

	"github.com/mRemoteNG/mremoteng-go/internal/connection"
)

type attributes map[string]string

func makeAttributes(values []stdxml.Attr) attributes {
	out := make(attributes, len(values))
	for _, value := range values {
		// mRemoteNG performs exact lookup of unqualified attributes. Default
		// element namespaces do not apply to attributes.
		if value.Name.Space != "" {
			continue
		}
		out[value.Name.Local] = value.Value
	}
	return out
}

func (a attributes) string(name string) string { return a[name] }

func (a attributes) boolean(name string) bool {
	value, err := strconv.ParseBool(a[name])
	return err == nil && value
}

func (a attributes) integer(name string) int {
	value, err := strconv.Atoi(a[name])
	if err != nil {
		return 0
	}
	return value
}

func enumValue[T ~string](a attributes, name string, fallback T, allowed ...T) T {
	value := a[name]
	for _, candidate := range allowed {
		if strings.EqualFold(value, string(candidate)) {
			return candidate
		}
	}
	return fallback
}

func protocolValue(a attributes, name string) connection.ProtocolType {
	return enumValue(a, name, connection.ProtocolRDP,
		connection.ProtocolRDP, connection.ProtocolVNC, connection.ProtocolSSH1,
		connection.ProtocolSSH2, connection.ProtocolTelnet, connection.ProtocolRlogin,
		connection.ProtocolRAW, connection.ProtocolHTTP, connection.ProtocolHTTPS,
		connection.ProtocolPowerShell, connection.ProtocolARD, connection.ProtocolTerminal,
		connection.ProtocolWSL, connection.ProtocolAnyDesk, connection.ProtocolIntApp,
		connection.ProtocolSerial)
}

func rdpVersionValue(a attributes, name string) connection.RDPVersion {
	return enumValue(a, name, connection.RDPVersionHighest,
		connection.RDPVersion6, connection.RDPVersion7, connection.RDPVersion8,
		connection.RDPVersion9, connection.RDPVersion10, connection.RDPVersion11,
		connection.RDPVersionHighest)
}

func frameColorValue(a attributes, name string) connection.ConnectionFrameColor {
	return enumValue(a, name, connection.FrameColorNone,
		connection.FrameColorNone, connection.FrameColorRed, connection.FrameColorYellow,
		connection.FrameColorGreen, connection.FrameColorBlue, connection.FrameColorPurple)
}

func authenticationValue(a attributes, name string) connection.AuthenticationLevel {
	return enumValue(a, name, connection.AuthenticationNone,
		connection.AuthenticationNone, connection.AuthenticationRequired, connection.AuthenticationWarn)
}

func renderingValue(a attributes, name string) connection.RenderingEngine {
	return enumValue(a, name, connection.RenderingEngine(""),
		connection.RenderingEngineIE, connection.RenderingEngineEdgeChromium)
}

func gatewayUsageValue(a attributes, name string) connection.RDGatewayUsageMethod {
	return enumValue(a, name, connection.RDGatewayNever,
		connection.RDGatewayNever, connection.RDGatewayAlways, connection.RDGatewayDetect)
}

func gatewayCredentialValue(a attributes, name string) connection.RDGatewayCredentialMode {
	return enumValue(a, name, connection.RDGatewayCredentialsDifferent,
		connection.RDGatewayCredentialsDifferent, connection.RDGatewayCredentialsSame,
		connection.RDGatewayCredentialsSmartCard, connection.RDGatewayCredentialsExternal,
		connection.RDGatewayCredentialsToken)
}

func resolutionValue(a attributes, name string) connection.RDPResolution {
	return enumValue(a, name, connection.RDPResolutionSmartSize,
		connection.RDPResolutionSmartSize, connection.RDPResolutionFitToWindow,
		connection.RDPResolutionFullscreen)
}

func colorsValue(a attributes, name string) connection.RDPColors {
	return enumValue(a, name, connection.RDPColors(""),
		connection.RDPColors256, connection.RDPColors15Bit, connection.RDPColors16Bit,
		connection.RDPColors24Bit, connection.RDPColors32Bit)
}

func diskDrivesValue(a attributes, name string) connection.RDPDiskDrives {
	return enumValue(a, name, connection.RDPDiskDrivesNone,
		connection.RDPDiskDrivesNone, connection.RDPDiskDrivesLocal,
		connection.RDPDiskDrivesAll, connection.RDPDiskDrivesCustom)
}

func soundsValue(a attributes, name string) connection.RDPSounds {
	return enumValue(a, name, connection.RDPSoundsLocal,
		connection.RDPSoundsLocal, connection.RDPSoundsRemote, connection.RDPSoundsNone)
}

func soundQualityValue(a attributes, name string) connection.RDPSoundQuality {
	return enumValue(a, name, connection.RDPSoundQualityDynamic,
		connection.RDPSoundQualityDynamic, connection.RDPSoundQualityMedium,
		connection.RDPSoundQualityHigh)
}

func vncCompressionValue(a attributes, name string) connection.VNCCompression {
	return enumValue(a, name, connection.VNCCompression0,
		connection.VNCCompressionNone, connection.VNCCompression0, connection.VNCCompression1,
		connection.VNCCompression2, connection.VNCCompression3, connection.VNCCompression4,
		connection.VNCCompression5, connection.VNCCompression6, connection.VNCCompression7,
		connection.VNCCompression8, connection.VNCCompression9)
}

func vncEncodingValue(a attributes, name string) connection.VNCEncoding {
	return enumValue(a, name, connection.VNCEncodingRaw,
		connection.VNCEncodingRaw, connection.VNCEncodingRRE, connection.VNCEncodingCorre,
		connection.VNCEncodingHextile, connection.VNCEncodingZlib, connection.VNCEncodingTight,
		connection.VNCEncodingZLibHex, connection.VNCEncodingZRLE)
}

func vncAuthValue(a attributes, name string) connection.VNCAuthMode {
	return enumValue(a, name, connection.VNCAuthVNC, connection.VNCAuthVNC, connection.VNCAuthWindows)
}

func vncProxyValue(a attributes, name string) connection.VNCProxyType {
	return enumValue(a, name, connection.VNCProxyNone,
		connection.VNCProxyNone, connection.VNCProxyHTTP, connection.VNCProxySocks5, connection.VNCProxyUltra)
}

func vncColorsValue(a attributes, name string) connection.VNCColors {
	return enumValue(a, name, connection.VNCColorsNormal, connection.VNCColorsNormal, connection.VNCColors8Bit)
}

func vncSmartSizeValue(a attributes, name string) connection.VNCSmartSizeMode {
	return enumValue(a, name, connection.VNCSmartSizeNone,
		connection.VNCSmartSizeNone, connection.VNCSmartSizeFree, connection.VNCSmartSizeAspect)
}

func credentialProviderValue(a attributes, name string) connection.ExternalCredentialProvider {
	return enumValue(a, name, connection.CredentialProviderNone,
		connection.CredentialProviderNone, connection.CredentialProviderDelineaSecretServer,
		connection.CredentialProviderClickstudiosPasswordState,
		connection.CredentialProviderOnePassword, connection.CredentialProviderVaultOpenbao)
}

func addressProviderValue(a attributes, name string) connection.ExternalAddressProvider {
	return enumValue(a, name, connection.AddressProviderNone,
		connection.AddressProviderNone, connection.AddressProviderAmazonWebServices)
}

func vaultEngineValue(a attributes, name string) connection.VaultOpenbaoSecretEngine {
	return enumValue(a, name, connection.VaultEngineKV,
		connection.VaultEngineKV, connection.VaultEngineLDAPDynamic,
		connection.VaultEngineLDAPStatic, connection.VaultEngineSSHOTP)
}
