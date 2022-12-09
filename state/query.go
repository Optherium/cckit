package state

import (
	"encoding/json"
	"fmt"
	. "github.com/optherium/cckit/errors"
	"reflect"
	"strings"
)

// QueryBuilder used to generate couchDB queries
type QueryBuilder struct {
	Ids          []string
	Fields       []string
	DocType      string
	Filters      map[string]interface{}
	Conditions   map[string]interface{}
	Combinations []*Combination
	Sort         []map[string]string
	HasSelector  bool
	HasLimit     bool
	HasSkip      bool
	HasIndex     bool
	Limit        int
	Skip         int
	Index        string
}

// Filter used to filter on a single field
type Filter struct {
	Field string
	Value interface{}
}

// Combination used for and,or,nor,all operators
type Combination struct {
	Type         CombinationType
	Value        []interface{}
	Filters      []Filter
	Conditions   []interface{}
	Combinations []*Combination
	Builder      *QueryBuilder
}

// CombinationType type of combination
type CombinationType string

const (
	// Matches if all the selectors in the array match.
	AND CombinationType = "$and"
	// Matches if any of the selectors in the array match. All selectors must use the same index.
	OR CombinationType = "$or"
	// Matches an array value if it contains all the elements of the argument array.
	ALL CombinationType = "$all"
	// Matches if none of the selectors in the array match.
	NOR CombinationType = "$nor"
)

// NewQB create a new instance of the QueryBuilder
func NewQB() *QueryBuilder {
	return &QueryBuilder{
		Filters:    make(map[string]interface{}),
		Conditions: make(map[string]interface{}),
	}
}

