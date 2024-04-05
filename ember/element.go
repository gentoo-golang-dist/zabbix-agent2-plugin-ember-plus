package ember

import (
	"strconv"
	"strings"

	"git.zabbix.com/ap/ember-plus/ember/asn1"
	"git.zabbix.com/ap/ember-plus/ember/s101"
	"git.zabbix.com/ap/plugin-support/errs"
)

var (
	_ RequestFunction = GetRootRequest
	_ RequestFunction = GetRequestByType

	_ valueHandlerFunc = ElementCollection(nil).handleQualifiedNodeTag
	_ valueHandlerFunc = ElementCollection(nil).handleParameter
	_ valueHandlerFunc = ElementCollection(nil).handleNodeTag
	_ valueHandlerFunc = ElementCollection(nil).handleFunction
)

// ElementKey used for element identification based on either element id or path.
type ElementKey struct {
	ID   string
	Path string
}

// Element contains all the values a glow element might contain.
type (
	Element struct {
		//nolint:tagliatelle
		IsOnline    bool        `json:"is_online,omitempty"`
		Identifier  string      `json:"identifier,omitempty"`
		Description string      `json:"description,omitempty"`
		Path        string      `json:"path"`
		ElementType ElementType `json:"type"`
		Enumeration string      `json:"enumeration,omitempty"`
	}

	// ElementCollection contains one level of elements and their Ids as key.
	ElementCollection map[ElementKey]*Element

	// RequestFunction request type for data retrieval.
	RequestFunction func(t ElementType, path string) ([]byte, error)

	// ElementType wrapper for string to define available element types.
	ElementType string

	// valueHandlerFunc functions used to handle different glow value types.
	valueHandlerFunc func(values []asn1.Context) error
)

// Populate filles in collection with data from the decoder.
//
//nolint:gocyclo,cyclop
func (ec ElementCollection) Populate(data *asn1.Decoder) error {
	app0Codec, err := data.Read(asn1.RootElementCollectionTag, asn1.ApplicationByte)
	if err != nil {
		return errs.Wrapf(err, "failed to read element root collection tag")
	}

	app11Codec, err := app0Codec.Read(asn1.RootElementTag, asn1.ApplicationByte)
	if err != nil {
		return errs.Wrapf(err, "failed to read element tag")
	}

	cnts, err := app11Codec.ReadAllContext()
	if err != nil {
		return errs.Wrapf(err, "failed to read all contexts")
	}

	for _, cont := range cnts {
		t, err := cont.Value.Peek()
		if err != nil {
			return errs.Wrapf(err, "failed to read context")
		}

		switch asn1.ApplicationByte(t) {
		case asn1.ApplicationByte(asn1.QualifiedNodeTag):
			err = handleApplication(cont, asn1.QualifiedNodeTag, ec.handleQualifiedNodeTag)
			if err != nil {
				return errs.Wrapf(err, "failed to handle qualified node")
			}
		case asn1.ApplicationByte(asn1.QualifiedParameterTag):
			err = handleApplication(cont, asn1.QualifiedParameterTag, ec.handleParameter)
			if err != nil {
				return errs.Wrapf(err, "failed to handle qualified parameter")
			}
		case asn1.ApplicationByte(nodeTag):
			err = handleApplication(cont, nodeTag, ec.handleNodeTag)
			if err != nil {
				return errs.Wrapf(err, "failed to handle node tag")
			}
		case asn1.ApplicationByte(functionTag):
			err = handleApplication(cont, functionTag, ec.handleFunction)
			if err != nil {
				return errs.Wrapf(err, "failed to handle function tag")
			}
		default:
			return errs.Errorf("unknown type: %x", t)
		}
	}

	return nil
}

// NewElementConnection creates a empty element collection.
func NewElementConnection() ElementCollection {
	return make(ElementCollection)
}

