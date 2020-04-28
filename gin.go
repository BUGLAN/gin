package gin

import (
	"encoding/json"
	"encoding/xml"
	"github.com/julienschmidt/httprouter"
	"html/template"
	"log"
	"math"
	"net/http"
	"path"
)

const (
	AbortIndex = math.MaxInt8 / 2
)

type (
	// HandlerFunc 可以理解为处理ctx的函数
	HandlerFunc func(*Context)

	// H  map[string]interface{} 快捷方式
	H map[string]interface{}

	// Used internally to collect a error ocurred during a http request.
	// 用于在一个http请求的内部收集错误
	ErrorMsg struct {
		Message string      `json:"msg"`
		Meta    interface{} `json:"meta"`
	}

	// Context is the most important part of gin. It allows us to pass variables between middleware,
	// manage the flow, validate the JSON of a request and render a JSON response for example.
	// Context是gin最终的一部分, 它允许我们在中间件中传递变量
	// 管理数据的流动, 例如可以渲染一个json相应或验证请求的json形式
	Context struct {
		Req      *http.Request
		Writer   http.ResponseWriter
		Keys     map[string]interface{}
		Errors   []ErrorMsg
		Params   httprouter.Params
		handlers []HandlerFunc
		engine   *Engine
		index    int8
	}

	// Used internally to configure router, a RouterGroup is associated with a prefix
	// and an array of handlers (middlewares)
	// 在内部管理一个路由, 一个RouterGroup有一个相关的前缀, 和一个列表的handlers(中间件)
	RouterGroup struct {
		Handlers []HandlerFunc // handlers处理Context的函数列表
		prefix   string // 前缀
		parent   *RouterGroup // 父级的RouterGroup
		engine   *Engine // engine实例
	}

	// Represents the web framework, it wrappers the blazing fast httprouter multiplexer and a list of global middlewares.
	// 代表了gin这个web框架, 包装了超快的httprouter和许多全局中间件
	Engine struct {
		*RouterGroup // 包装router, 拥有RouterGroup的所有方法
		handlers404   []HandlerFunc // gin 用于handler404的方法, 说实话没啥用
		router        *httprouter.Router // 包装的httprouter
		HTMLTemplates *template.Template // 包装的模板实例
	}
)

// Returns a new blank Engine instance without any middleware attached.
// The most basic configuration
// 返回了一个新的空的Engine实例, 没有附加任何的中间件
// 最基础的配置选项
func New() *Engine {
	engine := &Engine{}
	engine.RouterGroup = &RouterGroup{nil, "", nil, engine}
	engine.router = httprouter.New()
	engine.router.NotFound = engine
	return engine
}

// Returns a Engine instance with the Logger and Recovery already attached.
// 返回了一个新的空的Engine实例, 附加了Logger和Recover的中间件
func Default() *Engine {
	engine := New()
	engine.Use(Recovery(), Logger())
	return engine
}

// LoadHTMLTemplates 加载模板实例
func (engine *Engine) LoadHTMLTemplates(pattern string) {
	engine.HTMLTemplates = template.Must(template.ParseGlob(pattern))
}

// Adds handlers for NotFound. It return a 404 code by default.
// 添加了NotFound的handler, 它默认返回一个4040的http code
func (engine *Engine) NotFound404(handlers ...HandlerFunc) {
	engine.handlers404 = handlers
}

// handler404中间件, 似乎没有东西来引用这个函数
func (engine *Engine) handle404(w http.ResponseWriter, req *http.Request) {
	handlers := engine.combineHandlers(engine.handlers404)
	c := engine.createContext(w, req, nil, handlers)
	if engine.handlers404 == nil {
		http.NotFound(c.Writer, c.Req)
	} else {
		c.Writer.WriteHeader(404)
	}

	c.Next()
}

// ServeFiles serves files from the given file system root.
// The path must end with "/*filepath", files are then served from the local
// path /defined/root/dir/*filepath.
// For example if root is "/etc" and *filepath is "passwd", the local file
// "/etc/passwd" would be served.
// Internally a http.FileServer is used, therefore http.NotFound is used instead
// of the Router's NotFound handler.
// To use the operating system's file system implementation,
// use http.Dir:
//     router.ServeFiles("/src/*filepath", http.Dir("/var/www"))
// ServerFiles 提供了来自文件系统的文件服务
// 路径必须以`/*filepath`结尾, 文件将会在本地起起来
// todo: not translation finished
func (engine *Engine) ServeFiles(path string, root http.FileSystem) {
	engine.router.ServeFiles(path, root)
}

// ServeHTTP makes the router implement the http.Handler interface.
// ServeHTTP 使 httprouter 实现了http.Handler的接口
// 实际上是httprouter来处理请求
func (engine *Engine) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	engine.router.ServeHTTP(w, req)
}

