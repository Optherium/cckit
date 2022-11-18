// Package router provides base router for using in chaincode Invoke function
package router

import (
	"fmt"
	"os"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/protos/peer"
	. "github.com/optherium/cckit/errors"
	"github.com/optherium/cckit/response"
)

type (
	MethodType string

	// InterfaceMap map of interfaces
	InterfaceMap map[string]interface{}

	// ContextHandlerFunc use stub context as input parameter
	ContextHandlerFunc func(Context) peer.Response

	// StubHandlerFunc acts as raw chaincode invoke method, accepts stub and returns peer.Response
	StubHandlerFunc func(shim.ChaincodeStubInterface) peer.Response

	// HandlerFunc returns result as interface and error, this is converted to peer.Response via response.Create
	HandlerFunc func(Context) (interface{}, error)

	// ContextMiddlewareFunc Middleware for ContextHandlerFun
	ContextMiddlewareFunc func(nextOrPrev ContextHandlerFunc, pos ...int) ContextHandlerFunc

	// MiddlewareFunc Middleware for HandlerFunc
	MiddlewareFunc func(HandlerFunc, ...int) HandlerFunc

	HandlerMeta struct {
		Hdl  HandlerFunc
		Type MethodType
	}

	// Group of chain code functions
	Group struct {
		Logger *shim.ChaincodeLogger
		Prefix string

		// mapping chaincode method  => handler
		StubHandlers    map[string]StubHandlerFunc
		ContextHandlers map[string]ContextHandlerFunc
		Handlers        map[string]*HandlerMeta

		ContextMiddleware []ContextMiddlewareFunc
		Middleware        []MiddlewareFunc

		PreMiddleware   []ContextMiddlewareFunc
		AfterMiddleware []MiddlewareFunc

		Errs map[error]int32
	}

	Router interface {
		HandleInit(shim.ChaincodeStubInterface)
		Handle(shim.ChaincodeStubInterface)
		Query(path string, handler HandlerFunc, middleware ...MiddlewareFunc) Router
		Invoke(path string, handler HandlerFunc, middleware ...MiddlewareFunc) Router
	}
)

const (
	InitFunc                = `init`
	MethodInvoke MethodType = `invoke`
	MethodQuery  MethodType = `query`
)

func (g *Group) buildHandler() ContextHandlerFunc {
	return func(c Context) peer.Response {
		h := g.handleContext
		// build pre part
		for i := len(g.PreMiddleware) - 1; i >= 0; i-- {
			h = g.PreMiddleware[i](h, i)
		}

		return h(c)
	}
}

// HandleInit handle chaincode init method
func (g *Group) HandleInit(stub shim.ChaincodeStubInterface) peer.Response {
	// Pre context handling Middleware
	h := g.buildHandler()

	// add "init" as first arg
	return h(g.Context(stub).ReplaceArgs(append([][]byte{[]byte(InitFunc)}, stub.GetArgs()...)))
}

// Handle used for using in CC Invoke function
// Must be called after adding new routes using Add function
func (g *Group) Handle(stub shim.ChaincodeStubInterface) peer.Response {
	args := stub.GetArgs()
	if len(args) == 0 {
		return response.Error(ErrEmptyArgs)
	}

	h := g.buildHandler()
	return h(g.Context(stub))
}

func (g *Group) handleContext(c Context) peer.Response {

	// handle standard stub handler (accepts StubInterface, returns peer.Response)
	if stubHandler, ok := g.StubHandlers[c.Path()]; ok {
		g.Logger.Debug(`router stubHandler: `, c.Path())
		return stubHandler(c.Stub())

		// handle context handler (accepts Context, returns peer.Response)
	} else if contextHandler, ok := g.ContextHandlers[c.Path()]; ok {
		g.Logger.Debug(`router contextHandler: `, c.Path())
		h := func(c Context) peer.Response {
			h := contextHandler
			for i := len(g.ContextMiddleware) - 1; i >= 0; i-- {
				h = g.ContextMiddleware[i](h, i)
			}
			return h(c)
		}
		return h(c)
	} else if handlerMeta, ok := g.Handlers[c.Path()]; ok {

		g.Logger.Debug(`router handler: `, c.Path())
		h := func(c Context) (interface{}, error) {

			c.SetHandler(handlerMeta)
			h := handlerMeta.Hdl
			for i := len(g.Middleware) - 1; i >= 0; i-- {
				h = g.Middleware[i](h, i)
			}

			for i := 0; i <= len(g.AfterMiddleware)-1; i++ {
				h = g.AfterMiddleware[i](h, 0)
			}

			return h(c)
		}

		data, err := h(c)
		resp := response.Create(data, err)
		if resp.Status != shim.OK {
			g.Logger.Errorf(`%s: %s: %s`, ErrHandlerError, c.Path(), resp.Message)
			resp.Status = g.getErrorCode(err)
		}
		return resp
	}

	err := fmt.Errorf(`%s: %s`, ErrMethodNotFound, c.Path())
	g.Logger.Error(err)
	return shim.Error(err.Error())
}

