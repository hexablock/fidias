package fidias

import (
	"encoding/hex"
	"encoding/json"
)

// MarshalJSON is a custom marshaller to handle the entry key
func (kvp KeyValuePair) MarshalJSON() ([]byte, error) {
	obj := struct {
		Key          string
		Value        []byte
		Flags        int64
		Modification string
	}{
		Key:          string(kvp.Key),
		Value:        kvp.Value,
		Flags:        kvp.Flags,
		Modification: hex.EncodeToString(kvp.Modification),
	}
	return json.Marshal(obj)
}
