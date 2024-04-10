/*
** Zabbix
** Copyright 2001-2024 Zabbix SIA
**
** Licensed under the Apache License, Version 2.0 (the "License");
** you may not use this file except in compliance with the License.
** You may obtain a copy of the License at
**
**     http://www.apache.org/licenses/LICENSE-2.0
**
** Unless required by applicable law or agreed to in writing, software
** distributed under the License is distributed on an "AS IS" BASIS,
** WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
** See the License for the specific language governing permissions and
** limitations under the License.
**/

package ember

import (
	"strconv"
	"strings"

	"git.zabbix.com/ap/ember-plus/ember/asn1"
	"git.zabbix.com/ap/ember-plus/ember/s101"
	"git.zabbix.com/ap/plugin-support/errs"
)

const (
	// tag for defining glow node tag.
	nodeTag = 3
	// tag for defining glow function tag.
	functionTag = 20
	// parameterTag glow  parameter tag.
	parameterTag = 1
)

var (
	_ decoderHandlerFunc = (*Element)(nil).handlePath
	_ decoderHandlerFunc = (*Element)(nil).handleContent
	_ decoderHandlerFunc = (*Element)(nil).handleChildren
	_ decoderHandlerFunc = (*Element)(nil).setChild

	//nolint:gochecknoglobals
	notFoundErr = errs.New("element not found")
)

// ElementKey used for element identification based on either element id or path.
type ElementKey struct {
	ID   string
	Path string
}

// Element contains all the values a glow element might contain.
type (
	Element struct {
		Path        string      `json:"path"`
		ElementType ElementType `json:"element_type"`
		Identifier  string      `json:"identifier,omitempty"`
		Description string      `json:"description,omitempty"`
		Children    []*Element  `json:"children,omitempty"`

		IsOnline    bool   `json:"is_online,omitempty"`
		IsRoot      bool   `json:"is_root,omitempty"`
		Maximum     any    `json:"maximum,omitempty"`
		Minimum     any    `json:"minimum,omitempty"`
		Value       any    `json:"value,omitempty"`
		Access      int    `json:"access,omitempty"`
		Format      string `json:"format,omitempty"`
		Enumeration string `json:"enumeration,omitempty"`
		Factor      int    `json:"factor,omitempty"`
		Default     any    `json:"default,omitempty"`
		ValueType   int    `json:"type,omitempty"`
	}

	// ElementCollection contains one level of elements and their Ids as key.
	ElementCollection map[ElementKey]*Element

	// RequestFunction request type for data retrieval.
	RequestFunction func(t ElementType, path string) ([]byte, error)

	// ElementType wrapper for string to define available element types.
	ElementType string

	// decoderHandlerFunc functions used for wrapped to handle decoders with leftover data.
	decoderHandlerFunc func(values *asn1.Decoder) ([]*asn1.Decoder, error)
)

// Populate fills in collection with data from the decoder.
func (ec ElementCollection) Populate(data *asn1.Decoder) error {
	app0Codec, _, err := data.Read(asn1.RootElementCollectionTag, asn1.ApplicationByte)
	if err != nil {
		return errs.Wrap(err, "failed to read element root collection tag")
	}

	app11Codec, _, err := app0Codec.Read(asn1.RootElementTag, asn1.ApplicationByte)
	if err != nil {
		return errs.Wrap(err, "failed to read element tag")
	}

	for {
		context0, _, err := app11Codec.Read(asn1.ContextZeroTag, asn1.ContextByte)
		if err != nil {
			return errs.Wrap(err, "failed to read top level context 0")
		}

		el, decoder, err := getElement(context0)
		if err != nil {
			return errs.Wrap(err, "failed to read element")
		}

		ec[ElementKey{ID: el.Identifier, Path: el.Path}] = el

		_, err = decoder.ReadEnd() // current context end
		if err != nil {
			return errs.Wrap(err, "failed to decode context end")
		}

		_, err = decoder.ReadEnd() // current elements end
		if err != nil {
			return errs.Wrap(err, "failed to decode current sequence end")
		}

		end, err := app11Codec.ReadEnd() // all  element end
		if err != nil {
			return errs.Wrap(err, "failed to decode element sequence end")
		}

		if end {
			break
		}
	}

	return nil
}