// Build constructs the query and outputs the final result
func (builder *QueryBuilder) Build() (string, error) {

	if !builder.HasSelector {
		return "", NoQuerySelectorError
	}

	// Initial declaration
	queryMap := map[string]interface{}{}

	// add fields
	if len(builder.Fields) > 0 {
		queryMap["fields"] = builder.Fields
	}

	// add selector
	if builder.HasSelector {
		queryMap["selector"] = map[string]interface{}{}

		selector := queryMap["selector"]
		selectorMap, ok := selector.(map[string]interface{})
		if !ok {
			return "", UnexpectedError
		}

		// add doc type
		if builder.DocType != "" {
			selectorMap["docType"] = builder.DocType
		}

		// add filters
		if len(builder.Filters) > 0 {
			for k, v := range builder.Filters {
				selectorMap[k] = v
			}
		}

		// add conditions
		if len(builder.Conditions) > 0 {
			for k, v := range builder.Conditions {
				selectorMap[k] = v
			}
		}

		// combinations
		if len(builder.Combinations) > 0 {
			for _, combination := range builder.Combinations {
				addCombinationToRoot(selectorMap, combination)
			}
		}
	}

	// add sort
	if len(builder.Sort) > 0 {
		queryMap["sort"] = builder.Sort
	}

	// add limit
	if builder.HasLimit {
		queryMap["limit"] = builder.Limit
	}

	// add skip
	if builder.HasSkip {
		queryMap["skip"] = builder.Skip
	}

	if builder.HasIndex {
		queryMap["use_index"] = builder.Index
	}

	bytes, err := json.Marshal(&queryMap)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// AddField adds a field to the couchDB query
func (builder *QueryBuilder) AddField(fields ...string) *QueryBuilder {
	builder.Fields = append(builder.Fields, fields...)
	return builder
}

func (builder *QueryBuilder) AddUseIndex(index string) *QueryBuilder {
	builder.Index = index
	builder.HasIndex = true

	return builder
}

// AddFilter adds a filter to filter on in the couchDB query
func (builder *QueryBuilder) AddFilter(field string, value interface{}) *QueryBuilder {
	builder.Filters[field] = value
	builder.HasSelector = true
	return builder
}

// SetDocType set the main doc type for the couchDB query
func (builder *QueryBuilder) SetDocType(docType string) *QueryBuilder {
	builder.DocType = docType
	builder.HasSelector = true
	return builder
}

// SetLimit sets the limit for paging
func (builder *QueryBuilder) SetLimit(limit int) *QueryBuilder {
	builder.Limit = limit
	builder.HasLimit = true
	return builder
}

// SetSkip sets the skip value for paging
func (builder *QueryBuilder) SetSkip(skip int) *QueryBuilder {
	builder.Skip = skip
	builder.HasSkip = true
	return builder
}

// AddCondition adds a pre-defined CouchDB condition filter to the CouchDB query
func (builder *QueryBuilder) AddCondition(field string, condition interface{}) *QueryBuilder {
	builder.Conditions[field] = condition
	builder.HasSelector = true
	return builder
}

// AddSort adds a field to sort on in the couchDB query
func (builder *QueryBuilder) AddSort(field string, sortOrder string) *QueryBuilder {
	sort := map[string]string{}
	sort[field] = strings.ToLower(sortOrder)
	builder.Sort = append(builder.Sort, sort)
	return builder
}

func (builder *QueryBuilder) AddManySorts(sortRequest []string) error {
	for _, sort := range sortRequest {
		split := strings.Split(sort, ":")
		if len(split) != 2 || (split[1] != "asc" && split[1] != "desc") {
			return InvalidSortQueryError
		}

		builder.AddSort(split[0], split[1])
	}

	return nil
}

// AddCombination adds a combination to the builder query
func (builder *QueryBuilder) AddCombination(combinationType CombinationType, filters ...interface{}) *Combination {
	combination := Combination{Type: combinationType, Builder: builder}

	for _, filter := range filters {
		typeName := reflect.TypeOf(filter).Name()
		if typeName == "Filter" {
			original, ok := filter.(Filter)
			if ok {
				combination.Filters = append(combination.Filters, original)
			}
		} else {
			combination.Conditions = append(combination.Conditions, filter)
		}
	}

	builder.Combinations = append(builder.Combinations, &combination)
	return &combination
}

// AddCombination adds a combination to an existing one for nesting
func (c *Combination) AddCombination(combinationType CombinationType, filters ...interface{}) *Combination {
	combination := Combination{Type: combinationType}

	// loop through filters and check type, if filter then create custom, if condition add as condition
	for _, filter := range filters {
		typeName := reflect.TypeOf(filter).Name()
		if typeName == "Filter" {
			original, ok := filter.(Filter)
			if ok {
				combination.Filters = append(combination.Filters, original)
			}
		} else {
			combination.Conditions = append(combination.Conditions, filter)
		}
	}

	c.Combinations = append(c.Combinations, &combination)

	return &combination
}

// addCombinationToRoot converts the combination to a format that is useable by CouchDB
func addCombinationToRoot(root map[string]interface{}, combination *Combination) {

	combinationType := string(combination.Type)
	combinationRoot := map[string][]interface{}{}

	//loop through filters
	for _, filter := range combination.Filters {
		filterMap := map[string]interface{}{}
		filterMap[filter.Field] = filter.Value
		combinationRoot[combinationType] = append(combinationRoot[combinationType], filterMap)
	}

	// loop through combinations
	if len(combination.Combinations) > 0 {
		fmt.Println("has child combinations")
	}
	for _, child := range combination.Combinations {
		combinationRoot[combinationType] = addCombinationToParent(combinationRoot[combinationType], child)
	}

	root[combinationType] = combinationRoot[combinationType]
}

func addCombinationToParent(parent []interface{}, combination *Combination) []interface{} {
	combinationType := string(combination.Type)
	combinationRoot := map[string][]interface{}{}

	//loop through filters
	for _, filter := range combination.Filters {
		filterMap := map[string]interface{}{}
		filterMap[filter.Field] = filter.Value
		combinationRoot[combinationType] = append(combinationRoot[combinationType], filterMap)
	}

	// loop through combinations
	for _, child := range combination.Combinations {
		combinationRoot[combinationType] = addCombinationToParent(combinationRoot[combinationType], child)
	}
	parent = append(parent, combinationRoot)
	return parent
}
