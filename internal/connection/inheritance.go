package connection

import "reflect"

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

// inheritedField pairs an InheritanceFlags field with the ConnectionValues
// field of the same name that it controls.
type inheritedField struct {
	flag, value int
}

// inheritedFields is derived once from the two struct definitions. Every flag
// must name a ConnectionValues field; the init panic (and a test) catches a
// flag added without its value, so the two types cannot drift apart silently.
var inheritedFields = buildInheritedFields()

func buildInheritedFields() []inheritedField {
	flags := reflect.TypeOf(InheritanceFlags{})
	values := reflect.TypeOf(ConnectionValues{})
	fields := make([]inheritedField, 0, flags.NumField())
	for i := 0; i < flags.NumField(); i++ {
		name := flags.Field(i).Name
		if flags.Field(i).Type.Kind() != reflect.Bool {
			panic("connection: inheritance flag " + name + " is not a bool")
		}
		value, ok := values.FieldByName(name)
		if !ok {
			panic("connection: inheritance flag " + name + " has no ConnectionValues field")
		}
		fields = append(fields, inheritedField{flag: i, value: value.Index[0]})
	}
	return fields
}

// apply copies from src every field whose flag is set.
func (f InheritanceFlags) apply(dst *ConnectionValues, src ConnectionValues) {
	flags := reflect.ValueOf(f)
	to := reflect.ValueOf(dst).Elem()
	from := reflect.ValueOf(src)
	for _, field := range inheritedFields {
		if flags.Field(field.flag).Bool() {
			to.Field(field.value).Set(from.Field(field.value))
		}
	}
}

func allInheritanceFlags(value bool) InheritanceFlags {
	var flags InheritanceFlags
	target := reflect.ValueOf(&flags).Elem()
	for _, field := range inheritedFields {
		target.Field(field.flag).SetBool(value)
	}
	return flags
}
