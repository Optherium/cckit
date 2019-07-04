package state

import (
	"errors"
	qb "github.com/ypeckstadt/hyperledger-fabric-couchdb-query-builder-golang"
	"strings"
)

func AddSortToQuery(query *qb.QueryBuilder, sortRequest []string) error {
	for _, sort := range sortRequest {
		split := strings.Split(sort, ":")
		if len(split) != 2 || (split[1] != "asc" && split[1] != "desc") {
			return errors.New("errInvalidSortRequest")
		}

		query = query.AddSort(split[0], split[1])
	}

	return nil
}
