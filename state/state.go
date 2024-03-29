package state

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/protos/ledger/queryresult"
	"github.com/optherium/cckit/convert"
	. "github.com/optherium/cckit/errors"
	"github.com/pkg/errors"
)

// HistoryEntry struct containing history information of a single entry
type HistoryEntry struct {
	TxId      string      `json:"txId"`
	Timestamp int64       `json:"timestamp"`
	IsDeleted bool        `json:"isDeleted"`
	Value     interface{} `json:"value"`
}

// HistoryEntryList list of history entries
type HistoryEntryList []HistoryEntry

type (
	Key []string

	// StateKey stores origin and transformed state key
	TransformedKey struct {
		Origin Key
		Parts  Key
		String string
	}

	//KeyerFunc func(string) ([]string, error)
	KeyFunc func() (Key, error)

	// KeyerFunc transforms string to key
	KeyerFunc func(string) (Key, error)

	// Keyer interface for entity containing logic of its key creation
	Keyer interface {
		Key() (Key, error)
	}

	// StringsKeys interface for entity containing logic of its key creation - backward compatibility
	StringsKeyer interface {
		Key() ([]string, error)
	}

	// KeyValue interface combines Keyer as ToByter methods - state entry representation
	KeyValue interface {
		Keyer
		convert.ToByter
	}
)

// State interface for chain code CRUD operations
type State interface {
	// Get returns value from state, converted to target type
	// entry can be Key (string or []string) or type implementing Keyer interface
	Get(entry interface{}, target ...interface{}) (result interface{}, err error)

	// Get returns value from state, converted to int
	// entry can be Key (string or []string) or type implementing Keyer interface
	GetInt(entry interface{}, defaultValue int) (result int, err error)

	// GetHistory returns slice of history records for entry, with values converted to target type
	// entry can be Key (string or []string) or type implementing Keyer interface
	GetHistory(entry interface{}, target interface{}) (result HistoryEntryList, err error)

	// Exists returns entry existence in state
	// entry can be Key (string or []string) or type implementing Keyer interface
	Exists(entry interface{}) (exists bool, err error)

	// Put returns result of putting entry to state
	// entry can be Key (string or []string) or type implementing Keyer interface
	// if entry is implements Keyer interface and it's struct or type implementing
	// ToByter interface value can be omitted
	Put(entry interface{}, value ...interface{}) (err error)

	// Insert returns result of inserting entry to state
	// If same key exists in state error wil be returned
	// entry can be Key (string or []string) or type implementing Keyer interface
	// if entry is implements Keyer interface and it's struct or type implementing
	// ToByter interface value can be omitted
	Insert(entry interface{}, value ...interface{}) (err error)

	// List returns slice of target type
	// namespace can be part of key (string or []string) or entity with defined mapping
	List(namespace interface{}, target ...interface{}) (result interface{}, err error)

	// Delete returns result of deleting entry from state
	// entry can be Key (string or []string) or type implementing Keyer interface
	Delete(entry interface{}) (err error)

	Logger() *shim.ChaincodeLogger

	UseKeyTransformer(KeyTransformer) State
	UseStateGetTransformer(FromBytesTransformer) State
	UseStatePutTransformer(ToBytesTransformer) State

	// GetPrivate returns value from private state, converted to target type
	// entry can be Key (string or []string) or type implementing Keyer interface
	GetPrivate(collection string, entry interface{}, target ...interface{}) (result interface{}, err error)

	// PutPrivate returns result of putting entry to private state
	// entry can be Key (string or []string) or type implementing Keyer interface
	// if entry is implements Keyer interface and it's struct or type implementing
	// ToByter interface value can be omitted
	PutPrivate(collection string, entry interface{}, value ...interface{}) (err error)

	// InsertPrivate returns result of inserting entry to private state
	// If same key exists in state error wil be returned
	// entry can be Key (string or []string) or type implementing Keyer interface
	// if entry is implements Keyer interface and it's struct or type implementing
	// ToByter interface value can be omitted
	InsertPrivate(collection string, entry interface{}, value ...interface{}) (err error)

	// ListPrivate returns slice of target type from private state
	// namespace can be part of key (string or []string) or entity with defined mapping
	// If usePrivateDataIterator is true, used private state for iterate over objects
	// if false, used public state for iterate over keys and GetPrivateData for each key
	ListPrivate(collection string, usePrivateDataIterator bool, namespace interface{}, target ...interface{}) (result interface{}, err error)

	// DeletePrivate returns result of deleting entry from private state
	// entry can be Key (string or []string) or type implementing Keyer interface
	DeletePrivate(collection string, entry interface{}) (err error)

	// ExistsPrivate returns entry existence in private state
	// entry can be Key (string or []string) or type implementing Keyer interface
	ExistsPrivate(collection string, entry interface{}) (exists bool, err error)

	// PaginateList allows to list keys by prefix with pagination
	PaginateList(objectType interface{}, target interface{}, pageSize int32, start string) (result []interface{}, end string, err error)

	// RichListQuery allows to perform rich state DB query using state DB syntax with bookmark pagination
	RichListQuery(query string, target interface{}, pageSize int32, bookmark string) (result []interface{}, newBookmark string, err error)

	// RichQuery allows to perform rich state DB query using state DB syntax
	RichQuery(query string, target interface{}, pageSize int) ([]interface{}, int, error)
}

