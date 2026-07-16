// Package csv reads and writes the native mRemoteNG semicolon-separated
// connection format.
package csv

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/mRemoteNG/mremoteng-go/internal/connection"
)

var (
	ErrInvalidTree   = errors.New("csv: invalid connection tree")
	ErrEmptyDocument = errors.New("csv: empty connection document")
	ErrMalformedRow  = errors.New("csv: malformed row")
	ErrInvalidSchema = errors.New("csv: invalid column schema")
)

// SaveFilter controls the primary credential columns and inheritance block.
// Secondary credential columns are always emitted, matching the C# format.
type SaveFilter struct {
	Username    bool
	Domain      bool
	Password    bool
	Inheritance bool
}

type SerializeOptions struct {
	SaveFilter *SaveFilter
}

type csvRecord struct {
	node     connection.Node
	parentID string
}

// Serialize writes target and its descendants in C# depth-first postorder.
// Only a true connection-tree root is omitted.
func Serialize(target connection.Node, options SerializeOptions) ([]byte, error) {
	if target == nil || target.Base() == nil {
		return nil, ErrInvalidTree
	}
	if err := validateSchema(); err != nil {
		return nil, err
	}
	filter := normalizeFilter(options.SaveFilter)
	active := activeColumns(filter)
	var out bytes.Buffer
	out.WriteString(strings.Join(publishedHeaders(active), ";"))
	// The C# inheritance append omits its final delimiter. The base header and
	// every data row retain one.
	if !filter.Inheritance {
		out.WriteByte(';')
	}
	if err := serializePostorder(&out, target, active, filter); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func serializePostorder(out *bytes.Buffer, node connection.Node, active []column, filter SaveFilter) error {
	if container, ok := node.(*connection.ContainerInfo); ok {
		for _, child := range container.Children() {
			if err := serializePostorder(out, child, active, filter); err != nil {
				return err
			}
		}
	}
	if container, ok := node.(*connection.ContainerInfo); ok && container.IsRoot() {
		return nil
	}
	values := node.Base().Effective()
	row := make([]string, len(active))
	for i, descriptor := range active {
		value, err := encodeColumn(node, values, descriptor)
		if err != nil {
			return fmt.Errorf("csv: encode node %q: %w", node.Base().ID(), err)
		}
		row[i] = value
	}
	out.WriteByte('\n')
	writeRow(out, row)
	return nil
}

func encodeColumn(node connection.Node, values connection.ConnectionValues, descriptor column) (string, error) {
	info := node.Base()
	if descriptor.source == columnMeta {
		switch descriptor.header {
		case "Name":
			return info.Raw.Name, nil
		case "Id":
			return info.ID(), nil
		case "Parent":
			if info.Parent() == nil {
				return "", nil
			}
			return info.Parent().ID(), nil
		case "NodeType":
			return string(node.Kind()), nil
		}
	}
	if descriptor.source == columnInheritance {
		return formatBool(reflect.ValueOf(info.Inheritance).FieldByName(descriptor.field).Bool()), nil
	}
	return formatValue(reflect.ValueOf(values).FieldByName(descriptor.field)), nil
}

func formatValue(value reflect.Value) string {
	switch value.Kind() {
	case reflect.String:
		return value.String()
	case reflect.Bool:
		return formatBool(value.Bool())
	case reflect.Int:
		return strconv.FormatInt(value.Int(), 10)
	default:
		return fmt.Sprint(value.Interface())
	}
}

func formatBool(value bool) string {
	if value {
		return "True"
	}
	return "False"
}

func writeRow(out *bytes.Buffer, fields []string) {
	for _, field := range fields {
		out.WriteString(strings.ReplaceAll(field, ";", ""))
		out.WriteByte(';')
	}
}

// Deserialize reconstructs an ordered tree. Published C# files use positional
// decoding to account for their defective header; other inputs are header-led.
// Unknown columns and malformed typed values are ignored as by the C# reader.
func Deserialize(data []byte) (*connection.ContainerInfo, error) {
	if err := validateSchema(); err != nil {
		return nil, err
	}
	lines, err := nonEmptyLines(data)
	if err != nil {
		return nil, err
	}
	headings := splitLine(lines[0].text)
	descriptors, canonical := inputColumns(headings)
	records := make([]csvRecord, 0, len(lines)-1)
	byID := make(map[string]connection.Node, len(lines)-1)
	for _, line := range lines[1:] {
		fields := splitLine(line.text)
		if canonical {
			descriptors = canonicalRowColumns(headings)
		}
		rec, err := decodeRecord(fields, descriptors)
		if err != nil {
			return nil, fmt.Errorf("csv: line %d: %w", line.number, err)
		}
		if _, duplicate := byID[rec.node.Base().ID()]; duplicate {
			// Dictionary.Add throws in C#; retain a typed structural error.
			return nil, fmt.Errorf("%w: duplicate node ID %q", ErrMalformedRow, rec.node.Base().ID())
		}
		byID[rec.node.Base().ID()] = rec.node
		records = append(records, rec)
	}

	rootID := omittedRootID(records, byID)
	var root *connection.ContainerInfo
	if rootID == "" {
		root, err = connection.NewRootInfo()
	} else {
		root, err = connection.NewRootInfoWithID(rootID)
	}
	if err != nil {
		return nil, fmt.Errorf("csv: create root: %w", err)
	}
	for _, rec := range records {
		parent := root
		if candidate, exists := byID[rec.parentID]; exists {
			if container, ok := candidate.(*connection.ContainerInfo); ok {
				parent = container
			}
		}
		if err := parent.AddChild(rec.node); err != nil && parent != root {
			if fallbackErr := root.AddChild(rec.node); fallbackErr != nil {
				return nil, fmt.Errorf("%w: attach node %q: %v", ErrInvalidTree, rec.node.Base().ID(), fallbackErr)
			}
		}
	}
	return root, nil
}

type inputLine struct {
	number int
	text   string
}

func nonEmptyLines(data []byte) ([]inputLine, error) {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 64*1024), 16*1024*1024)
	lines := make([]inputLine, 0)
	for number := 1; scanner.Scan(); number++ {
		text := strings.TrimSuffix(scanner.Text(), "\r")
		if strings.TrimSpace(text) != "" {
			lines = append(lines, inputLine{number: number, text: text})
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("csv: read document: %w", err)
	}
	if len(lines) == 0 {
		return nil, ErrEmptyDocument
	}
	return lines, nil
}

