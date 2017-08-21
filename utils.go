package fidias

import "bytes"

// // Checks if a key is STRICTLY between two ID's exclusively
// func between(id1, id2, key []byte) bool {
// 	// Check for ring wrap around
// 	if bytes.Compare(id1, id2) == 1 {
// 		return bytes.Compare(id1, key) == -1 ||
// 			bytes.Compare(id2, key) == 1
// 	}
//
// 	// Handle the normal case
// 	return bytes.Compare(id1, key) == -1 &&
// 		bytes.Compare(id2, key) == 1
// }
//
// // Checks if a key is between two ID's, right inclusive
// func betweenRightIncl(id1, id2, key []byte) bool {
// 	// Check for ring wrap around
// 	if bytes.Compare(id1, id2) == 1 {
// 		return bytes.Compare(id1, key) == -1 ||
// 			bytes.Compare(id2, key) >= 0
// 	}
//
// 	return bytes.Compare(id1, key) == -1 &&
// 		bytes.Compare(id2, key) >= 0
// }

func betweenLeftIncl(id1, id2, key []byte) bool {
	// Check for ring wrap around
	if bytes.Compare(id1, id2) == 1 {
		return bytes.Compare(id1, key) <= 0 ||
			bytes.Compare(id2, key) == 1
	}

	return bytes.Compare(id1, key) <= 0 &&
		bytes.Compare(id2, key) == 1
}

// get the location id of the given vnode id/hash from the location set
func getVnodeLocID(hash []byte, locs [][]byte) []byte {
	l := len(locs)
	for i := range locs[:l-1] {
		if betweenLeftIncl(locs[i], locs[i+1], hash) {
			return locs[i]
		}
	}

	return locs[l-1]
}
