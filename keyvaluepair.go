package fidias

import (
	"encoding/hex"
	"encoding/json"

	"github.com/hexablock/hexalog"
)

// MarshalJSON is a custom marshaller to handle the entry key
func (kvp KeyValuePair) MarshalJSON() ([]byte, error) {
	obj := struct {
		Key          string
		Value        []byte
		Flags        int64
		Modification string
		Entry        *hexalog.Entry
	}{
		Key:          string(kvp.Key),
		Value:        kvp.Value,
		Flags:        kvp.Flags,
		Modification: hex.EncodeToString(kvp.Modification),
		Entry:        kvp.Entry,
	}
	return json.Marshal(obj)
}