func (k Key) Append(key Key) Key {
	return append(k, key...)
}

type Impl struct {
	stub                shim.ChaincodeStubInterface
	logger              *shim.ChaincodeLogger
	StateKeyTransformer KeyTransformer
	StateGetTransformer FromBytesTransformer
	StatePutTransformer ToBytesTransformer
}

// NewState creates wrapper on shim.ChaincodeStubInterface for working with state
func NewState(stub shim.ChaincodeStubInterface, logger *shim.ChaincodeLogger) *Impl {
	return &Impl{
		stub:                stub,
		logger:              logger,
		StateKeyTransformer: KeyAsIs,
		StateGetTransformer: ConvertFromBytes,
		StatePutTransformer: ConvertToBytes,
	}
}

func (s *Impl) Logger() *shim.ChaincodeLogger {
	return s.logger
}

func KeyToString(stub shim.ChaincodeStubInterface, key Key) (string, error) {
	switch len(key) {
	case 0:
		return ``, ErrKeyPartsLength
	case 1:
		return key[0], nil
	default:
		return stub.CreateCompositeKey(key[0], key[1:])
	}
}

func (s *Impl) Key(key interface{}) (*TransformedKey, error) {
	var (
		trKey = &TransformedKey{}
		err   error
	)

	if trKey.Origin, err = NormalizeStateKey(key); err != nil {
		return nil, errors.Wrap(err, `key normalizing`)
	}

	s.logger.Debugf(`state KEY: %s`, trKey.Origin)

	if trKey.Parts, err = s.StateKeyTransformer(trKey.Origin); err != nil {
		return nil, err
	}

	if trKey.String, err = KeyToString(s.stub, trKey.Parts); err != nil {
		return nil, err
	}

	return trKey, nil
}

// Get data by key from state, trying to convert to target interface
func (s *Impl) Get(entry interface{}, config ...interface{}) (interface{}, error) {
	key, err := s.Key(entry)
	if err != nil {
		s.logger.Errorf("Unable to compose key at state.Get: %s", err)
		return nil, UnexpectedError
	}

	//bytes from state
	s.logger.Debugf(`state GET %s`, key.String)
	bb, err := s.stub.GetState(key.String)
	if err != nil {
		s.logger.Errorf("Unable to retrieve data from state by key %s: %s", key.String, err)
		return nil, SetGetError
	}
	if len(bb) == 0 {
		// config[1] default value
		if len(config) >= 2 {
			return config[1], nil
		}
		s.logger.Debugf("Key %s not found in state", key.String)
		return nil, ErrKeyNotFound
	}

	// config[0] - target type
	return s.StateGetTransformer(bb, config...)
}