// GetRootRequest returns a S101 request packet with an encoded request for root collection.
func GetRootRequest(_ ElementType, _ string) ([]byte, error) {
	asn1 := asn1.NewEncoder()
	err := asn1.WriteRootTreeRequest()

	if err != nil {
		return nil, errs.Wrap(err, "failed to write root command request")
	}

	return s101.Encode(asn1.GetData(), s101.FirstMultiPacket), nil
}

// GetRequestByType returns S101 packet with an encoded request for element with the provided type and path.
func GetRequestByType(et ElementType, path string) ([]byte, error) {
	asn1 := asn1.NewEncoder()

	parsed, err := parsePath(path)
	if err != nil {
		return nil, errs.Wrap(err, "failed to parse path")
	}

	err = asn1.WriteRequest(parsed, string(et))
	if err != nil {
		return nil, errs.Wrap(err, "failed to write request")
	}

	return s101.Encode(asn1.GetData(), s101.FirstMultiPacket), nil
}

// handleParameter used to decodes context data for parameters.
func (ec ElementCollection) handleParameter(contexts []asn1.Context) error {
	var el Element

	el.ElementType = asn1.ParameterType

	for _, c := range contexts {
		switch c.Tag {
		case childrenContextTag:
			// currently not implemented
		case propertiesContextTag:
			err := el.handlePropertyContext(c)
			if err != nil {
				return errs.Wrap(err, "failed to handle property")
			}
		case pathContextTag:
			err := el.handleParameterPath(c)
			if err != nil {
				return errs.Wrap(err, "failed to handle property")
			}
		default:
			return errs.New("incorrect parameter tag")
		}

		ec[ElementKey{ID: el.Identifier, Path: el.Path}] = &el
	}

	return nil
}

// handleQualifiedNodeTag used to decodes context data for qualified node tags.
func (ec ElementCollection) handleQualifiedNodeTag(contexts []asn1.Context) error {
	var el Element

	el.ElementType = asn1.QualifiedNodeType

	for _, c := range contexts {
		switch c.Tag {
		case childrenContextTag:
			// currently not implemented
		case propertiesContextTag:
			err := el.handlePropertyContext(c)
			if err != nil {
				return errs.Wrap(err, "failed to handle property")
			}
		case pathContextTag:
			err := el.handlePathFromUniversal(c)
			if err != nil {
				return errs.Wrap(err, "failed to handle property")
			}
		default:
			return errs.New("incorrect node values")
		}

		ec[ElementKey{ID: el.Identifier, Path: el.Path}] = &el
	}

	return nil
}

// handleNodeTag used to decodes context data for node tags.
func (ec ElementCollection) handleNodeTag(contexts []asn1.Context) error {
	var (
		el  Element
		err error
	)

	el.ElementType = asn1.NodeType

	for _, c := range contexts {
		switch c.Tag {
		case childrenContextTag:
			// currently not implemented
		case propertiesContextTag:
			err = el.handleNodeContext(c)
			if err != nil {
				return errs.New("failed to get properties values from set")
			}
		case pathContextTag:
			path, err := c.Value.DecodeInteger()
			if err != nil {
				return err
			}

			el.Path = strconv.Itoa(path)
		default:
			return errs.New("incorrect node values")
		}

		ec[ElementKey{ID: el.Identifier, Path: el.Path}] = &el
	}

	return nil
}

// handleFunction used to decodes context data for functions.
func (ec ElementCollection) handleFunction(contexts []asn1.Context) error {
	var el Element

	el.ElementType = asn1.FunctionType

	for _, c := range contexts {
		switch c.Tag {
		case childrenContextTag:
			// currently not implemented
		case propertiesContextTag:
			err := el.handleFunctionContextSetTag(c)
			if err != nil {
				return errs.Wrapf(err, "failed to read identifier")
			}
		case pathContextTag:
			err := el.handlePathFromUniversal(c)
			if err != nil {
				return errs.Wrapf(err, "failed to decode integer")
			}
		default:
			return errs.New("incorrect node values")
		}

		ec[ElementKey{ID: el.Identifier, Path: el.Path}] = &el
	}

	return nil
}