func splitLine(line string) []string {
	fields := strings.Split(line, ";")
	if len(fields) > 0 && fields[len(fields)-1] == "" {
		fields = fields[:len(fields)-1]
	}
	return fields
}

func inputColumns(headings []string) ([]column, bool) {
	for _, filter := range possibleFilters(headings) {
		active := activeColumns(filter)
		if slicesEqual(headings, publishedHeaders(active)) {
			return active, true
		}
	}
	descriptors := make([]column, len(headings))
	for i, heading := range headings {
		descriptors[i] = columnForHeading(heading)
	}
	return descriptors, false
}

func canonicalRowColumns(headings []string) []column {
	for _, filter := range possibleFilters(headings) {
		active := activeColumns(filter)
		if slicesEqual(headings, publishedHeaders(active)) {
			return active
		}
	}
	return nil
}

func possibleFilters(headings []string) []SaveFilter {
	has := func(want string) bool {
		for _, heading := range headings {
			if heading == want {
				return true
			}
		}
		return false
	}
	return []SaveFilter{{
		Username: has("Username"), Domain: has("Domain"), Password: has("Password"),
		Inheritance: has("InheritCacheBitmaps"),
	}}
}

func columnForHeading(heading string) column {
	aliases := map[string]column{
		"RedirectDiskDrivesCustom":    {header: heading, source: columnValue, field: "RedirectDiskDrivesCustom"},
		"RedirectPorts":               {header: heading, source: columnValue, field: "RedirectPorts"},
		"InheritUserViaAPI":           {header: heading, source: columnInheritance, field: "UserViaAPI"},
		"InheritRedirectAudioCapture": {header: heading, source: columnInheritance, field: "RedirectAudioCapture"},
		"InheritRdpVersion":           {header: heading, source: columnInheritance, field: "RDPVersion"},
	}
	if descriptor, ok := aliases[heading]; ok {
		return descriptor
	}
	for _, descriptor := range columns {
		if descriptor.header == heading {
			return descriptor
		}
	}
	return column{header: heading, source: 255}
}

