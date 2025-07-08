package sqlWrapper

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
)

func Exec(destination [16]byte, parsedQueries []ParsedQuery) [32]byte {
	db, err := sql.Open("sqlite3", fmt.Sprintf("%x.db", destination))
	if err != nil {
		fmt.Printf("(%x)>> Error: %s\n", destination, err)
	}
	defer db.Close()

	hasher := sha256.New()

	for _, parsedQuery := range parsedQueries {
		fmt.Printf("(%x)>> Query: %s\n", destination, parsedQuery.Query)
		if parsedQuery.Returning {
			rows, err := db.Query(parsedQuery.Query)
			if err != nil {
				fmt.Printf("(%x)>> Error: %s\n", destination, err)
				continue
			}
			defer rows.Close()

			cols, err := rows.Columns()
			if err != nil {
				fmt.Printf("(%x)>> Error: %s\n", destination, err)
				continue
			}

			result := make([]sql.RawBytes, len(cols))
			resultAddr := make([]interface{}, len(cols))
			for i := range result {
				resultAddr[i] = &result[i]
			}

			for rows.Next() {
				err := rows.Scan(resultAddr...)
				if err != nil {
					fmt.Printf("(%x)>> Error: %s\n", destination, err)
					continue
				}
			}

			hashPerQuery := hash(result)
			hasher.Write(hashPerQuery[:])
		} else {
			_, err := db.Exec(parsedQuery.Query)
			if err != nil {
				fmt.Printf("(%x)>> Error: %s\n", destination, err)
			}
		}
	}

	var hash [32]byte
	copy(hash[:], hasher.Sum(nil)[:])

	return hash
}
