package ember

import (
	"errors"
	"strconv"
	"strings"

	"git.zabbix.com/ap/plugin-support/errs"
)

const (
	ParameterType          = "parameter"
	QualifiedParameterType = "qualified_parameter"
	QualifiedNodeType      = "qualified_node"
	NodeType               = "node"
	FunctionType           = "function"
)

var (
	_ RequestFunction = GetRootRequest
	_ RequestFunction = GetRequestByType
)

type ElementKey struct {
	Id   string
	Path string
}

type (
	Element struct {
		IsRoot      bool        `json:"is_root,omitempty"`
		IsOnline    bool        `json:"is_online,omitempty"`
		Identifier  string      `json:"identifier,omitempty"`
		Description string      `json:"description,omitempty"`
		Path        string      `json:"path"`
		ElementType ElementType `json:"type"`
		Enumeration string      `json:"Enumeration,omitempty"`
	}

	ElementCollection map[ElementKey]*Element

	RequestFunction func(t ElementType, path string) ([]byte, error)

	ElementType string

	valueHandlerFunc func(values []cntxt) error
)

func (ec ElementCollection) Populate(data *DefaultASN1Codec) error {
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
		t, err := cont.data.Peek()
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

func NewElementConnection() ElementCollection {
	return make(ElementCollection)
}

func GetRootRequest(_ ElementType, path string) ([]byte, error) {
	asn1 := &DefaultASN1Codec{}
	asn1.GetRootTreeRequest()

	return NewCodec().Encode(asn1.GetData(), FirstMultiPacket), nil
}

func GetRequestByType(et ElementType, path string) ([]byte, error) {
	asn1 := &DefaultASN1Codec{}

	parsed, err := parsePath(path)
	if err != nil {
		return nil, err
	}

	err = asn1.GetRequest(parsed, et)
	if err != nil {
		return nil, err
	}

	return NewCodec().Encode(asn1.GetData(), FirstMultiPacket), nil
}

func (ec ElementCollection) handleParameter(values []cntxt) error {
	var el Element

	el.ElementType = ParameterType

	for _, v := range values {
		switch v.tag {
		case childrenContextTag:
			//currently not implemented
		case propertiesContextTag:
			el.handlePropertyContext(v)
		case pathContextTag:
			var path []int

			b, err := v.data.Peek()
			if err != nil {
				return err
			}

			switch b {
			case universalObjectTag:
				path, err = v.data.DecodeUniversal()
				if err != nil {
					return err
				}
			case intObjectTag:
				p, err := v.data.DecodeInteger()
				if err != nil {
					return err
				}

				path = append(path, p)
			}

			var strPath []string

			for _, p := range path {
				strPath = append(strPath, strconv.Itoa(p))
			}

			el.Path = strings.Join(strPath, ".")
		default:
			return errors.New("incorrect node tag")
		}

		ec[ElementKey{Id: el.Identifier, Path: el.Path}] = &el
	}

	return nil
}

func (ec ElementCollection) handleQualifiedNodeTag(values []cntxt) error {
	var el Element

	el.ElementType = QualifiedNodeType

	for _, node := range values {
		switch node.tag {
		case childrenContextTag:
			//currently not implemented
		case propertiesContextTag:
			el.handlePropertyContext(node)
		case pathContextTag:
			path, err := node.data.DecodeUniversal()
			if err != nil {
				return err
			}

			var strPath []string

			for _, p := range path {
				strPath = append(strPath, strconv.Itoa(p))
			}

			el.Path = strings.Join(strPath, ".")
		default:
			return errors.New("incorrect node values")
		}

		ec[ElementKey{Id: el.Identifier, Path: el.Path}] = &el
	}

	return nil
}

func (ec ElementCollection) handleNodeTag(values []cntxt) error {
	var el Element

	el.ElementType = NodeType

	for _, node := range values {
		switch node.tag {
		case childrenContextTag:
			//currently not implemented
		case propertiesContextTag:
			conts, err := node.data.ReadSet()
			if err != nil {
				return err
			}

			for _, c := range conts {
				switch context(uint8(c.tag)) {
				case context(0):
					el.Identifier, err = decodeString(c.data.glow.Bytes())
					if err != nil {
						return err
					}
				case context(3):
					el.IsOnline, err = decodeBool(c.data.glow.Bytes())
					if err != nil {
						return err
					}
				}
			}
		case pathContextTag:
			path, err := node.data.DecodeInteger()
			if err != nil {
				return err
			}

			el.Path = strconv.Itoa(path)
		default:
			return errors.New("incorrect node values")
		}

		ec[ElementKey{Id: el.Identifier, Path: el.Path}] = &el
	}

	return nil
}

func (ec ElementCollection) handleFunction(values []cntxt) error {
	var el Element

	el.ElementType = FunctionType

	for _, v := range values {
		switch v.tag {
		case childrenContextTag:
			//currently not implemented
		case propertiesContextTag:
			conts, err := v.data.ReadSet()
			if err != nil {
				return errs.Wrapf(err, "failed to read set")
			}

			for _, c := range conts {
				switch context(uint8(c.tag)) {
				case context(0):
					el.Identifier, err = decodeString(c.data.glow.Bytes())
					if err != nil {
						return errs.Wrapf(err, "failed to decode string")
					}
					/*
						contains other cases that might be required, but these are sequences and require different
						decoding.
					*/
				}
			}
		case pathContextTag:
			path, err := v.data.DecodeUniversal()
			if err != nil {
				return errs.Wrapf(err, "failed to decode integer")
			}

			var strPath []string

			for _, p := range path {
				strPath = append(strPath, strconv.Itoa(p))
			}

			el.Path = strings.Join(strPath, ".")
		default:
			return errors.New("incorrect node values")
		}

		ec[ElementKey{Id: el.Identifier, Path: el.Path}] = &el
	}

	return nil
}

func (el *Element) handlePropertyContext(node cntxt) error {
	conts, err := node.data.ReadSet()
	if err != nil {
		return err
	}

	for _, c := range conts {
		switch context(uint8(c.tag)) {
		case context(0):
			el.Identifier, err = decodeString(c.data.glow.Bytes())
			if err != nil {
				return err
			}
		case context(1):
			el.Description, err = decodeString(c.data.glow.Bytes())
			if err != nil {
				return err
			}
		case context(7):
			el.Enumeration, err = decodeString(c.data.glow.Bytes())
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func handleApplication(cont cntxt, tag uint8, valHandler valueHandlerFunc) error {
	appl, err := cont.data.Read(tag, application)
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

func parsePath(path string) ([]int, error) {
	if len(path) == 0 {
		return nil, nil
	}

	paths := strings.Split(path, ".")
	var out []int

	for _, p := range paths {
		i, err := strconv.Atoi(p)
		if err != nil {
			return nil, err
		}

		out = append(out, i)
	}

	return out, nil
}