// GetElementByPath returns element from collection with the provided path OID.
func (ec ElementCollection) GetElementByPath(currentPath string) (*Element, error) {
	for key, value := range ec {
		if key.Path == currentPath {
			return value, nil
		}
	}

	return nil, notFoundErr
}

// GetElementByID returns element from collection with the provided identifier.
func (ec ElementCollection) GetElementByID(id string) (*Element, error) {
	for key, value := range ec {
		if key.ID == id {
			return value, nil
		}
	}

	return nil, notFoundErr
}

// ToJSONCompatible returns the collection with path(string) in key value instead of a structure for json marshaling.
func (ec ElementCollection) ToJSONCompatible() map[string]*Element {
	out := make(map[string]*Element)
	for k, v := range ec {
		out[k.Path] = v
	}

	return out
}

// NewElementConnection creates a empty element collection.
func NewElementConnection() ElementCollection {
	return make(ElementCollection)
}

// GetRootRequest returns a S101 request packet with an encoded request for root collection.
func GetRootRequest(_ ElementType, _ string) ([]byte, error) {
	encoder := asn1.NewEncoder()

	err := encoder.WriteRootTreeRequest()
	if err != nil {
		return nil, errs.Wrap(err, "failed to write root command request")
	}

	return s101.Encode(encoder.GetData(), s101.FirstMultiPacket), nil
}

// GetRequestByType returns S101 packet with an encoded request for element with the provided type and path.
func GetRequestByType(et ElementType, path string) ([]byte, error) {
	encoder := asn1.NewEncoder()

	parsed, err := parsePath(path)
	if err != nil {
		return nil, errs.Wrap(err, "failed to parse path")
	}

	err = encoder.WriteRequest(parsed, string(et))
	if err != nil {
		return nil, errs.Wrap(err, "failed to write request")
	}

	return s101.Encode(encoder.GetData(), s101.FirstMultiPacket), nil
}

//nolint:gocyclo,cyclop
func (el *Element) handleApplication(decoder *asn1.Decoder) (*asn1.Decoder, error) {
	for {
		t, err := decoder.Peek()
		if err != nil {
			return nil, errs.Wrapf(err, "failed to peek context")
		}

		switch asn1.ContextByte(t) {
		case asn1.ContextByte(asn1.ContextZeroTag):
			decoder, err = decoderWrapper(decoder, el.handlePath)
			if err != nil {
				return nil, errs.Wrapf(err, "failed to read path")
			}
		case asn1.ContextByte(asn1.ContextTagOne):
			decoder, err = decoderWrapper(decoder, el.handleContent)
			if err != nil {
				return nil, errs.Wrapf(err, "failed to read content")
			}
		case asn1.ContextByte(asn1.ContextTagTwo):
			decoder, err = decoderWrapper(decoder, el.handleChildren)
			if err != nil {
				return nil, errs.Wrapf(err, "failed to read children")
			}
		}

		atEnd, err := decoder.ReadEnd()
		if err != nil {
			return nil, errs.Wrap(err, "failed to read end bytes")
		}

		if atEnd {
			return decoder, nil
		}
	}
}