func (s *Impl) GetInt(key interface{}, defaultValue int) (int, error) {
	val, err := s.Get(key, convert.TypeInt, defaultValue)
	if err != nil {
		return 0, SetGetError
	}
	return val.(int), nil
}

// GetHistory by key from state, trying to convert to target interface
func (s *Impl) GetHistory(entry interface{}, target interface{}) (HistoryEntryList, error) {
	key, err := s.Key(entry)
	if err != nil {
		s.logger.Errorf("Unable to compose key at state.GetHistory: %s", err)
		return nil, UnexpectedError
	}

	iter, err := s.stub.GetHistoryForKey(key.String)
	if err != nil {
		s.logger.Errorf("Unable to retrieve history from state by key %s: %s", key.String, err)
		return nil, SetGetError
	}

	defer func() { _ = iter.Close() }()

	results := HistoryEntryList{}

	for iter.HasNext() {
		state, err := iter.Next()
		if err != nil {
			s.logger.Errorf("Unable to get next item from iterator at state.GetHistory: %s", err)
			return nil, UnexpectedError
		}
		value, err := s.StateGetTransformer(state.Value, target)
		if err != nil {
			s.logger.Errorf("Unable to transform state entry at state.GetHistory: %s", err)
			return nil, UnexpectedError
		}

		entry := HistoryEntry{
			TxId:      state.GetTxId(),
			Timestamp: state.GetTimestamp().GetSeconds(),
			IsDeleted: state.GetIsDelete(),
			Value:     value,
		}
		results = append(results, entry)
	}

	return results, nil
}

// Exists check entry with key exists in chaincode state
func (s *Impl) Exists(entry interface{}) (bool, error) {
	key, err := s.Key(entry)
	if err != nil {
		s.logger.Errorf("Unable to compose key at state.Exists: %s", err)
		return false, UnexpectedError
	}
	s.logger.Debugf(`state check EXISTENCE %s`, key.String)
	bb, err := s.stub.GetState(key.String)
	if err != nil {
		s.logger.Errorf("Unable to check key %s existence: %s", key.String, err)
		return false, SetGetError
	}
	return len(bb) != 0, nil
}

// List data from state using objectType prefix in composite key, trying to convert to target interface.
// Keys -  additional components of composite key
func (s *Impl) List(namespace interface{}, target ...interface{}) (interface{}, error) {
	stateList := NewStateList(target...)
	key, err := NormalizeStateKey(namespace)
	if err != nil {
		s.logger.Errorf("Unable to normalize state key at state.List: %s", err)
		return nil, UnexpectedError
	}
	s.logger.Debugf(`state LIST namespace: %s`, key)

	key, err = s.StateKeyTransformer(key)
	if err != nil {
		s.logger.Errorf("Unable to construct state key transformer: %s", err)
		return nil, UnexpectedError
	}
	s.logger.Debugf(`state LIST with composite key: %s`, key)

	iter, err := s.stub.GetStateByPartialCompositeKey(key[0], key[1:])
	if err != nil {
		s.logger.Errorf("Unable to get state by partial composite key: %s", err)
		return nil, SetGetError
	}
	defer func() { _ = iter.Close() }()

	return stateList.Fill(iter, s.StateGetTransformer)
}

func NormalizeStateKey(key interface{}) (Key, error) {
	switch k := key.(type) {
	case Key:
		return k, nil
	case Keyer:
		return k.Key()
	case StringsKeyer:
		return k.Key()
	case string:
		return Key{k}, nil
	case []string:
		return k, nil
	}
	return nil, fmt.Errorf(`%s: %s`, ErrUnableToCreateStateKey, reflect.TypeOf(key))
}

func (s *Impl) argKeyValue(arg interface{}, values []interface{}) (key Key, value interface{}, err error) {
	// key must be
	if key, err = NormalizeStateKey(arg); err != nil {
		return
	}

	switch len(values) {
	// arg is key and  value
	case 0:
		return key, arg, nil
	case 1:
		return key, values[0], nil
	default:
		return nil, nil, ErrAllowOnlyOneValue
	}
}

