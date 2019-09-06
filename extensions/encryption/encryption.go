package encryption

import (
	"github.com/hyperledger/fabric/bccsp/factory"
	"github.com/hyperledger/fabric/core/chaincode/shim/ext/entities"
	"github.com/pkg/errors"
	"github.com/optherium/cckit/convert"
	"github.com/optherium/cckit/router"
)

const TransientMapKey = `ENCODE_KEY`

func init() {
	if err := factory.InitFactories(nil); err != nil {
		panic(err)
	}
}

// EncryptArgs convert args to [][]byte and encrypt args with key
func EncryptArgs(key []byte, args ...interface{}) ([][]byte, error) {
	argBytes, err := convert.ArgsToBytes(args...)
	if err != nil {
		return nil, err
	}

	return EncryptArgsBytes(key, argBytes)
}

// EncryptArgsBytes encrypt args with key
func EncryptArgsBytes(key []byte, argsBytes [][]byte) ([][]byte, error) {
	eargs := make([][]byte, len(argsBytes))
	for i, bb := range argsBytes {
		encrypted, err := Encrypt(key, bb)
		if err != nil {
			return nil, errors.Wrap(err, `encryption error`)
		}

		eargs[i] = encrypted
	}
	return eargs, nil
}

// DecryptArgs decrypt args
func DecryptArgs(key []byte, args [][]byte) ([][]byte, error) {
	dargs := make([][]byte, len(args))
	for i, a := range args {

		// do not try to decrypt init function
		if i == 0 && string(a) == router.InitFunc {
			dargs[i] = a
			continue
		}

		decrypted, err := Decrypt(key, a)
		if err != nil {
			return nil, errors.Wrap(err, `decryption error`)
		}
		dargs[i] = decrypted
	}
	return dargs, nil
}

// Encrypt converts value to []byte  and encrypts its with key
func Encrypt(key []byte, value interface{}) ([]byte, error) {
	// TODO: customize  IV
	encrypter, err := entities.NewAES256EncrypterEntity("ID", factory.GetDefault(), key, make([]byte, 16))
	if err != nil {
		return nil, err
	}

	bb, err := convert.ToBytes(value)
	if err != nil {
		return nil, errors.Wrap(err, `convert values to bytes`)
	}

	return encrypter.Encrypt(bb)
}

// Decrypt decrypts value with key
func Decrypt(key, value []byte) ([]byte, error) {
	encrypter, err := entities.NewAES256EncrypterEntity("ID", factory.GetDefault(), key, nil)
	if err != nil {
		return nil, err
	}
	bb := make([]byte, len(value))
	copy(bb, value)
	return encrypter.Decrypt(bb)
}

// TransientMapWithKey creates transient map with encrypting/decrypting key
func TransientMapWithKey(key []byte) map[string][]byte {
	return map[string][]byte{TransientMapKey: key}
}