// 使用http.ListenAndServe的形式将代码运行起来, 同时也代表engine实现了http.Handler的接口
//type Handler interface {
//	ServeHTTP(ResponseWriter, *Request)
//}
func (engine *Engine) Run(addr string) {
	http.ListenAndServe(addr, engine)
}

/************************************/
/********** ROUTES GROUPING *********/
/************************************/

// createContext 创建context
func (group *RouterGroup) createContext(w http.ResponseWriter, req *http.Request, params httprouter.Params, handlers []HandlerFunc) *Context {
	return &Context{
		Writer:   w,
		Req:      req,
		index:    -1,
		engine:   group.engine,
		Params:   params,
		handlers: handlers,
	}
}

// Adds middlewares to the group, see example code in github.
// 添加中间件到group, 请看示例代码
func (group *RouterGroup) Use(middlewares ...HandlerFunc) {
	group.Handlers = append(group.Handlers, middlewares...)
}

// Group a new router group. You should create add all the routes that share that have common middlwares or same path prefix.
// For example, all the routes that use a common middlware for authorization could be grouped.
// Group 创建一个New的group, 你应该创建所有的routers, 然后这些routers共用公共的middlewares或者是一个相同的路径前缀
// 例如, 一个group中所有routers应该使用一个功能的授权中间件.
func (group *RouterGroup) Group(component string, handlers ...HandlerFunc) *RouterGroup {
	// 构造当前group的路由前缀
	prefix := path.Join(group.prefix, component)
	return &RouterGroup{
		Handlers: group.combineHandlers(handlers), // 组合handlers, 因为可能有group := engine.Group("/some", Auth())
		parent:   group, // 父group
		prefix:   prefix, // 当前group的前缀
		engine:   group.engine, // engine实例指针
	}
}

// Handle registers a new request handle and middlewares with the given path and method.
// The last handler should be the real handler, the other ones should be middlewares that can and should be shared among different routes.
// See the example code in github.
//
// For GET, POST, PUT, PATCH and DELETE requests the  shortcut
// functions can be used.
//
// This function is intended for bulk loading and to allow the usage of less
// frequently used, non-standardized or custom methods (e.g. for internal
// communication with a proxy).
// Handler注册了一个新的请求handler和中间件, 他有给定的路径和方法
// 最后一个handler应该是真是的handler, 其他的应该是中间件, 他可以共享不同的router
// 在github查看实例代码
//
// 对于  GET, POST, PUT, PATCH and DELETE 请求, 各自的快捷韩式应该被使用
//
// 这个函数用于批量加载, 并且不要经常使用这个函数, 不是标准的自定义函数(例如, 内部的沟通和代理)
func (group *RouterGroup) Handle(method, p string, handlers []HandlerFunc) {
	p = path.Join(group.prefix, p)
	handlers = group.combineHandlers(handlers)
	group.engine.router.Handle(method, p, func(w http.ResponseWriter, req *http.Request, params httprouter.Params) {
		group.createContext(w, req, params, handlers).Next()
	})
}

// POST is a shortcut for router.Handle("POST", path, handle)
func (group *RouterGroup) POST(path string, handlers ...HandlerFunc) {
	group.Handle("POST", path, handlers)
}

// GET is a shortcut for router.Handle("GET", path, handle)
func (group *RouterGroup) GET(path string, handlers ...HandlerFunc) {
	group.Handle("GET", path, handlers)
}

// DELETE is a shortcut for router.Handle("DELETE", path, handle)
func (group *RouterGroup) DELETE(path string, handlers ...HandlerFunc) {
	group.Handle("DELETE", path, handlers)
}

// PATCH is a shortcut for router.Handle("PATCH", path, handle)
func (group *RouterGroup) PATCH(path string, handlers ...HandlerFunc) {
	group.Handle("PATCH", path, handlers)
}

// PUT is a shortcut for router.Handle("PUT", path, handle)
func (group *RouterGroup) PUT(path string, handlers ...HandlerFunc) {
	group.Handle("PUT", path, handlers)
}

// combineHandlers 简单的将传入handler加在group.handlers的后面
func (group *RouterGroup) combineHandlers(handlers []HandlerFunc) []HandlerFunc {
	s := len(group.Handlers) + len(handlers)
	h := make([]HandlerFunc, 0, s)
	h = append(h, group.Handlers...)
	h = append(h, handlers...)
	return h
}

/************************************/
/****** FLOW AND ERROR MANAGEMENT****/
/************************************/

// Next should be used only in the middlewares.
// It executes the pending handlers in the chain inside the calling handler.
// See example in github.
// Next 应该被用于中间件中
// 它在链式调用的handler中执行将要发生的handler
func (c *Context) Next() {
	c.index++
	s := int8(len(c.handlers))
	for ; c.index < s; c.index++ {
		c.handlers[c.index](c)
	}
}

