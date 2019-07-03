package state

import (
	"os"
	"strings"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/optherium/cckit/convert"
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

// Keyer interface for entity containing logic of its key creation
type Keyer interface {
	Key() ([]string, error)
}

type KeyerFunc func(string) ([]string, error)

type FromBytesTransformer func(bb []byte, config ...interface{}) (interface{}, error)
type ToBytesTransformer func(v interface{}, config ...interface{}) ([]byte, error)
type KeyPartsTransformer func(key interface{}) ([]string, error)

func ConvertFromBytes(bb []byte, config ...interface{}) (interface{}, error) {
	// convertation not needed
	if len(config) == 0 {
		return bb, nil
	}
	return convert.FromBytes(bb, config[0])
}

func ConvertToBytes(v interface{}, config ...interface{}) ([]byte, error) {
	return convert.ToBytes(v)
}

// KeyParts returns string parts of composite key
func KeyParts(key interface{}) ([]string, error) {
	switch key.(type) {
	case Keyer:
		return key.(Keyer).Key()
	case string:
		return []string{key.(string)}, nil
	case []string:
		return key.([]string), nil
	}

	return nil, UnableToCreateKeyError
}

// State interface for chain code CRUD operations
type State interface {
	Get(key interface{}, target ...interface{}) (result interface{}, err error)
	GetInt(key interface{}, defaultValue int) (result int, err error)
	GetHistory(key interface{}, target interface{}) (result HistoryEntryList, err error)
	Exists(key interface{}) (exists bool, err error)
	Put(key interface{}, value ...interface{}) (err error)
	Insert(key interface{}, value ...interface{}) (err error)
	List(objectType interface{}, target ...interface{}) (result []interface{}, err error)
	PaginateList(objectType interface{}, target interface{}, pageSize int32, start string) (result []interface{}, end string, err error)
	Delete(key interface{}) (err error)
}

type StateImpl struct {
	stub                shim.ChaincodeStubInterface
	KeyParts            KeyPartsTransformer
	StateGetTransformer FromBytesTransformer
	StatePutTransformer ToBytesTransformer
	logger              *shim.ChaincodeLogger
}

// New creates wrapper on shim.ChaincodeStubInterface working with state
func New(stub shim.ChaincodeStubInterface) *StateImpl {
	logger := shim.NewLogger("StateLogger")
	loggingLevel, err := shim.LogLevel(os.Getenv(`CORE_CHAINCODE_LOGGING_LEVEL`))
	if err == nil {
		logger.SetLevel(loggingLevel)
	}
	return &StateImpl{
		stub:                stub,
		logger:              logger,
		KeyParts:            KeyParts,
		StateGetTransformer: ConvertFromBytes,
		StatePutTransformer: ConvertToBytes}
}

func (s *StateImpl) Key(key interface{}) (string, error) {
	keyParts, err := s.KeyParts(key)
	if err != nil {
		return ``, err
	}
	return KeyFromParts(s.stub, keyParts)
}

// Get data by key from state, trying to convert to target interface
func (s *StateImpl) Get(key interface{}, config ...interface{}) (result interface{}, err error) {
	strKey, err := s.Key(key)

	if err != nil {
		s.logger.Error("can't compose key at State.Get")
		return nil, UnExpectedError
	}

	bb, err := s.stub.GetState(strKey)

	if err != nil {
		s.logger.Error("can't get bytes from ledger at State.Get")
		return nil, SetGetError
	}

	if bb == nil || len(bb) == 0 {
		// default value
		if len(config) >= 2 {
			return config[1], nil
		}
		s.logger.Error(strings.Replace(strKey, "\x00", ` | `, -1), KeyNotFoundError.Error())
		return nil, KeyNotFoundError
	}

	result, err = s.StateGetTransformer(bb, config...)

	if err != nil {
		s.logger.Error("can't make StateGetTransformer at State.Get")
		return nil, SetGetError
	}

	return result, nil
}

func (s *StateImpl) GetInt(key interface{}, defaultValue int) (result int, err error) {
	val, err := s.Get(key, convert.TypeInt, defaultValue)

	if err != nil {
		return 0, UnExpectedError
	}

	return val.(int), nil
}

// GetHistory by key from state, trying to convert to target interface
func (s *StateImpl) GetHistory(key interface{}, target interface{}) (result HistoryEntryList, err error) {
	strKey, err := s.Key(key)

	if err != nil {
		s.logger.Error("can't compose key at State.GetHistory")
		return nil, UnExpectedError
	}

	iter, err := s.stub.GetHistoryForKey(strKey)

	if err != nil {
		s.logger.Error("can't get iterator at State.GetHistory")
		return nil, SetGetError
	}

	defer func() { _ = iter.Close() }()

	results := HistoryEntryList{}

	for iter.HasNext() {
		state, err := iter.Next()

		if err != nil {
			s.logger.Error("can't get iterator next item at State.GetHistory")
			return nil, SetGetError
		}

		value, err := s.StateGetTransformer(state.Value, target)

		if err != nil {
			s.logger.Error("can't get make StateGetTransformer at State.GetHistory")
			return nil, UnExpectedError
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
func (s *StateImpl) Exists(key interface{}) (exists bool, err error) {
	stringKey, err := s.Key(key)

	if err != nil {
		s.logger.Error("can't compose key at State.Exists")
		return false, UnExpectedError
	}

	bb, err := s.stub.GetState(stringKey)

	if err != nil {
		s.logger.Error("can't get state from stub at State.Exists")
		return false, SetGetError
	}

	return !(bb == nil || len(bb) == 0), nil
}

// List data from state using objectType prefix in composite key, trying to conver to target interface.
// Keys -  additional components of composite key
func (s *StateImpl) List(objectType interface{}, target ...interface{}) (result []interface{}, err error) {
	keyParts, err := s.KeyParts(objectType)

	if err != nil {
		s.logger.Error("can't compose key at State.List")
		return nil, UnExpectedError
	}

	iter, err := s.stub.GetStateByPartialCompositeKey(keyParts[0], keyParts[1:])

	if err != nil {
		s.logger.Error("can't get iterator at State.List")
		return nil, SetGetError
	}

	entries := make([]interface{}, 0)
	defer func() { _ = iter.Close() }()

	for iter.HasNext() {
		v, err := iter.Next()

		if err != nil {
			s.logger.Error("can't get next item from iterator at State.List")
			return nil, SetGetError
		}

		entry, err := s.StateGetTransformer(v.Value, target...)

		if err != nil {
			s.logger.Error("can't make StateGetTransformer at State.List")
			return nil, UnExpectedError
		}

		entries = append(entries, entry)
	}
	return entries, nil
}

func (s *StateImpl) PaginateList(objectType interface{}, target interface{}, limit int32, start string) (result []interface{}, end string, err error) {
	keyParts, err := s.KeyParts(objectType)

	if err != nil {
		s.logger.Error("can't compose key at PaginateList.List")
		return nil, "", UnExpectedError
	}

	iter, meta, err := s.stub.GetStateByPartialCompositeKeyWithPagination(keyParts[0], keyParts[1:], limit, start)

	if err != nil {
		s.logger.Error("can't get iterator at PaginateList.List")
		return nil, "", SetGetError
	}

	end = meta.Bookmark

	entries := make([]interface{}, 0)
	defer func() { _ = iter.Close() }()

	for iter.HasNext() {
		v, err := iter.Next()

		if err != nil {
			s.logger.Error("can't get next item from iterator at PaginateList.List")
			return nil, "", SetGetError
		}

		entry, err := s.StateGetTransformer(v.Value, target)

		if err != nil {
			s.logger.Error("can't make StateGetTransformer at PaginateList.List")
			return nil, "", UnExpectedError
		}

		entries = append(entries, entry)
	}
	return entries, end, nil
}

func getValue(key interface{}, values []interface{}) (interface{}, error) {
	switch len(values) {

	// key is struct implementing keyer interface
	case 0:
		if _, ok := key.(Keyer); !ok {
			return nil, KeyNotSupportKeyerInterfaceError
		}
		return key, nil
	case 1:
		return values[0], nil
	default:
		return nil, AllowOnlyOneValueError
	}
}

// Put data value in state with key, trying convert data to []byte
func (s *StateImpl) Put(key interface{}, values ...interface{}) (err error) {
	value, err := getValue(key, values)

	if err != nil {
		s.logger.Error("can't make getValue at State.Put")
		return UnExpectedError
	}

	bb, err := s.StatePutTransformer(value)

	if err != nil {
		s.logger.Error("can't make StatePutTransformer at State.Put")
		return UnExpectedError
	}

	stringKey, err := s.Key(key)

	if err != nil {
		s.logger.Error("can't make Key at State.Put")
		return UnExpectedError
	}

	err = s.stub.PutState(stringKey, bb)

	if err != nil {
		s.logger.Error("can't put value in ledger at State.Put")
		return SetGetError
	}

	return nil
}

// Insert value into chaincode state, returns error if key already exists
func (s *StateImpl) Insert(key interface{}, values ...interface{}) (err error) {
	exists, err := s.Exists(key)

	if err != nil {
		s.logger.Error("can't make Exists at State.Insert")
		return err
	}

	if exists {
		strKey, _ := s.Key(key)
		s.logger.Error(strKey, AlreadyExistsError.Error())
		return AlreadyExistsError
	}

	value, err := getValue(key, values)

	if err != nil {
		s.logger.Error("can't make getValue at State.Insert")
		return UnExpectedError
	}

	err = s.Put(key, value)

	if err != nil {
		s.logger.Error("can't put value at State.Insert")
		return SetGetError
	}

	return nil
}

// Delete entry from state
func (s *StateImpl) Delete(key interface{}) (err error) {
	stringKey, err := s.Key(key)
	if err != nil {
		s.logger.Error("can't compose key at State.Delete")
		return UnExpectedError
	}

	err = s.stub.DelState(stringKey)

	if err != nil {
		s.logger.Error("can't delete value at State.Delete")
		return SetGetError
	}

	return nil
}

// KeyFromParts creates composite key by string slice
func KeyFromParts(stub shim.ChaincodeStubInterface, keyParts []string) (string, error) {
	switch len(keyParts) {
	case 0:
		return ``, KeyPartsLengthError
	case 1:
		return keyParts[0], nil
	default:
		return stub.CreateCompositeKey(keyParts[0], keyParts[1:])
	}
}

// KeyError error with key
func KeyError(strKey string) error {
	return errors.New(strings.Replace(strKey, "\x00", ` | `, -1))
}

type stringKeyer struct {
	str   string
	keyer KeyerFunc
}

func (sk stringKeyer) Key() ([]string, error) {
	return sk.keyer(sk.str)
}

// StringKeyer constructor for struct implementing Keyer interface
func StringKeyer(str string, keyer KeyerFunc) Keyer {
	return stringKeyer{str, keyer}
}