func (el *Element) handleParameterPath(c asn1.Context) error {
	var path []int

	b, err := c.Value.Peek()
	if err != nil {
		return errs.Wrap(err, "failed to read path")
	}

	switch b {
	case asn1.UniversalObjectTag:
		path, err = c.Value.DecodeUniversal()
		if err != nil {
			return errs.Wrap(err, "failed to universal")
		}
	case asn1.IntObjectTag:
		p, err := c.Value.DecodeInteger()
		if err != nil {
			return errs.Wrap(err, "failed to decode int")
		}

		path = append(path, p)
	}

	strPath := make([]string, 0, len(path))

	for _, p := range path {
		strPath = append(strPath, strconv.Itoa(p))
	}

	el.Path = strings.Join(strPath, ".")

	return nil
}

func (el *Element) handleNodeContext(c asn1.Context) error {
	conts, err := c.Value.ReadSet()
	if err != nil {
		return errs.Wrap(err, "failed to read set")
	}

	for _, c := range conts {
		switch asn1.ContextByte(uint8(c.Tag)) {
		case asn1.ContextByte(0):
			id, err := asn1.DecodeString(c.Bytes())
			if err != nil {
				return errs.Wrap(err, "failed to decode identifier")
			}

			el.Identifier = id
		case asn1.ContextByte(3):
			isOnline, err := asn1.DecodeBool(c.Bytes())
			if err != nil {
				return errs.Wrap(err, "failed to decode is online ")
			}

			el.IsOnline = isOnline
		}
	}

	return nil
}

func (el *Element) handleFunctionContextSetTag(c asn1.Context) error {
	pContexts, err := c.Value.ReadSet()
	if err != nil {
		return errs.Wrapf(err, "failed to read set")
	}

	for _, pc := range pContexts {
		if asn1.ContextByte(uint8(c.Tag)) == asn1.ContextByte(0) {
			identifier, err := asn1.DecodeString(pc.Bytes())
			if err != nil {
				return errs.Wrapf(err, "failed to decode string")
			}

			el.Identifier = identifier

			break
		}
	}

	return nil
}

func (el *Element) handlePathFromUniversal(c asn1.Context) error {
	path, err := c.Value.DecodeUniversal()
	if err != nil {
		return errs.Wrapf(err, "failed to decode integer")
	}

	strPath := make([]string, 0, len(path))

	for _, p := range path {
		strPath = append(strPath, strconv.Itoa(p))
	}

	el.Path = strings.Join(strPath, ".")

	return nil
}

// handlePropertyContext decodes context property tag.
func (el *Element) handlePropertyContext(node asn1.Context) error {
	conts, err := node.Value.ReadSet()
	if err != nil {
		return err
	}

	for _, c := range conts {
		switch asn1.ContextByte(uint8(c.Tag)) {
		case asn1.ContextByte(0):
			el.Identifier, err = asn1.DecodeString(c.Bytes())
			if err != nil {
				return err
			}
		case asn1.ContextByte(1):
			el.Description, err = asn1.DecodeString(c.Bytes())
			if err != nil {
				return err
			}
		case asn1.ContextByte(7):
			el.Enumeration, err = asn1.DecodeString(c.Bytes())
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// handleApplication decodes application from context based on the value handler function and the application tag.
func handleApplication(cont asn1.Context, tag uint8, valHandler valueHandlerFunc) error {
	appl, err := cont.Value.Read(tag, asn1.ApplicationByte)
	if err != nil {
		return err
	}

	values, err := appl.ReadAllContext()
	if err != nil {
		return err
	}

	err = valHandler(values)
	if err != nil {
		return err
	}

	return nil
}

// parsePath returns string oid path as integer array.
func parsePath(path string) ([]int, error) {
	if path == "" {
		return nil, nil
	}

	paths := strings.Split(path, ".")
	out := make([]int, 0, len(paths))

	for _, p := range paths {
		i, err := strconv.Atoi(p)
		if err != nil {
			return nil, errs.Wrap(err, "failed to parse path component")
		}

		out = append(out, i)
	}

	return out, nil
}