// Put data value in state with key, trying convert data to []byte
func (s *Impl) Put(entry interface{}, values ...interface{}) error {
	entryKey, value, err := s.argKeyValue(entry, values)
	if err != nil {
		s.logger.Errorf("Unable to get entryKey at state.Put: %s", err)
		return UnexpectedError
	}
	bb, err := s.StatePutTransformer(value)
	if err != nil {
		s.logger.Errorf("Unable to construct state put transfomer: %s", err)
		return UnexpectedError
	}

	key, err := s.Key(entryKey)
	if err != nil {
		s.logger.Errorf("Unable to compose key at state.Put: %s", err)
		return UnexpectedError
	}

	s.logger.Debugf(`state PUT with string key: %s`, key.String)
	return s.stub.PutState(key.String, bb)
}

// Insert value into chaincode state, returns error if key already exists
func (s *Impl) Insert(entry interface{}, values ...interface{}) error {
	if exists, err := s.Exists(entry); err != nil {
		s.logger.Errorf("Unable to check key existence at state.Insert: %s", err)
		return SetGetError
	} else if exists {
		return ErrKeyAlreadyExists
	}

	key, value, err := s.argKeyValue(entry, values)
	if err != nil {
		s.logger.Errorf("Unable to get entryKey at state.Insert: %s", err)
		return UnexpectedError
	}
	return s.Put(key, value)
}

// Delete entry from state
func (s *Impl) Delete(entry interface{}) error {
	key, err := s.Key(entry)
	if err != nil {
		s.logger.Errorf("Unable to compose key at state.Delete: %s", err)
		return UnexpectedError
	}

	s.logger.Debugf(`state DELETE with string key: %s`, key.String)
	return s.stub.DelState(key.String)
}

func (s *Impl) UseKeyTransformer(kt KeyTransformer) State {
	s.StateKeyTransformer = kt
	return s
}
func (s *Impl) UseStateGetTransformer(fb FromBytesTransformer) State {
	s.StateGetTransformer = fb
	return s
}

func (s *Impl) UseStatePutTransformer(tb ToBytesTransformer) State {
	s.StatePutTransformer = tb
	return s
}

// KeyError error with key
func KeyError(key *TransformedKey) error {
	// show origin key
	return errors.New(strings.Join(key.Origin, ` | `))
}

type stringKeyer struct {
	str   string
	keyer KeyerFunc
}

func (sk stringKeyer) Key() (Key, error) {
	return sk.keyer(sk.str)
}

// StringKeyer constructor for struct implementing Keyer interface
func StringKeyer(str string, keyer KeyerFunc) Keyer {
	return stringKeyer{str, keyer}
}

// Get data by key from private state, trying to convert to target interface
func (s *Impl) GetPrivate(collection string, entry interface{}, config ...interface{}) (interface{}, error) {
	key, err := s.Key(entry)
	if err != nil {
		s.logger.Errorf("Unable to compose key at state.GetPrivate: %s", err)
		return nil, UnexpectedError
	}

	//bytes from private state
	s.logger.Debugf(`private state GET %s`, key.String)
	bb, err := s.stub.GetPrivateData(collection, key.String)
	if err != nil {
		s.logger.Errorf("Unable to get private data from state: %s", err)
		return nil, SetGetError
	}
	if len(bb) == 0 {
		// config[1] default value
		if len(config) >= 2 {
			return config[1], nil
		}
		s.logger.Debugf("Key %s not found in state", key.String)
		return nil, ErrKeyNotFound
	}

	// config[0] - target type
	return s.StateGetTransformer(bb, config...)
}