func (el *Element) handleChildren(decoder *asn1.Decoder) ([]*asn1.Decoder, error) {
	anyDec, _, err := decoder.Read(asn1.ContextByte(asn1.ContextTagTwo), asn1.ContextByte)
	if err != nil {
		return nil, errs.Wrapf(err, "failed to read child context")
	}

	childDec, _, err := anyDec.Read(asn1.ApplicationByte(asn1.ElementCollectionTag), asn1.ApplicationByte)
	if err != nil {
		return nil, errs.Wrapf(err, "failed to get children elements")
	}

	for {
		childDec, err = decoderWrapper(childDec, el.setChild)
		if err != nil {
			return nil, errs.Wrapf(err, "failed to get child")
		}

		_, err := childDec.ReadEnd() // current child context end
		if err != nil {
			return nil, errs.Wrapf(err, "failed to decode child context end")
		}

		_, err = childDec.ReadEnd() // current child elements end
		if err != nil {
			return nil, errs.Wrapf(err, "failed to decode current child sequence end")
		}

		end, err := childDec.ReadEnd() // all child element end
		if err != nil {
			return nil, errs.Wrapf(err, "failed to decode child element sequence end")
		}

		if end {
			break
		}
	}

	return []*asn1.Decoder{decoder, anyDec, childDec}, nil
}

func (el *Element) setChild(childrenDecoder *asn1.Decoder) ([]*asn1.Decoder, error) {
	allChild, _, err := childrenDecoder.Read(asn1.ContextByte(asn1.ContextZeroTag), asn1.ContextByte)
	if err != nil {
		return nil, errs.Wrapf(err, "failed to read next child element")
	}

	child, tmp, err := getElement(allChild)
	if err != nil {
		return nil, errs.Wrapf(err, "failed to decode next child element")
	}

	el.Children = append(el.Children, child)

	return []*asn1.Decoder{childrenDecoder, allChild, tmp}, nil
}

func (el *Element) handleContent(decoder *asn1.Decoder) ([]*asn1.Decoder, error) {
	content, _, err := decoder.Read(asn1.ContextByte(asn1.ContextTagOne), asn1.ContextByte)
	if err != nil {
		return nil, errs.Wrap(err, "failed to read context")
	}

	set, _, err := content.Read(asn1.SetTag, asn1.UniversalByte)
	if err != nil {
		return nil, errs.Wrap(err, "failed to read set")
	}

	for {
		set, err = decoderWrapper(set, el.handleContext)
		if err != nil {
			return nil, errs.Wrap(err, "failed to read set element")
		}

		end, err := set.ReadEnd()
		if err != nil {
			return nil, errs.Wrap(err, "failed to read sequence end")
		}

		if end {
			break
		}
	}

	return []*asn1.Decoder{decoder, content, set}, nil
}

func (el *Element) handleContext(decoder *asn1.Decoder) ([]*asn1.Decoder, error) {
	t, err := decoder.Peek()
	if err != nil {
		return nil, errs.Wrapf(err, "failed to node read context")
	}

	context, _, err := decoder.Read(t, asn1.ContextByte)
	if err != nil {
		return nil, errs.Wrap(err, "failed to read context")
	}

	switch el.ElementType {
	case asn1.QualifiedParameterType, asn1.ParameterType:
		err = el.handleParameterContext(context, t)
		if err != nil {
			return nil, errs.Wrap(err, "failed to decode parameter context")
		}
	case asn1.NodeType, asn1.QualifiedNodeType:
		err = el.handleNodeContext(context, t)
		if err != nil {
			return nil, errs.Wrap(err, "failed to decode node context")
		}
	case asn1.FunctionType:
		err = el.handleFunctionContext(context, t)
		if err != nil {
			return nil, errs.Wrap(err, "failed to decode function context")
		}
	}

	return []*asn1.Decoder{decoder, context}, nil
}

func (el *Element) handlePath(decoder *asn1.Decoder) ([]*asn1.Decoder, error) {
	contextDec, _, err := decoder.Read(asn1.ContextZeroTag, asn1.ContextByte)
	if err != nil {
		return nil, errs.Wrap(err, "failed to context")
	}

	path, err := getPath(contextDec)
	if err != nil {
		return nil, errs.Wrap(err, "failed to get path")
	}

	el.Path = path

	atEnd, err := contextDec.ReadEnd()
	if err != nil {
		return nil, errs.Wrap(err, "failed to read end bytes")
	}

	if !atEnd {
		return nil, errs.New("not at sequence")
	}

	return []*asn1.Decoder{decoder, contextDec}, nil
}