func decodeRecord(fields []string, descriptors []column) (csvRecord, error) {
	meta := make(map[string]string)
	for i, descriptor := range descriptors {
		if i >= len(fields) {
			break
		}
		if descriptor.source == columnMeta {
			meta[descriptor.header] = fields[i]
		}
	}
	id := meta["Id"]
	var node connection.Node
	var base *connection.ConnectionInfo
	var err error
	if strings.EqualFold(meta["NodeType"], "Container") {
		var container *connection.ContainerInfo
		if id == "" {
			container, err = connection.NewContainerInfo()
		} else {
			container, err = connection.NewContainerInfoWithID(id)
		}
		node = container
		if container != nil {
			base = container.Base()
		}
	} else {
		if id == "" {
			base, err = connection.NewConnectionInfo()
		} else {
			base, err = connection.NewConnectionInfoWithID(id)
		}
		node = base
	}
	if err != nil {
		return csvRecord{}, fmt.Errorf("%w: ID %q: %v", ErrMalformedRow, id, err)
	}
	if name, exists := meta["Name"]; exists {
		base.Raw.Name = name
	}
	for i, descriptor := range descriptors {
		if i >= len(fields) || descriptor.source == columnMeta || descriptor.source > columnInheritance {
			continue
		}
		target := reflect.ValueOf(&base.Raw).Elem()
		if descriptor.source == columnInheritance {
			target = reflect.ValueOf(&base.Inheritance).Elem()
		}
		assignField(target.FieldByName(descriptor.field), fields[i])
	}
	return csvRecord{node: node, parentID: meta["Parent"]}, nil
}

func assignField(field reflect.Value, value string) {
	if !field.IsValid() {
		return
	}
	switch field.Kind() {
	case reflect.String:
		if field.Type().Name() != "string" && !validEnum(field.Type().Name(), value) {
			return
		}
		field.SetString(value)
	case reflect.Bool:
		if parsed, err := strconv.ParseBool(value); err == nil {
			field.SetBool(parsed)
		}
	case reflect.Int:
		if parsed, err := strconv.Atoi(value); err == nil {
			field.SetInt(int64(parsed))
		}
	}
}