// PrivateExists check entry with key exists in chaincode private state
func (s *Impl) ExistsPrivate(collection string, entry interface{}) (bool, error) {
	key, err := s.Key(entry)
	if err != nil {
		s.logger.Errorf("Unable to compose key at state.ExistsPrivate: %s", err)
		return false, UnexpectedError
	}
	s.logger.Debugf(`private state check EXISTENCE %s`, key.String)
	bb, err := s.stub.GetPrivateData(collection, key.String)
	if err != nil {
		s.logger.Errorf("Unable to get private data for key %s: %s", key.String, err)
		return false, SetGetError
	}
	return len(bb) != 0, nil
}

// List data from private state using objectType prefix in composite key, trying to convert to target interface.
// Keys -  additional components of composite key
// If usePrivateDataIterator is true, used private state for iterate over objects
// if false, used public state for iterate over keys and GetPrivateData for each key
func (s *Impl) ListPrivate(collection string, usePrivateDataIterator bool, namespace interface{}, target ...interface{}) (interface{}, error) {
	stateList := NewStateList(target...)
	key, err := NormalizeStateKey(namespace)
	if err != nil {
		s.logger.Errorf("Unable to normalize state key at state.ListPrivate: %s", err)
		return nil, UnexpectedError
	}
	s.logger.Debugf(`state LIST namespace: %s`, key)

	if key, err = s.StateKeyTransformer(key); err != nil {
		s.logger.Errorf("Unable to construct state key transformer: %s", err)
		return nil, UnexpectedError
	}
	s.logger.Debugf(`state LIST with composite key: %s`, key)

	if usePrivateDataIterator {
		iter, err := s.stub.GetPrivateDataByPartialCompositeKey(collection, key[0], key[1:])
		if err != nil {
			return nil, errors.Wrap(err, `create list iterator`)
		}
		defer func() { _ = iter.Close() }()
		return stateList.Fill(iter, s.StateGetTransformer)
	}

	iter, err := s.stub.GetStateByPartialCompositeKey(key[0], key[1:])
	if err != nil {
		s.logger.Errorf("Unable to get state by partial composite key: %s", err)
		return nil, SetGetError
	}
	defer func() { _ = iter.Close() }()

	var (
		kv       *queryresult.KV
		objKey   string
		keyParts []string
	)
	for iter.HasNext() {
		if kv, err = iter.Next(); err != nil {
			s.logger.Errorf("Unable to get next item from iterator at state.ListPrivate: %s", err)
			return nil, UnexpectedError
		}
		if objKey, keyParts, err = s.stub.SplitCompositeKey(kv.Key); err != nil {
			s.logger.Errorf("Unable to split composite key: %s", err)
			return nil, UnexpectedError
		}

		object, err := s.GetPrivate(collection, append([]string{objKey}, keyParts...), target...)
		if err != nil {
			s.logger.Errorf("Unable to get private data from state at state.ListPrivate: %s", err)
			return nil, SetGetError
		}
		stateList.AddElementToList(object)
	}

	return stateList.Get()
}

// Put data value in private state with key, trying convert data to []byte
func (s *Impl) PutPrivate(collection string, entry interface{}, values ...interface{}) (err error) {
	entryKey, value, err := s.argKeyValue(entry, values)
	if err != nil {
		s.logger.Errorf("Unable to get entryKey at state.PutPrivate: %s", err)
		return UnexpectedError
	}
	bb, err := s.StatePutTransformer(value)
	if err != nil {
		s.logger.Errorf("Unable to construct state put transfomer: %s", err)
		return UnexpectedError
	}

	key, err := s.Key(entryKey)
	if err != nil {
		s.logger.Errorf("Unable to compose key at state.Put: %s", err)
		return UnexpectedError
	}

	s.logger.Debugf(`state PUT with string key: %s`, key.String)
	return s.stub.PutPrivateData(collection, key.String, bb)
}

// Insert value into chaincode private state, returns error if key already exists
func (s *Impl) InsertPrivate(collection string, entry interface{}, values ...interface{}) (err error) {
	if exists, err := s.ExistsPrivate(collection, entry); err != nil {
		s.logger.Errorf("Unable to check key existence at state.InsertPrivate: %s", err)
		return SetGetError
	} else if exists {
		return ErrKeyAlreadyExists
	}

	key, value, err := s.argKeyValue(entry, values)
	if err != nil {
		s.logger.Errorf("Unable to get entryKey at state.Insert: %s", err)
		return UnexpectedError
	}
	return s.PutPrivate(collection, key, value)
}