func getPath(decoder *asn1.Decoder) (string, error) {
	tag, err := decoder.Peek()
	if err != nil {
		return "", errs.Wrap(err, "failed to peek path byte")
	}

	if tag == asn1.UniversalObjectTag {
		var path string

		path, err = handlePathFromUniversal(decoder)
		if err != nil {
			return "", errs.Wrap(err, "failed to read path from universal")
		}

		return path, nil
	}

	p, err := decoder.DecodeInteger()
	if err != nil {
		return "", errs.Wrap(err, "failed to handle path context")
	}

	return strconv.Itoa(p), nil
}

// getElement reads next full element from the decoder and returns leftover decoder.
func getElement(d *asn1.Decoder) (*Element, *asn1.Decoder, error) {
	t, err := d.Peek()
	if err != nil {
		return nil, nil, errs.Wrapf(err, "failed to read context")
	}

	el := &Element{}

	decoder, _, err := d.Read(t, asn1.ApplicationByte)
	if err != nil {
		return nil, nil, errs.Wrapf(err, "failed to read element application")
	}

	switch asn1.ApplicationByte(t) {
	case asn1.ApplicationByte(asn1.QualifiedNodeTag):
		el.ElementType = asn1.QualifiedNodeType
	case asn1.ApplicationByte(asn1.QualifiedParameterTag):
		el.ElementType = asn1.QualifiedParameterType
	case asn1.ApplicationByte(nodeTag):
		el.ElementType = asn1.NodeType
	case asn1.ApplicationByte(parameterTag):
		el.ElementType = asn1.ParameterType
	case asn1.ApplicationByte(functionTag):
		el.ElementType = asn1.FunctionType
	default:
		return nil, nil, errs.Errorf("unknown type: %x", t)
	}

	decoder, err = el.handleApplication(decoder)
	if err != nil {
		return el, nil, errs.Wrapf(err, "failed to handle application with type %x", asn1.ApplicationByte(t))
	}

	return el, decoder, nil
}

//nolint:gocyclo,cyclop
func (el *Element) handleFunctionContext(context *asn1.Decoder, tag byte) error {
	var (
		n   int
		err error
	)

	switch asn1.ContextByte(tag) {
	case asn1.ContextByte(0):
		var id string

		n, err = asn1.DecodeAny(context.Bytes(), &id)
		if err != nil {
			return errs.Wrap(err, "failed to decode identifier")
		}

		el.Identifier = id
	case asn1.ContextByte(1):
		var desc string

		n, err = asn1.DecodeAny(context.Bytes(), &desc)
		if err != nil {
			return errs.Wrap(err, "failed to decode description")
		}

		el.Description = desc
	case asn1.ContextByte(2):
		context, err = readOverElement(context)
		if err != nil {
			return errs.Wrapf(err, "failed to skip element at %x", asn1.ContextByte(2))
		}
	case asn1.ContextByte(3):
		context, err = readOverElement(context)
		if err != nil {
			return errs.Wrapf(err, "failed to skip element at %x", asn1.ContextByte(3))
		}
	}

	for i := 0; i < n; i++ {
		_, err := context.ReadByte()
		if err != nil {
			return errs.Wrapf(err, "failed to read over used bytes")
		}
	}

	return nil
}

