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
	"encoding/json"

	"git.zabbix.com/ap/ember-plus/ember/asn1"
	"git.zabbix.com/ap/ember-plus/ember/s101"
	"git.zabbix.com/ap/plugin-support/errs"
)

// ElementKey used for element identification based on either element id or path.
type ElementKey struct {
	ID   string
	Path string
}

// ElementCollection contains one level of elements and their Ids as key.
type ElementCollection map[ElementKey]*Element

// Populate fills in collection with data from the decoder.
//
//nolint:gocyclo,cyclop
func (ec ElementCollection) Populate(data *asn1.Decoder) error {
	var end bool

	app0Codec, _, err := data.Read(asn1.RootElementCollectionTag, asn1.ApplicationByte)
	if err != nil {
		return errs.Wrap(err, "failed to read element root collection tag")
	}

	app11Codec, _, err := app0Codec.Read(asn1.RootElementTag, asn1.ApplicationByte)
	if err != nil {
		return errs.Wrap(err, "failed to read element tag")
	}

	for {
		var context0 *asn1.Decoder

		context0, _, err = app11Codec.Read(asn1.ContextZeroTag, asn1.ContextByte)
		if err != nil {
			return errs.Wrap(err, "failed to read top level context 0")
		}

		var (
			decoder *asn1.Decoder
			el      *Element
		)

		el, decoder, err = getElement(context0)
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

		end, err = app11Codec.ReadEnd() // all  element end
		if err != nil {
			return errs.Wrap(err, "failed to decode element sequence end")
		}

		if end {
			break
		}
	}

	end, err = app0Codec.ReadEnd() // end of the whole element
	if err != nil {
		return errs.Wrap(err, "failed to read sequence end of application 0 (the whole payload)")
	}

	if !end {
		return errs.Wrap(err, "main application decoder still has data remaining")
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

	return nil, ErrElementNotFound
}

// GetElementByID returns element from collection with the provided identifier.
func (ec ElementCollection) GetElementByID(id string) (*Element, error) {
	for key, value := range ec {
		if key.ID == id {
			return value, nil
		}
	}

	return nil, ErrElementNotFound
}

// MarshalJSON returns the collection with path(string) in key value instead of a structure for json marshaling.
func (ec ElementCollection) MarshalJSON() ([]byte, error) {
	out := make(map[string]any)

	for k, v := range ec {
		switch v.ElementType {
		case asn1.NodeType, asn1.QualifiedNodeType:
			out[k.Path] = Node{
				Path:        v.Path,
				ElementType: v.ElementType,
				Identifier:  v.Identifier,
				Description: v.Description,
				Children:    v.Children,
				IsOnline:    v.IsOnline,
				IsRoot:      v.IsRoot,
			}
		case asn1.ParameterType, asn1.QualifiedParameterType:
			out[k.Path] = Parameter{
				Path:        v.Path,
				ElementType: v.ElementType,
				Children:    v.Children,
				Identifier:  v.Identifier,
				Description: v.Description,
				Value:       v.Value,
				Minimum:     v.Minimum,
				Maximum:     v.Maximum,
				Access:      v.Access,
				Format:      v.Format,
				Enumeration: v.Enumeration,
				Factor:      v.Factor,
				IsOnline:    v.IsOnline,
				Default:     v.Default,
				ValueType:   v.ValueType,
			}
		case asn1.FunctionType:
			out[k.Path] = Function{
				Path:        v.Path,
				ElementType: v.ElementType,
				Identifier:  v.Identifier,
				Description: v.Description,
			}
		default:
			return nil, errs.New("failed unknown element type")
		}
	}

	bytes, err := json.Marshal(out)
	if err != nil {
		return nil, errs.Wrap(err, "failed native marshal")
	}

	return bytes, nil
}

// NewElementConnection creates a empty element collection.
func NewElementConnection() ElementCollection {
	return make(ElementCollection)
}

// GetRootRequest returns a S101 request packet with an encoded request for root collection.
func GetRootRequest() ([]byte, error) {
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