func (g *Group) getErrorCode(err error) int32 {
	if val, ok := g.Errs[err]; ok {
		return val
	}

	return shim.ERROR
}

func (g *Group) Pre(middleware ...ContextMiddlewareFunc) *Group {
	g.PreMiddleware = append(g.PreMiddleware, middleware...)
	return g
}

func (g *Group) After(middleware ...MiddlewareFunc) *Group {
	g.AfterMiddleware = append(g.AfterMiddleware, middleware...)
	return g
}

// Use Middleware function in chain code functions group
func (g *Group) Use(middleware ...MiddlewareFunc) *Group {
	g.Middleware = append(g.Middleware, middleware...)
	return g
}

// Group gets new group using presented path
// New group can be used as independent
func (g *Group) Group(path string) *Group {
	return &Group{
		Logger:          g.Logger,
		Prefix:          g.Prefix + path,
		StubHandlers:    g.StubHandlers,
		ContextHandlers: g.ContextHandlers,
		Handlers:        g.Handlers,
		Middleware:      g.Middleware,
	}
}

// StubHandler adds new stub handler using presented path
func (g *Group) StubHandler(path string, fn StubHandlerFunc) *Group {
	g.StubHandlers[g.Prefix+path] = fn
	return g
}

// ContextHandler adds new context handler using presented path
func (g *Group) ContextHandler(path string, fn ContextHandlerFunc) *Group {
	g.ContextHandlers[g.Prefix+path] = fn
	return g
}

// Query defines handler and Middleware for querying chaincode method (no state change, no send to orderer)
func (g *Group) Query(path string, handler HandlerFunc, middleware ...MiddlewareFunc) *Group {
	return g.addHandler(MethodQuery, path, handler, middleware...)
}

// Invoke defines handler and Middleware for invoke chaincode method  (state change,  need to send to orderer)
func (g *Group) Invoke(path string, handler HandlerFunc, middleware ...MiddlewareFunc) *Group {
	return g.addHandler(MethodInvoke, path, handler, middleware...)
}

func (g *Group) addHandler(t MethodType, path string, handler HandlerFunc, middleware ...MiddlewareFunc) *Group {
	g.Handlers[g.Prefix+path] = &HandlerMeta{
		Type: t,
		Hdl: func(context Context) (interface{}, error) {
			h := handler
			for i := len(middleware) - 1; i >= 0; i-- {
				h = middleware[i](h, i)
			}
			return h(context)
		}}
	return g
}

func (g *Group) Init(handler HandlerFunc, middleware ...MiddlewareFunc) *Group {
	return g.Invoke(InitFunc, handler, middleware...)
}

// Context returns chain code invoke context  for provided path and stub
func (g *Group) Context(stub shim.ChaincodeStubInterface) Context {
	return NewContext(stub, g.Logger)
}

// New group of chain code functions
func New(name string) *Group {
	g := new(Group)
	g.Logger = NewLogger(name)
	g.StubHandlers = make(map[string]StubHandlerFunc)
	g.ContextHandlers = make(map[string]ContextHandlerFunc)
	g.Handlers = make(map[string]*HandlerMeta)

	return g
}

// NewWithErrorMappings - new group of chain code functions with error mappings
func NewWithErrorMappings(name string, errs map[error]int32) *Group {
	g := new(Group)
	g.Logger = NewLogger(name)
	g.StubHandlers = make(map[string]StubHandlerFunc)
	g.ContextHandlers = make(map[string]ContextHandlerFunc)
	g.Handlers = make(map[string]*HandlerMeta)
	g.Errs = errs

	return g
}

// NewContext creates new instance of router.Context
func NewContext(stub shim.ChaincodeStubInterface, logger *shim.ChaincodeLogger) *context {
	return &context{
		stub:   stub,
		logger: logger,
	}
}

// NewLogger creates new instance of shim.ChaincodeLogger
func NewLogger(name string) *shim.ChaincodeLogger {
	logger := shim.NewLogger(name)
	loggingLevel, err := shim.LogLevel(os.Getenv(`CORE_CHAINCODE_LOGGING_LEVEL`))
	if err == nil {
		logger.SetLevel(loggingLevel)
	}

	return logger
}