func validEnum(typeName, value string) bool {
	allowed := map[string][]string{
		"ConnectionFrameColor":       {"None", "Red", "Yellow", "Green", "Blue", "Purple"},
		"ProtocolType":               {"RDP", "VNC", "SSH1", "SSH2", "Telnet", "Rlogin", "RAW", "HTTP", "HTTPS", "PowerShell", "ARD", "Terminal", "WSL", "AnyDesk", "IntApp", "Serial"},
		"RDPVersion":                 {"Rdc6", "Rdc7", "Rdc8", "Rdc9", "Rdc10", "Rdc11", "Highest"},
		"AuthenticationLevel":        {"NoAuth", "AuthRequired", "WarnOnFailedAuth"},
		"RenderingEngine":            {"IE", "EdgeChromium"},
		"RDGatewayUsageMethod":       {"Never", "Always", "Detect"},
		"RDGatewayCredentialMode":    {"No", "Yes", "SmartCard", "ExternalCredentialProvider", "AccessToken"},
		"RDPResolution":              {"SmartSize", "FitToWindow", "Fullscreen"},
		"RDPColors":                  {"Colors256", "Colors15Bit", "Colors16Bit", "Colors24Bit", "Colors32Bit"},
		"RDPDiskDrives":              {"None", "Local", "All", "Custom"},
		"RDPSounds":                  {"BringToThisComputer", "LeaveAtRemoteComputer", "DoNotPlay"},
		"VNCCompression":             {"CompNone", "Comp0", "Comp1", "Comp2", "Comp3", "Comp4", "Comp5", "Comp6", "Comp7", "Comp8", "Comp9"},
		"VNCEncoding":                {"EncRaw", "EncRRE", "EncCorre", "EncHextile", "EncZlib", "EncTight", "EncZLibHex", "EncZRLE"},
		"VNCAuthMode":                {"AuthVNC", "AuthWin"},
		"VNCProxyType":               {"ProxyNone", "ProxyHTTP", "ProxySocks5", "ProxyUltra"},
		"VNCColors":                  {"ColNormal", "Col8Bit"},
		"VNCSmartSizeMode":           {"SmartSNo", "SmartSFree", "SmartSAspect"},
		"ExternalCredentialProvider": {"None", "DelineaSecretServer", "ClickstudiosPasswordState", "OnePassword", "VaultOpenbao"},
		"ExternalAddressProvider":    {"None", "AmazonWebServices"},
	}
	values, known := allowed[typeName]
	if !known {
		return false
	}
	for _, candidate := range values {
		if strings.EqualFold(value, candidate) {
			return true
		}
	}
	return false
}

func omittedRootID(records []csvRecord, byID map[string]connection.Node) string {
	rootID := ""
	for _, rec := range records {
		if rec.parentID == "" {
			continue
		}
		if _, exists := byID[rec.parentID]; exists {
			continue
		}
		if rootID == "" {
			rootID = rec.parentID
		} else if rootID != rec.parentID {
			return ""
		}
	}
	return rootID
}

func activeColumns(filter SaveFilter) []column {
	active := make([]column, 0, len(columns))
	for _, descriptor := range columns {
		if descriptor.source == columnInheritance && !filter.Inheritance {
			continue
		}
		if descriptor.filter != filterNone && !filterAllows(filter, descriptor.filter) {
			continue
		}
		active = append(active, descriptor)
	}
	return active
}

func publishedHeaders(descriptors []column) []string {
	headers := make([]string, 0, len(descriptors))
	for _, descriptor := range descriptors {
		if descriptor.header != "" {
			headers = append(headers, descriptor.header)
		}
	}
	return headers
}

func filterAllows(filter SaveFilter, kind filterKind) bool {
	switch kind {
	case filterUsername:
		return filter.Username
	case filterPassword:
		return filter.Password
	case filterDomain:
		return filter.Domain
	default:
		return true
	}
}

func normalizeFilter(filter *SaveFilter) SaveFilter {
	if filter == nil {
		return SaveFilter{Username: true, Domain: true, Password: true, Inheritance: true}
	}
	return *filter
}

func slicesEqual(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func validateSchema() error {
	valuesType := reflect.TypeOf(connection.ConnectionValues{})
	inheritanceType := reflect.TypeOf(connection.InheritanceFlags{})
	emptyHeaders := 0
	for i, descriptor := range columns {
		if descriptor.source == columnMeta {
			if descriptor.header == "" {
				return fmt.Errorf("%w: metadata column %d has no header", ErrInvalidSchema, i)
			}
			continue
		}
		if descriptor.header == "" {
			emptyHeaders++
		}
		target := valuesType
		if descriptor.source == columnInheritance {
			target = inheritanceType
		}
		field, ok := target.FieldByName(descriptor.field)
		if !ok {
			return fmt.Errorf("%w: column %d references %s", ErrInvalidSchema, i, descriptor.field)
		}
		if descriptor.source == columnInheritance && field.Type.Kind() != reflect.Bool {
			return fmt.Errorf("%w: inheritance field %s is not bool", ErrInvalidSchema, descriptor.field)
		}
	}
	if emptyHeaders != 1 {
		return fmt.Errorf("%w: expected one merged-header value, got %d", ErrInvalidSchema, emptyHeaders)
	}
	return nil
}
