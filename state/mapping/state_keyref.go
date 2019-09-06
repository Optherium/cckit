package mapping

import (
	"strings"

	"github.com/optherium/cckit/state"
	"github.com/optherium/cckit/state/schema"
)

// KeyRefNamespace namespace for uniq indexes
const KeyRefNamespace = `_idx`

// KeyRefIDKeyer keyer for KeyRef entity
var KeyRefIDKeyer = attrsPKeyer([]string{`Schema`, `Idx`, `RefKey`})

var KeyRefMapper = &StateMapping{
	schema:       &schema.KeyRef{},
	namespace:    state.Key{KeyRefNamespace},
	primaryKeyer: KeyRefIDKeyer,
}

var KeyRefIDMapper = &StateMapping{
	schema:       &schema.KeyRefId{},
	namespace:    state.Key{KeyRefNamespace},
	primaryKeyer: KeyRefIDKeyer,
}

func NewKeyRef(target interface{}, idx string, refKey, pKey state.Key) *schema.KeyRef {
	return &schema.KeyRef{
		Schema: strings.Join(SchemaNamespace(target), `-`),
		Idx:    idx,
		RefKey: []string(refKey),
		PKey:   []string(pKey),
	}
}

func NewKeyRefID(target interface{}, idx string, refKey state.Key) *schema.KeyRefId {
	return &schema.KeyRefId{
		Schema: strings.Join(SchemaNamespace(target), `-`),
		Idx:    idx,
		RefKey: []string(refKey),
	}
}

func NewKeyRefMapped(target interface{}, idx string, refKey, pKey state.Key) *ProtoStateMapped {
	return NewProtoStateMapped(NewKeyRef(target, idx, refKey, pKey), KeyRefMapper)
}

func NewKeyRefIDMapped(target interface{}, idx string, refKey state.Key) *ProtoStateMapped {
	return NewProtoStateMapped(
		NewKeyRefID(target, idx, refKey),
		KeyRefIDMapper)
}
