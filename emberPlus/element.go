package ember

import (
	"errors"
	"strconv"
	"strings"

	"git.zabbix.com/ap/plugin-support/errs"
)

const (
	parameterType = "parameter"
	nodeType      = "node"
	functionType  = "function"
)

var (
	_ RequestFunction = GetRootRequest
	_ RequestFunction = GetPathRequest
)

type (
	Element struct {
		IsRoot      bool   `json:"is_root,omitempty"`
		IsOnline    bool   `json:"is_online,omitempty"`
		Identifier  string `json:"identifier,omitempty"`
		Description string `json:"description,omitempty"`
		Path        string `json:"path"`
		ElementType string `json:"type,omitempty"`
		Enumeration string `json:"Enumeration,omitempty"`
	}

	ElementCollection map[string]*Element

	RequestFunction func(path string) ([]byte, error)

	valueHandlerFunc func(values []cntxt) error
)

func (ec ElementCollection) Populate(data *DefaultASN1Codec) error {
	app0Codec, err := data.Read(rootElementCollectionTag, application)
	if err != nil {
		return err
	}

	app11Codec, err := app0Codec.Read(rootElementTag, application)
	if err != nil {
		return err
	}

	cnts, err := app11Codec.ReadAllContext()
	if err != nil {
		return err
	}

	for _, cont := range cnts {
		t, err := cont.data.Peek()
		if err != nil {
			return err
		}

		switch application(t) {
		case application(qualifiedNodeTag):
			handleApplication(cont, qualifiedNodeTag, ec.handleQualifiedNodeTag)
		case application(qualifiedParameterTag):
			handleApplication(cont, qualifiedParameterTag, ec.handleProperty)
		case application(nodeTag):
			handleApplication(cont, nodeTag, ec.handleNodeTag)
		case application(functionTag):
			handleApplication(cont, functionTag, ec.handleFunction)
		default:
			return errs.Errorf("unknown type: %x", t)
		}
	}

	return nil
}

func NewElementConnection() ElementCollection {
	return make(ElementCollection)
}

func GetRootRequest(_ string) ([]byte, error) {
	asn1 := &DefaultASN1Codec{}
	asn1.GetRootTreeRequest()

	return NewCodec().Encode(asn1.GetData(), FirstMultiPacket), nil
}

func GetPathRequest(path string) ([]byte, error) {
	asn1 := &DefaultASN1Codec{}

	parsed, err := parsePath(path)
	if err != nil {
		return nil, err
	}

	asn1.GetRequest(parsed)

	return NewCodec().Encode(asn1.GetData(), FirstMultiPacket), nil
}

func (ec ElementCollection) handleProperty(values []cntxt) error {
	var el Element

	el.ElementType = parameterType

	for _, node := range values {
		switch node.tag {
		case childrenContextTag:
			//currently not implemented
		case propertiesContextTag:
			el.handlePropertyContext(node)
		case pathContextTag:
			var path []int

			b, err := node.data.Peek()
			if err != nil {
				return err
			}

			switch b {
			case universalObjectTag:
				path, err = node.data.DecodeUniversal()
				if err != nil {
					return err
				}
			case intObjectTag:
				p, err := node.data.DecodeInteger()
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

		ec[el.Path] = &el
	}

	return nil
}

func (ec ElementCollection) handleQualifiedNodeTag(values []cntxt) error {
	var el Element

	el.ElementType = nodeType

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

		ec[el.Path] = &el
	}

	return nil
}

func (ec ElementCollection) handleNodeTag(values []cntxt) error {
	var el Element

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

		ec[el.Path] = &el
	}

	return nil
}

func (ec ElementCollection) handleFunction(values []cntxt) error {
	var el Element

	for _, v := range values {
		switch v.tag {
		case childrenContextTag:
			//currently not implemented
		case propertiesContextTag:
			conts, err := v.data.ReadSet()
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
					/*
						contains other cases that might be required, but these are sequences and require different
						decoding.
					*/
				}
			}
		case pathContextTag:
			path, err := v.data.DecodeInteger()
			if err != nil {
				return err
			}

			el.Path = strconv.Itoa(path)
		default:
			return errors.New("incorrect node values")
		}

		ec[el.Path] = &el
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
