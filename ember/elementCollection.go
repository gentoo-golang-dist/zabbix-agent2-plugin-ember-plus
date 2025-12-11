/*
** Copyright (C) 2001-2025 Zabbix SIA
**
** This program is free software: you can redistribute it and/or modify it under the terms of
** the GNU Affero General Public License as published by the Free Software Foundation, version 3.
**
** This program is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY;
** without even the implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.
** See the GNU Affero General Public License for more details.
**
** You should have received a copy of the GNU Affero General Public License along with this program.
** If not, see <https://www.gnu.org/licenses/>.
**/

package ember

import (
	"encoding/json"
	"fmt"

	"golang.zabbix.com/plugin/ember-plus/ember/asn1"
	"golang.zabbix.com/sdk/errs"
)

// ElementKey used for element identification based on either element id or path.
type ElementKey struct {
	ID   string
	Path string
}

// ElementCollection contains one level of elements and their Ids as key.
type ElementCollection map[ElementKey]*Element

// NewElementCollection creates a empty element collection.
func NewElementCollection() ElementCollection {
	return make(ElementCollection)
}

// GetElementByPath returns element from collection with the provided path OID.
func (ec ElementCollection) GetElementByPath(currentPath string) (*Element, error) {
	for key, el := range ec {
		if key.Path == currentPath {
			return el, nil
		}

		for _, ch := range el.Children {
			childPath := fmt.Sprintf("%s.%s", key.Path, ch.Path)
			if childPath == currentPath {
				return ch, nil
			}
		}
	}

	return nil, errs.Wrapf(ErrElementNotFound, "failed to find element with path %q", currentPath)
}

// GetElementByID returns element from collection with the provided identifier.
func (ec ElementCollection) GetElementByID(id string) (*Element, string, error) {
	for key, el := range ec {
		if key.ID == id {
			return el, key.Path, nil
		}

		for _, ch := range el.Children {
			if ch.Identifier == id {
				return ch, fmt.Sprintf("%s.%s", key.Path, ch.Path), nil
			}
		}
	}

	return nil, "", ErrElementNotFound
}

// MarshalJSON returns the collection with path(string) in key value instead of a structure for json marshaling.
func (ec ElementCollection) MarshalJSON() ([]byte, error) {
	out := make(map[string]any)

	for k, v := range ec {
		switch v.ElementType {
		case asn1.NodeType:
			out[k.Path] = node{
				Path:              v.Path,
				ElementType:       v.ElementType,
				Identifier:        v.Identifier,
				Description:       v.Description,
				IsOnline:          v.IsOnline,
				IsRoot:            v.IsRoot,
				SchemaIdentifiers: v.SchemaIdentifiers,
			}
		case asn1.ParameterType:
			out[k.Path] = parameter{
				Path:        v.Path,
				ElementType: v.ElementType,
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
			out[k.Path] = function{
				Path:        v.Path,
				ElementType: v.ElementType,
				Identifier:  v.Identifier,
				Description: v.Description,
			}
		case asn1.MatrixType:
			out[k.Path] = matrix{
				Path:        v.Path,
				ElementType: v.ElementType,
				Identifier:  v.Identifier,
				Description: v.Description,
			}
		case asn1.CommandType:
			out[k.Path] = command{
				Path:        v.Path,
				ElementType: v.ElementType,
				Number:      v.Number,
			}
		case asn1.StreamType:
			out[k.Path] = stream{
				Path:        v.Path,
				ElementType: v.ElementType,
				Value:       v.StreamValue,
				Identifier:  v.StreamIdentifier,
				ValueType:   v.ValueType,
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
