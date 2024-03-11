package ember

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

const (
	parameterType = "parameter"
	nodeType      = "node"
	Command
)

type ElementType int

type ElementCollection map[string]*Element

type Element struct {
	IsRoot              bool
	IsOnline            bool
	Identifier          string
	Description         string
	Path                string
	ElementType         string
	Enumeration         string
	ElementTreeChildren ElementCollection
	Classification      ElementType
	Children            interface{}
}

func (ec ElementCollection) PopulateRootElement(conn net.Conn) {
	ass1 := &DefaultASN1Codec{}
	ass1.GetRootTreeRequest()

	codec := NewCodec()
	m := codec.Encode(ass1.GetData(), FirstMultiPacket)

	_, err := conn.Write(m)
	if err != nil {
		panic(err)
	}

	response := make([]byte, 1024) // Adjust the buffer size as needed
	_, err = conn.Read(response)
	if err != nil {
		panic(err)
	}

	glow, err := codec.Decode(response)
	if err != nil {
		panic(err)
	}

	respCodec := NewASN1Decoder(glow)

	ec.Populate(respCodec)
}

func (ec ElementCollection) PopulateByPath(conn net.Conn, path string) {
	ass1 := &DefaultASN1Codec{}

	parsed, err := parsePath(path)
	if err != nil {
		panic(err)
	}

	ass1.GetRequest(parsed)

	codec := NewCodec()
	m := codec.Encode(ass1.GetData(), FirstMultiPacket)
	_, err = conn.Write(m)
	if err != nil {
		panic(err)
	}

	response := make([]byte, 1024) // Adjust the buffer size as needed
	_, err = conn.Read(response)
	if err != nil {
		panic(err)
	}
	fmt.Println("got here")

	glow, err := codec.Decode(response)
	if err != nil {
		panic(err)
	}

	respCodec := NewASN1Decoder(glow)

	ec.Populate(respCodec)
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

func (ec ElementCollection) Populate(data *DefaultASN1Codec) {
	app0Codec, err := data.Read(rootElementCollectionTag, application)
	if err != nil {
		panic(err)
	}

	app11Codec, err := app0Codec.Read(rootElementTag, application)
	if err != nil {
		panic(err)
	}

	cnts, err := app11Codec.ReadAllContext()
	if err != nil {
		panic(err)
	}

	for _, conts := range cnts {

		t, err := conts.data.Peek()
		if err != nil {
			panic(err)
		}

		var appl *DefaultASN1Codec

		switch application(t) {
		case application(qualifiedNodeTag):
			appl, err = conts.data.Read(qualifiedNodeTag, application)
			if err != nil {
				panic(err)
			}

			values, err := appl.ReadAllContext()
			if err != nil {
				panic(err)
			}

			ec.handleNode(values)
		case application(qualifiedParameterTag):
			appl, err = conts.data.Read(qualifiedParameterTag, application)
			if err != nil {
				panic(err)
			}

			values, err := appl.ReadAllContext()
			if err != nil {
				panic(err)
			}

			ec.handleProperty(values)
		default:
			panic("unknown type")
		}

	}
}

func (ec ElementCollection) handleProperty(values []cntxt) {
	var el Element

	el.ElementType = parameterType

	for _, node := range values {
		switch node.tag {
		case nodeChildren:
			//currently not implemented
		case nodeProperties:
			conts, err := node.data.ReadSet()
			if err != nil {
				panic(err)
			}

			for _, c := range conts {
				switch context(uint8(c.tag)) {
				case context(0):
					el.Identifier, err = decodeString(c.data.glow.Bytes())
					if err != nil {
						panic(err)
					}
				case context(1):
					el.Description, err = decodeString(c.data.glow.Bytes())
					if err != nil {
						panic(err)
					}
				case context(7):
					el.Enumeration, err = decodeString(c.data.glow.Bytes())
					if err != nil {
						panic(err)
					}
				}
			}

		case nodePath:
			path, err := node.data.DecodeUniversal()
			if err != nil {
				panic(err)
			}

			var strPath []string

			for _, p := range path {
				strPath = append(strPath, strconv.Itoa(p))
			}

			el.Path = strings.Join(strPath, ".")
		default:
			panic("incorrect node values")
		}

		ec[el.Path] = &el
	}
}

func (ec ElementCollection) handleNode(values []cntxt) {
	var el Element

	el.ElementType = nodeType

	for _, node := range values {
		switch node.tag {
		case nodeChildren:
			//currently not implemented
		case nodeProperties:
			conts, err := node.data.ReadSet()
			if err != nil {
				panic(err)
			}

			for _, c := range conts {
				switch context(uint8(c.tag)) {
				case context(0):
					el.Identifier, err = decodeString(c.data.glow.Bytes())
					if err != nil {
						panic(err)
					}
				case context(1):
					el.Description, err = decodeString(c.data.glow.Bytes())
					if err != nil {
						panic(err)
					}
				case context(7):
					el.Description, err = decodeString(c.data.glow.Bytes())
					if err != nil {
						panic(err)
					}
				}
			}

		case nodePath:
			path, err := node.data.DecodeUniversal()
			if err != nil {
				panic(err)
			}

			var strPath []string

			for _, p := range path {
				strPath = append(strPath, strconv.Itoa(p))
			}

			el.Path = strings.Join(strPath, ".")
		default:
			panic("incorrect node values")
		}

		ec[el.Path] = &el
	}
}

func NewElementConnection() ElementCollection {
	return make(ElementCollection)
}