//nolint:gocyclo,cyclop
func (el *Element) handleNodeContext(context *asn1.Decoder, tag byte) error {
	var (
		n   int
		err error
	)

	switch asn1.ContextByte(tag) {
	case asn1.ContextByte(0):
		var id string

		n, err = asn1.DecodeAny(context.Bytes(), &id)
		if err != nil {
			return errs.Wrap(err, "failed to decode identifier")
		}

		el.Identifier = id
	case asn1.ContextByte(1):
		var desc string

		n, err = asn1.DecodeAny(context.Bytes(), &desc)
		if err != nil {
			return errs.Wrap(err, "failed to decode description")
		}

		el.Description = desc
	case asn1.ContextByte(2):
		var root bool

		n, err = asn1.DecodeAny(context.Bytes(), &root)
		if err != nil {
			return errs.Wrap(err, "failed to decode is root ")
		}

		el.IsOnline = root
	case asn1.ContextByte(3):
		var online bool

		n, err = asn1.DecodeAny(context.Bytes(), &online)
		if err != nil {
			return errs.Wrap(err, "failed to decode is online ")
		}

		el.IsOnline = online
	case asn1.ContextByte(4):
		context, err = readOverElement(context)
		if err != nil {
			return errs.Wrapf(err, "failed to skip element at %x", asn1.ContextByte(4))
		}
	case asn1.ContextByte(5):
		context, err = readOverElement(context)
		if err != nil {
			return errs.Wrapf(err, "failed to skip element at %x", asn1.ContextByte(5))
		}
	}

	for i := 0; i < n; i++ {
		_, err := context.ReadByte()
		if err != nil {
			return errs.Wrapf(err, "failed to read over used bytes")
		}
	}

	return nil
}

// handlePropertyContext decodes context property tag.
//
//nolint:gocognit,gocyclo,cyclop
func (el *Element) handleParameterContext(context *asn1.Decoder, tag byte) error {
	var (
		n   int
		err error
	)

	switch asn1.ContextByte(tag) {
	case asn1.ContextByte(0):
		var id string

		n, err = asn1.DecodeAny(context.Bytes(), &id)
		if err != nil {
			return errs.Wrap(err, "failed to decode identifier")
		}

		el.Identifier = id
	case asn1.ContextByte(1):
		var desc string

		n, err = asn1.DecodeAny(context.Bytes(), &desc)
		if err != nil {
			return errs.Wrap(err, "failed to decode description")
		}

		el.Description = desc
	case asn1.ContextByte(2):
		var value any

		n, err = asn1.DecodeAny(context.Bytes(), &value)
		if err != nil {
			return errs.Wrap(err, "failed to decode parameter value")
		}

		el.Value = value
	case asn1.ContextByte(3):
		var min any

		n, err = asn1.DecodeAny(context.Bytes(), &min)
		if err != nil {
			return errs.Wrap(err, "failed to decode is min")
		}

		el.Minimum = min
	case asn1.ContextByte(4):
		var max any

		n, err = asn1.DecodeAny(context.Bytes(), &max)
		if err != nil {
			return errs.Wrap(err, "failed to decode is max")
		}

		el.Maximum = max
	case asn1.ContextByte(5):
		var access int

		access, err = context.DecodeInteger()
		if err != nil {
			return errs.Wrap(err, "failed to decode is access")
		}

		el.Access = access
	case asn1.ContextByte(6):
		var format string

		n, err = asn1.DecodeAny(context.Bytes(), &format)
		if err != nil {
			return errs.Wrap(err, "failed to decode is format")
		}

		el.Format = format
	case asn1.ContextByte(7):
		var enum string

		n, err = asn1.DecodeAny(context.Bytes(), &enum)
		if err != nil {
			return errs.Wrap(err, "failed to decode enumeration")
		}

		el.Enumeration = enum
	case asn1.ContextByte(8):
		var factor int

		factor, err = context.DecodeInteger()
		if err != nil {
			return errs.Wrap(err, "failed to decode is factor")
		}

		el.Factor = factor
	case asn1.ContextByte(9):
		var online bool

		n, err = asn1.DecodeAny(context.Bytes(), &online)
		if err != nil {
			return errs.Wrap(err, "failed to decode is online")
		}

		el.IsOnline = online
	case asn1.ContextByte(10):
		context, err = readOverElement(context)
		if err != nil {
			return errs.Wrapf(err, "failed to skip element at %x", asn1.ContextByte(10))
		}
	case asn1.ContextByte(11):
		context, err = readOverElement(context)
		if err != nil {
			return errs.Wrapf(err, "failed to skip element at %x", asn1.ContextByte(11))
		}
	case asn1.ContextByte(12):
		var def any

		n, err = asn1.DecodeAny(context.Bytes(), &def)
		if err != nil {
			return errs.Wrap(err, "failed to decode default value")
		}

		el.Default = def
	case asn1.ContextByte(13):
		var valType int

		valType, err = context.DecodeInteger()
		if err != nil {
			return errs.Wrap(err, "failed to decode default value")
		}

		el.ValueType = valType

	case asn1.ContextByte(14):
		context, err = readOverElement(context)
		if err != nil {
			return errs.Wrapf(err, "failed to skip element at %x", asn1.ContextByte(14))
		}
	case asn1.ContextByte(15):
		context, err = readOverElement(context)
		if err != nil {
			return errs.Wrapf(err, "failed to skip element at %x", asn1.ContextByte(15))
		}
	case asn1.ContextByte(16):
		context, err = readOverElement(context)
		if err != nil {
			return errs.Wrapf(err, "failed to skip element at %x", asn1.ContextByte(16))
		}
	case asn1.ContextByte(17):
		context, err = readOverElement(context)
		if err != nil {
			return errs.Wrapf(err, "failed to skip element at %x", asn1.ContextByte(17))
		}
	case asn1.ContextByte(18):
		context, err = readOverElement(context)
		if err != nil {
			return errs.Wrapf(err, "failed to skip element at %x", asn1.ContextByte(18))
		}
	}

	el.setDefaultElementValue()

	for i := 0; i < n; i++ {
		_, err := context.ReadByte()
		if err != nil {
			return errs.Wrapf(err, "failed to read over used bytes")
		}
	}

	return nil
}

