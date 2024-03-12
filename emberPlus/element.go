package ember

import (
	"errors"
	"net"
	"strconv"
	"strings"

	"git.zabbix.com/ap/plugin-support/log"
)

const (
	parameterType = "parameter"
	nodeType      = "node"
	Command
)

type ElementCollection map[string]*Element

type Element struct {
	IsRoot      bool   `json:"is_root,omitempty"`
	IsOnline    bool   `json:"is_online,omitempty"`
	Identifier  string `json:"identifier,omitempty"`
	Description string `json:"description,omitempty"`
	Path        string `json:"path"`
	ElementType string `json:"type,omitempty"`
	Enumeration string `json:"Enumeration,omitempty"`
}

func (ec ElementCollection) PopulateRootElement(conn net.Conn, log log.Logger) error {
	ass1 := &DefaultASN1Codec{}
	ass1.GetRootTreeRequest()

	codec := NewCodec()
	m := codec.Encode(ass1.GetData(), FirstMultiPacket)

	_, err := conn.Write(m)
	if err != nil {
		return err
	}

	response := make([]byte, 1024)
	_, err = conn.Read(response)
	if err != nil {
		return err
	}

	log.Infof("resp %x", response)

	glow, err := codec.Decode(response)
	if err != nil {
		return err
	}

	return ec.Populate(NewASN1Decoder(glow))
}

func (ec ElementCollection) PopulateByPath(conn net.Conn, path string) error {
	ass1 := &DefaultASN1Codec{}

	parsed, err := parsePath(path)
	if err != nil {
		return err
	}

	ass1.GetRequest(parsed)

	codec := NewCodec()
	m := codec.Encode(ass1.GetData(), FirstMultiPacket)
	_, err = conn.Write(m)
	if err != nil {
		return err
	}

	response := make([]byte, 1024)
	_, err = conn.Read(response)
	if err != nil {
		return err
	}

	glow, err := codec.Decode(response)
	if err != nil {
		return err
	}

	respCodec := NewASN1Decoder(glow)

	return ec.Populate(respCodec)
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

	for _, conts := range cnts {
		t, err := conts.data.Peek()
		if err != nil {
			return err
		}

		var appl *DefaultASN1Codec

		switch application(t) {
		case application(qualifiedNodeTag):
			appl, err = conts.data.Read(qualifiedNodeTag, application)
			if err != nil {
				return err
			}

			values, err := appl.ReadAllContext()
			if err != nil {
				return err
			}

			err = ec.handleQualifiedNodeTag(values)
			if err != nil {
				return err
			}
		case application(qualifiedParameterTag):
			appl, err = conts.data.Read(qualifiedParameterTag, application)
			if err != nil {
				return err
			}

			values, err := appl.ReadAllContext()
			if err != nil {
				return err
			}

			err = ec.handleProperty(values)
			if err != nil {
				return err
			}
		case application(nodeTag):
			appl, err = conts.data.Read(nodeTag, application)
			if err != nil {
				return err
			}

			values, err := appl.ReadAllContext()
			if err != nil {
				return err
			}

			err = ec.handleNodeTag(values)
			if err != nil {
				return err
			}
		default:
			return errors.New("unknown type")
		}
	}

	return nil
}

func (ec ElementCollection) handleProperty(values []cntxt) error {
	var el Element

	el.ElementType = parameterType

	for _, node := range values {
		switch node.tag {
		case nodeChildren:
			//currently not implemented
		case nodeProperties:
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

		case nodePath:
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
		case nodeChildren:
			//currently not implemented
		case nodeProperties:
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
					el.Description, err = decodeString(c.data.glow.Bytes())
					if err != nil {
						return err
					}
				}
			}

		case nodePath:
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
			errors.New("incorrect node values")
		}

		ec[el.Path] = &el
	}

	return nil
}

func (ec ElementCollection) handleNodeTag(values []cntxt) error {
	var el Element

	for _, node := range values {
		switch node.tag {
		case nodeChildren:
			//currently not implemented
		case nodeProperties:
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
		case nodePath:
			path, err := node.data.DecodeInteger()
			if err != nil {
				return err
			}

			el.Path = strconv.Itoa(path)
		default:
			errors.New("incorrect node values")
		}

		ec[el.Path] = &el
	}

	return nil
}

func NewElementConnection() ElementCollection {
	return make(ElementCollection)
}
