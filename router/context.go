package router

import (
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/hyperledger/fabric/core/chaincode/lib/cid"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/optherium/cckit/convert"
	"github.com/optherium/cckit/state"
)

type (
	// Context of chaincode invoke
	Context interface {
		Stub() shim.ChaincodeStubInterface
		Client() (cid.ClientIdentity, error)
		Response() Response
		Logger() *shim.ChaincodeLogger
		Path() string
		State() state.State
		Time() (time.Time, error)

		ReplaceArgs(args [][]byte) Context // replace args, for usage in preMiddleware
		GetArgs() [][]byte

		// to remove, be only get/set
		Args() InterfaceMap
		Arg(string) interface{}
		ArgString(string) string
		ArgBytes(string) []byte
		ArgInt(string) int
		SetArg(string, interface{})

		Get(string) interface{}
		Set(string, interface{})
		SetEvent(string, interface{}) error
		Errorf(string, ...interface{})
		Debugf(string, ...interface{})
		Criticalf(string, ...interface{})
		Noticef(string, ...interface{})
		Infof(string, ...interface{})
		Warningf(string, ...interface{})
	}

	context struct {
		stub       shim.ChaincodeStubInterface
		logger     *shim.ChaincodeLogger
		path       string
		args       [][]byte
		invokeArgs InterfaceMap
		store      InterfaceMap
	}
)

func (c *context) Stub() shim.ChaincodeStubInterface {
	return c.stub
}

func (c *context) Client() (cid.ClientIdentity, error) {
	return cid.New(c.Stub())
}

func (c *context) Response() Response {
	return ContextResponse{c}
}

func (c *context) Logger() *shim.ChaincodeLogger {
	return c.logger
}

func (c *context) Path() string {
	return string(c.GetArgs()[0])
}

func (c *context) State() state.State {
	return state.New(c.stub)
}

// Time
func (c *context) Time() (time.Time, error) {
	txTimestamp, err := c.stub.GetTxTimestamp()
	if err != nil {
		return time.Unix(0, 0), err
	}
	return time.Unix(txTimestamp.GetSeconds(), int64(txTimestamp.GetNanos())), nil
}

// ReplaceArgs replace args, for usage in preMiddleware
func (c *context) ReplaceArgs(args [][]byte) Context {
	c.args = args
	return c
}

func (c *context) GetArgs() [][]byte {
	if c.args != nil {
		return c.args
	}
	return c.stub.GetArgs()
}

func (c *context) Args() InterfaceMap {
	return c.invokeArgs
}

func (c *context) SetArg(name string, value interface{}) {
	if c.invokeArgs == nil {
		c.invokeArgs = make(InterfaceMap)
	}
	c.invokeArgs[name] = value
}

func (c *context) Arg(name string) interface{} {
	return c.invokeArgs[name]
}

func (c *context) ArgString(name string) string {
	out, _ := c.Arg(name).(string)
	return out
}

func (c *context) ArgBytes(name string) []byte {
	out, _ := c.Arg(name).([]byte)
	return out
}

func (c *context) ArgInt(name string) int {
	out, _ := c.Arg(name).(int)
	return out
}

func (c *context) Set(key string, val interface{}) {
	if c.store == nil {
		c.store = make(InterfaceMap)
	}
	c.store[key] = val
}

func (c *context) Get(key string) interface{} {
	return c.store[key]
}

func (c *context) SetEvent(name string, payload interface{}) error {
	bb, err := convert.ToBytes(payload)

	if err != nil {
		c.logger.Error("Can't convert bytes at Context.SetEvent")
		return UnExpectedError
	}

	err = c.stub.SetEvent(name, bb)

	if err != nil {
		c.logger.Error("Can't SetEvent at Context.SetEvent")
		return UnExpectedError
	}

	return nil
}

func (c *context) Errorf(format string, args ...interface{}) {
	c.logger.Errorf(fmt.Sprintf("[%s]: %s", getCaller(), format), args...)
}

func (c *context) Debugf(format string, args ...interface{}) {
	c.logger.Debugf(fmt.Sprintf("[%s]: %s", getCaller(), format), args...)
}

func (c *context) Infof(format string, args ...interface{}) {
	c.logger.Infof(fmt.Sprintf("[%s]: %s", getCaller(), format), args...)
}

func (c *context) Warningf(format string, args ...interface{}) {
	c.logger.Warningf(fmt.Sprintf("[%s]: %s", getCaller(), format), args...)
}

func (c *context) Criticalf(format string, args ...interface{}) {
	c.logger.Criticalf(fmt.Sprintf("[%s]: %s", getCaller(), format), args...)
}

func (c *context) Noticef(format string, args ...interface{}) {
	c.logger.Noticef(fmt.Sprintf("[%s]: %s", getCaller(), format), args...)
}

func getCaller() string {
	pc := make([]uintptr, 15)
	n := runtime.Callers(3, pc)
	frames := runtime.CallersFrames(pc[:n])
	frame, _ := frames.Next()
	fileSplit := strings.Split(frame.File, "/")
	file := fileSplit[len(fileSplit)-1]
	functionSplit := strings.Split(frame.Function, ".")
	function := functionSplit[len(functionSplit)-1]
	return fmt.Sprintf("%s:%d->%s", file, frame.Line, function)
}