func (el *Element) setDefaultElementValue() {
	if el.Value == nil {
		switch el.ValueType {
		case 1, 2:
			el.Value = 0
		case 3:
			el.Value = ""
		case 4:
			el.Value = false
		}
	}
}

func decoderWrapper(decoder *asn1.Decoder, handler decoderHandlerFunc) (*asn1.Decoder, error) {
	decoders, err := handler(decoder)
	if err != nil {
		return nil, errs.Wrap(err, "failed to execute handler")
	}

	var out *asn1.Decoder
	for _, d := range decoders {
		if out != nil && d.Len() > 0 {
			return nil, errs.New("after value handling both new and original decoders have data left")
		}

		if d.Len() > 0 {
			out = d

			continue
		}
	}

	if out != nil {
		return out, nil
	}

	return asn1.NewDecoder([]byte{}), nil
}

func handlePathFromUniversal(dec *asn1.Decoder) (string, error) {
	path, err := dec.DecodeUniversal()
	if err != nil {
		return "", errs.Wrapf(err, "failed to decode integer")
	}

	strPath := make([]string, 0, len(path))

	for _, p := range path {
		strPath = append(strPath, strconv.Itoa(p))
	}

	return strings.Join(strPath, "."), nil
}

// readOverElement skips next element in decoder.
func readOverElement(decoder *asn1.Decoder) (*asn1.Decoder, error) {
	tag, err := decoder.Peek()
	if err != nil {
		return nil, errs.Wrapf(err, "failed to peek next element tag")
	}

	newDec, allDataInNew, err := decoder.Read(tag, asn1.UniversalByte)
	if err != nil {
		return nil, errs.Wrapf(err, "failed to read next element")
	}

	if !allDataInNew {
		return decoder, nil
	}

	for {
		_, err := newDec.ReadByte()
		if err != nil {
			return nil, errs.Wrapf(err, "failed to read next element bytes")
		}

		end, err := newDec.ReadEnd()
		if err != nil {
			return nil, errs.Wrapf(err, "failed to read next element end")
		}

		if end {
			return newDec, nil
		}
	}
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
