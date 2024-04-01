//nolint:gci,gofmt
package ember

import (
	"strconv"
	"strings"

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
	valueHandlerFunc func(values []Context) error
)

// Populate filles in collection with data from the decoder.
func (ec ElementCollection) Populate(data *ASN1Decoder) error {
	app0Codec, err := data.Read(rootElementCollectionTag, application)
	if err != nil {
		return errs.Wrapf(err, "failed to read element root collection tag")
	}

	app11Codec, err := app0Codec.Read(rootElementTag, application)
	if err != nil {
		return errs.Wrapf(err, "failed to read element tag")
	}

	cnts, err := app11Codec.ReadAllContext()
	if err != nil {
		return errs.Wrapf(err, "failed to read all contexts")
	}

	for _, cont := range cnts {
		t, err := cont.value.Peek()
		if err != nil {
			return errs.Wrapf(err, "failed to read context")
		}

		switch application(t) {
		case application(qualifiedNodeTag):
			err = handleApplication(cont, qualifiedNodeTag, ec.handleQualifiedNodeTag)
			if err != nil {
				return errs.Wrapf(err, "failed to handle qualified node")
			}
		case application(qualifiedParameterTag):
			err = handleApplication(cont, qualifiedParameterTag, ec.handleParameter)
			if err != nil {
				return errs.Wrapf(err, "failed to handle qualified parameter")
			}
		case application(nodeTag):
			err = handleApplication(cont, nodeTag, ec.handleNodeTag)
			if err != nil {
				return errs.Wrapf(err, "failed to handle node tag")
			}
		case application(functionTag):
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
	asn1 := NewASN1Encoder()
	err := asn1.WriteRootTreeRequest()

	if err != nil {
		return nil, errs.Wrap(err, "failed to write root command request")
	}

	return NewCodec().Encode(asn1.GetData(), FirstMultiPacket), nil
}

// GetRequestByType returns S101 packet with an encoded request for element with the provided type and path.
func GetRequestByType(et ElementType, path string) ([]byte, error) {
	asn1 := NewASN1Encoder()

	parsed, err := parsePath(path)
	if err != nil {
		return nil, errs.Wrap(err, "failed to parse path")
	}

	err = asn1.WriteRequest(parsed, et)
	if err != nil {
		return nil, errs.Wrap(err, "failed to write request")
	}

	return NewCodec().Encode(asn1.GetData(), FirstMultiPacket), nil
}

// handleParameter used to decodes context data for parameters.
func (ec ElementCollection) handleParameter(contexts []Context) error {
	var el Element

	el.ElementType = ParameterType

	for _, c := range contexts {
		switch c.tag {
		case childrenContextTag:
			// currently not implemented
		case propertiesContextTag:
			err := el.handlePropertyContext(c)
			if err != nil {
				return errs.Wrap(err, "failed to handle property")
			}
		case pathContextTag:
			var path []int

			b, err := c.value.Peek()
			if err != nil {
				return errs.Wrap(err, "failed to read path")
			}

			switch b {
			case universalObjectTag:
				path, err = c.value.DecodeUniversal()
				if err != nil {
					return errs.Wrap(err, "failed to universal")
				}
			case intObjectTag:
				p, err := c.value.DecodeInteger()
				if err != nil {
					return errs.Wrap(err, "failed to decode int")
				}

				path = append(path, p)
			}

			var strPath []string

			for _, p := range path {
				strPath = append(strPath, strconv.Itoa(p))
			}

			el.Path = strings.Join(strPath, ".")
		default:
			return errs.New("incorrect parameter tag")
		}

		ec[ElementKey{ID: el.Identifier, Path: el.Path}] = &el
	}

	return nil
}

// handleQualifiedNodeTag used to decodes context data for qualified node tags.
func (ec ElementCollection) handleQualifiedNodeTag(contexts []Context) error {
	var el Element

	el.ElementType = QualifiedNodeType

	for _, c := range contexts {
		switch c.tag {
		case childrenContextTag:
			// currently not implemented
		case propertiesContextTag:
			err := el.handlePropertyContext(c)
			if err != nil {
				return errs.Wrap(err, "failed to handle property")
			}
		case pathContextTag:
			path, err := c.value.DecodeUniversal()
			if err != nil {
				return errs.Wrap(err, "failed to get path")
			}

			var strPath []string

			for _, p := range path {
				strPath = append(strPath, strconv.Itoa(p))
			}

			el.Path = strings.Join(strPath, ".")
		default:
			return errs.New("incorrect node values")
		}

		ec[ElementKey{ID: el.Identifier, Path: el.Path}] = &el
	}

	return nil
}

// handleNodeTag used to decodes context data for node tags.
func (ec ElementCollection) handleNodeTag(contexts []Context) error {
	var el Element

	el.ElementType = NodeType

	for _, c := range contexts {
		switch c.tag {
		case childrenContextTag:
			// currently not implemented
		case propertiesContextTag:
			conts, err := c.value.ReadSet()
			if err != nil {
				return err
			}

			for _, c := range conts {
				switch context(uint8(c.tag)) {
				case context(0):
					el.Identifier, err = decodeString(c.value.data.Bytes())
					if err != nil {
						return err
					}
				case context(3):
					el.IsOnline, err = decodeBool(c.value.data.Bytes())
					if err != nil {
						return err
					}
				}
			}
		case pathContextTag:
			path, err := c.value.DecodeInteger()
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
func (ec ElementCollection) handleFunction(contexts []Context) error {
	var el Element

	el.ElementType = FunctionType

	for _, c := range contexts {
		switch c.tag {
		case childrenContextTag:
			// currently not implemented
		case propertiesContextTag:
			pContexts, err := c.value.ReadSet()
			if err != nil {
				return errs.Wrapf(err, "failed to read set")
			}

			for _, pc := range pContexts {
				if context(uint8(c.tag)) == context(0) {
					el.Identifier, err = decodeString(pc.value.data.Bytes())
					if err != nil {
						return errs.Wrapf(err, "failed to decode string")
					}

					break
				}
			}
		case pathContextTag:
			path, err := c.value.DecodeUniversal()
			if err != nil {
				return errs.Wrapf(err, "failed to decode integer")
			}

			var strPath []string

			for _, p := range path {
				strPath = append(strPath, strconv.Itoa(p))
			}

			el.Path = strings.Join(strPath, ".")
		default:
			return errs.New("incorrect node values")
		}

		ec[ElementKey{ID: el.Identifier, Path: el.Path}] = &el
	}

	return nil
}

// handlePropertyContext decodes context property tag.
func (el *Element) handlePropertyContext(node Context) error {
	conts, err := node.value.ReadSet()
	if err != nil {
		return err
	}

	for _, c := range conts {
		switch context(uint8(c.tag)) {
		case context(0):
			el.Identifier, err = decodeString(c.value.data.Bytes())
			if err != nil {
				return err
			}
		case context(1):
			el.Description, err = decodeString(c.value.data.Bytes())
			if err != nil {
				return err
			}
		case context(7):
			el.Enumeration, err = decodeString(c.value.data.Bytes())
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// handleApplication decodes application from context based on the value handler function and the application tag.
func handleApplication(cont Context, tag uint8, valHandler valueHandlerFunc) error {
	appl, err := cont.value.Read(tag, application)
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