// Delete entry from private state
func (s *Impl) DeletePrivate(collection string, entry interface{}) error {
	key, err := s.Key(entry)
	if err != nil {
		s.logger.Errorf("Unable to compose key at state.DeletePrivate: %s", err)
		return UnexpectedError
	}
	s.logger.Debugf(`private state DELETE with string key: %s`, key.String)
	return s.stub.DelPrivateData(collection, key.String)
}

func (s *Impl) PaginateList(objectType interface{}, target interface{}, limit int32, start string) (result []interface{}, end string, err error) {
	key, err := s.Key(objectType)

	if err != nil {
		s.logger.Errorf("Unable to construct key at state.PaginateList: %s", err)
		return nil, "", UnexpectedError
	}

	iter, meta, err := s.stub.GetStateByPartialCompositeKeyWithPagination(key.Parts[0], key.Parts[1:], limit, start)

	if err != nil {
		s.logger.Errorf("Unable to get state iterator at state.PaginateList: %s", err)
		return nil, "", SetGetError
	}

	end = meta.Bookmark

	entries := make([]interface{}, 0)
	defer func() { _ = iter.Close() }()

	for iter.HasNext() {
		v, err := iter.Next()

		if err != nil {
			s.logger.Errorf("Unable to get next item from iterator at state.PaginateList: %s", err)
			return nil, "", UnexpectedError
		}

		entry, err := s.StateGetTransformer(v.Value, target)

		if err != nil {
			s.logger.Errorf("Unable to transform state entry at state.PaginateList: %s", err)
			return nil, "", UnexpectedError
		}

		entries = append(entries, entry)
	}
	return entries, end, nil
}

func (s *Impl) RichQuery(query string, target interface{}, pageSize int) ([]interface{}, int, error) {
	iter, err := s.stub.GetQueryResult(query)
	if err != nil {
		s.logger.Errorf("Unable to get query result at state.RichQuery: %s", err)
		return nil, 0, SetGetError
	}

	entries := make([]interface{}, 0)

	defer func() { _ = iter.Close() }()

	i := 0

	for iter.HasNext() {
		if i < pageSize {
			v, err := iter.Next()
			if err != nil {
				s.logger.Errorf("Unable to get next item from iterator at state.RichQuery: %s", err)
				return nil, 0, UnexpectedError
			}

			entry, err := s.StateGetTransformer(v.Value, target)
			if err != nil {
				s.logger.Errorf("Unable to transform state entry at state.RichQuery: %s", err)
				return nil, 0, UnexpectedError
			}

			entries = append(entries, entry)
		} else {
			break
		}
		i++
	}

	return entries, i, nil
}

func (s *Impl) RichListQuery(query string, target interface{}, pageSize int32, bookmark string) (result []interface{}, newBookmark string, err error) {
	iter, meta, err := s.stub.GetQueryResultWithPagination(query, pageSize, bookmark)

	if err != nil {
		s.logger.Errorf("Unable to get query result with pagination at state.RichListQuery: %s", err)
		return nil, "", SetGetError
	}

	entries := make([]interface{}, 0)

	defer func() { _ = iter.Close() }()

	for iter.HasNext() {
		v, err := iter.Next()

		if err != nil {
			s.logger.Errorf("Unable to get next item from iterator at state.RichListQuery: %s", err)
			return nil, "", UnexpectedError
		}

		entry, err := s.StateGetTransformer(v.Value, target)

		if err != nil {
			s.logger.Errorf("Unable to transform state entry at state.RichListQuery: %s", err)
			return nil, "", UnexpectedError
		}

		entries = append(entries, entry)
	}
	return entries, meta.Bookmark, nil
}
