package sqlWrapper

import (
	"crypto/sha256"
	"database/sql"
)

func hash(result []sql.RawBytes) [32]byte {
	hasher := sha256.New()
	for _, bytes := range result {
		if bytes == nil {
			hasher.Write([]byte{0xDE, 0xAD, 0xBE, 0xEF})
		} else {
			hasher.Write(bytes)
		}
		hasher.Write([]byte{0})
	}

	var hash [32]byte
	copy(hash[:], hasher.Sum(nil)[:])

	return hash
}