// Forces the system to do not continue calling the pending handlers.
// For example, the first handler checks if the request is authorized. If it's not, context.Abort(401) should be called.
// The rest of pending handlers would never be called for that request.
func (c *Context) Abort(code int) {
	c.Writer.WriteHeader(code)
	c.index = AbortIndex
}

// Fail is the same than Abort plus an error message.
// Calling `context.Fail(500, err)` is equivalent to:
// ```
// context.Error("Operation aborted", err)
// context.Abort(500)
// ```
func (c *Context) Fail(code int, err error) {
	c.Error(err, "Operation aborted")
	c.Abort(code)
}

// Attachs an error to the current context. The error is pushed to a list of errors.
// It's a gooc idea to call Error for each error ocurred during the resolution of a request.
// A middleware can be used to collect all the errors and push them to a database together, print a log, or append it in the HTTP response.
func (c *Context) Error(err error, meta interface{}) {
	c.Errors = append(c.Errors, ErrorMsg{
		Message: err.Error(),
		Meta:    meta,
	})
}

/************************************/
/******** METADATA MANAGEMENT********/
/************************************/

// Sets a new pair key/value just for the specefied context.
// It also lazy initializes the hashmap
func (c *Context) Set(key string, item interface{}) {
	if c.Keys == nil {
		c.Keys = make(map[string]interface{})
	}
	c.Keys[key] = item
}

// Returns the value for the given key.
// It panics if the value doesn't exist.
func (c *Context) Get(key string) interface{} {
	var ok bool
	var item interface{}
	if c.Keys != nil {
		item, ok = c.Keys[key]
	} else {
		item, ok = nil, false
	}
	if !ok || item == nil {
		log.Panicf("Key %s doesn't exist", key)
	}
	return item
}

/************************************/
/******** ENCOGING MANAGEMENT********/
/************************************/

// Like ParseBody() but this method also writes a 400 error if the json is not valid.
func (c *Context) EnsureBody(item interface{}) bool {
	if err := c.ParseBody(item); err != nil {
		c.Fail(400, err)
		return false
	}
	return true
}

// Parses the body content as a JSON input. It decodes the json payload into the struct specified as a pointer.
func (c *Context) ParseBody(item interface{}) error {
	decoder := json.NewDecoder(c.Req.Body)
	if err := decoder.Decode(&item); err == nil {
		return Validate(c, item)
	} else {
		return err
	}
}

// Serializes the given struct as a JSON into the response body in a fast and efficient way.
// It also sets the Content-Type as "application/json"
// 快速且高效的讲给定的响应体里面将struct序列化成json格式
// 他也会将content-type改成 application/json
func (c *Context) JSON(code int, obj interface{}) {
	if code >= 0 {
		c.Writer.WriteHeader(code)
	}
	c.Writer.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(c.Writer)
	if err := encoder.Encode(obj); err != nil {
		c.Error(err, obj)
		http.Error(c.Writer, err.Error(), 500)
	}
}

// Serializes the given struct as a XML into the response body in a fast and efficient way.
// It also sets the Content-Type as "application/xml"
func (c *Context) XML(code int, obj interface{}) {
	if code >= 0 {
		c.Writer.WriteHeader(code)
	}
	c.Writer.Header().Set("Content-Type", "application/xml")
	encoder := xml.NewEncoder(c.Writer)
	if err := encoder.Encode(obj); err != nil {
		c.Error(err, obj)
		http.Error(c.Writer, err.Error(), 500)
	}
}

// Renders the HTTP template specified by his file name.
// It also update the HTTP code and sets the Content-Type as "text/html".
// See http://golang.org/doc/articles/wiki/
func (c *Context) HTML(code int, name string, data interface{}) {
	if code >= 0 {
		c.Writer.WriteHeader(code)
	}
	c.Writer.Header().Set("Content-Type", "text/html")
	if err := c.engine.HTMLTemplates.ExecuteTemplate(c.Writer, name, data); err != nil {
		c.Error(err, map[string]interface{}{
			"name": name,
			"data": data,
		})
		http.Error(c.Writer, err.Error(), 500)
	}
}

// Writes the given string into the response body and sets the Content-Type to "text/plain"
func (c *Context) String(code int, msg string) {
	c.Writer.Header().Set("Content-Type", "text/plain")
	c.Writer.WriteHeader(code)
	c.Writer.Write([]byte(msg))
}

// Writes some data into the body stream and updates the HTTP code
func (c *Context) Data(code int, data []byte) {
	c.Writer.WriteHeader(code)
	c.Writer.Write(data)
}
