package mwebserv

import (
	"encoding/json"
	"errors"
	"html/template"
	"io/ioutil"
	"mime"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type (
	MWeb struct {
		webserver     *http.Server
		post          map[string]MHandler
		get           map[string]MHandler
		middleware    []MHandler
		NotFound      MHandler
		static        string
		staticBindata string
		asset         func(string) ([]byte, error)
		assetnames    func() []string
		view          *template.Template
	}

	MContext struct {
		W            http.ResponseWriter
		R            *http.Request
		Data         map[string]interface{}
		V            map[string]string
		Q            url.Values
		server       *MWeb
		handlerChain int
		handler      []MHandler
	}

	MHandler func(*MContext)
)

func New() *MWeb {
	m := new(MWeb)
	m.webserver = &http.Server{}
	m.post = make(map[string]MHandler)
	m.get = make(map[string]MHandler)
	m.middleware = make([]MHandler, 0)
	m.NotFound = notFound
	m.static = ""
	m.staticBindata = ""
	return m
}

func (m *MWeb) ReadTimeout(t time.Duration) {
	m.webserver.ReadTimeout = t
}

func (m *MWeb) WriteTimeout(t time.Duration) {
	m.webserver.WriteTimeout = t
}

func (m *MWeb) SetAsset(asset func(string) ([]byte, error), assetnames func() []string) {
	m.asset = asset
	m.assetnames = assetnames
}

func (m *MWeb) Static(dir string) {
	m.static = dir
}

func (m *MWeb) StaticBindata(dir string) {
	m.staticBindata = dir
}

func (m *MWeb) Post(r string, f MHandler) {
	m.post[r] = f
}

func (m *MWeb) Get(r string, f MHandler) {
	m.get[r] = f
}

func (m *MWeb) Use(f MHandler) {
	m.middleware = append(m.middleware, f)
}

func (m *MWeb) View(dir string) {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return
	}
	t := template.New("")
	for _, file := range files {
		fname := filepath.Join(dir, file.Name())
		if file.IsDir() {
			continue
		}
		if filepath.Ext(fname) == ".html" {
			data, err := ioutil.ReadFile(fname)
			if err != nil {
				continue
			}
			t2, err := t.Parse(string(data))
			if err != nil {
				continue
			}
			t = t2
		}
	}
	m.view = t
}

func (m *MWeb) ViewBindata(dir string) {
	fname := m.assetnames()
	ndir := filepath.Join(dir) + "/"
	t := template.New("")
	for _, v := range fname {
		if ndir != v[:len(ndir)] {
			continue
		}

		if filepath.Ext(v) == ".html" {
			data, err := m.asset(v)
			if err != nil {
				continue
			}

			t2, err := t.Parse(string(data))
			if err != nil {
				continue
			}
			t = t2
		}
	}
	m.view = t
}

func (m *MWeb) Serve(host string) {
	m.webserver.Addr = host
	m.webserver.Handler = m
	m.webserver.ListenAndServe()
}

func notFound(c *MContext) {
	c.W.Header().Set("Content-Type", "text/plain")
	c.W.WriteHeader(http.StatusNotFound)
	c.W.Write([]byte("Error 404 : Page Not Found!!!"))
}

func processQueryURI(c *MContext) {
	u, err := url.Parse(c.R.RequestURI)
	if err != nil {
		c.Q = url.Values{}
		return
	}
	c.Q = u.Query()
}

func (m *MWeb) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	context := new(MContext)
	context.server = m
	context.handlerChain = 0
	context.handler = make([]MHandler, 0)
	context.W = w
	context.R = r
	context.Data = make(map[string]interface{})
	processQueryURI(context)
	method := context.R.Method
	if method == "GET" {
		route := matchRoute(m.get, context)
		context.makeHandlerChain(route)
	} else if method == "POST" {
		route := matchRoute(m.post, context)
		context.makeHandlerChain(route)
	}
}

func (c *MContext) makeHandlerChain(route string) {
	for _, v := range c.server.middleware {
		c.handler = append(c.handler, v)
	}
	c.handler = append(c.handler, c.server.defaultHandler)
	c.Data["route"] = route
	c.handler[0](c)
}

func (c *MContext) Next() {
	c.handlerChain++
	c.handler[c.handlerChain](c)
	c.handlerChain--
}

func (m *MWeb) defaultHandler(c *MContext) {
	method := c.R.Method
	route := c.Data["route"].(string)
	if method == "GET" {
		if route != "" {
			m.get[route](c)
			return
		}

		if m.static != "" {
			err := m.serveStatic("", c)
			if err == nil {
				return
			}
		}

		if m.staticBindata != "" {
			err := m.serveStaticBindata("", c)
			if err == nil {
				return
			}
		}

		m.NotFound(c)

	} else if method == "POST" {
		if route == "" {
			m.NotFound(c)
		} else {
			m.post[route](c)
		}
	}
}

func (m *MWeb) serveStaticBindata(fname string, c *MContext) error {
	filename := ""
	if fname == "" {
		filename = filepath.Join(m.staticBindata, c.R.URL.Path)
	} else {
		filename = filepath.Join(m.staticBindata, fname)
	}

	data, err := m.asset(filename)
	if err != nil {
		filename = filepath.Join(filename, "index.html")
		data, err = m.asset(filepath.Join(filename))
		if err != nil {
			return errors.New("error to open file")
		}
	}

	ctype := mime.TypeByExtension(filepath.Ext(filename))
	if ctype == "" {
		ctype = "application/octet-stream"
	}

	c.W.Header().Set("Content-Type", ctype)
	c.W.WriteHeader(http.StatusOK)

	c.W.Write(data)

	return nil
}

func (m *MWeb) serveStatic(fname string, c *MContext) error {
	filename := ""
	if fname == "" {
		filename = filepath.Join(m.static, c.R.URL.Path)
	} else {
		filename = filepath.Join(m.static, fname)
	}

	file, err := os.Open(filename)
	if err != nil {
		filename = filepath.Join(filename, "index.html")
		file, err = os.Open(filename)
		if err != nil {
			return errors.New("error to open file")
		}
	}
	defer file.Close()

	http.ServeFile(c.W, c.R, filename)

	return nil
}

func matchRoute(route map[string]MHandler, c *MContext) string {
	c.V = make(map[string]string)
	path := normPath(strings.Split(c.R.URL.Path, "/"))
	for k := range route {
		kpath := normPath(strings.Split(k, "/"))
		pathMatch := true
		if len(kpath) == len(path) {
			for kk := range path {
				if path[kk] == "" {
					if kpath[kk] == "" {
						continue
					}
				}
				if kpath[kk][0] == ':' {
					c.V[kpath[kk][1:]] = path[kk]
					continue
				}
				if path[kk] == kpath[kk] {
					continue
				}
				pathMatch = false
				break
			}
		} else {
			continue
		}

		if pathMatch {
			return k
		}
	}
	return ""
}

func normPath(p []string) []string {
	if len(p) > 1 {
		if p[len(p)-1] == "" {
			return p[:len(p)-1]
		}
	}
	return p
}

func (c *MContext) ReadJSON(v interface{}) error {
	if c.R.Body == nil {
		return errors.New("empty body")
	}

	rawData, err := ioutil.ReadAll(c.R.Body)
	if err != nil {
		return err
	}

	return json.Unmarshal(rawData, v)
}

func (c *MContext) WriteJSON(v interface{}) {
	c.WriteJSONStatus(http.StatusOK, v)
}

func (c *MContext) WriteJSONStatus(status int, v interface{}) {
	b, err := json.Marshal(v)
	if err != nil {
		return
	}

	c.W.Header().Set("Content-Type", "application/json")
	c.W.WriteHeader(status)

	c.W.Write(b)
}

func (c *MContext) WriteString(data string) {
	c.WriteStringStatus(http.StatusOK, data)
}

func (c *MContext) WriteStringStatus(status int, data string) {
	c.W.Header().Set("Content-Type", "text/plain")
	c.W.WriteHeader(status)

	c.W.Write([]byte(data))
}

func (c *MContext) WriteHTML(data string) {
	c.WriteHTMLStatus(http.StatusOK, data)
}

func (c *MContext) WriteHTMLStatus(status int, data string) {
	c.W.Header().Set("Content-Type", "text/html")
	c.W.WriteHeader(status)

	c.W.Write([]byte(data))
}

func (c *MContext) Render(tname string, data interface{}) {
	c.RenderStatus(http.StatusOK, tname, data)
}

func (c *MContext) RenderStatus(status int, tname string, data interface{}) {
	c.W.Header().Set("Content-Type", "text/html")
	c.W.WriteHeader(status)

	c.server.view.ExecuteTemplate(c.W, tname, data)
}

func (c *MContext) ServeFileStatic(fname string) {
	c.server.serveStatic(fname, c)
}

func (c *MContext) ServeFileStaticBindata(fname string) {
	c.server.serveStaticBindata(fname, c)
}

func (c *MContext) Redirect(route string) {
	http.Redirect(c.W, c.R, route, http.StatusTemporaryRedirect)
}

func (c *MContext) RemoteAddr() string {
	xforward := c.R.Header.Get("X-Forwarded-For")
	if xforward != "" {
		return strings.TrimSpace(xforward)
	}
	xforward = c.R.Header.Get("X-Real-IP")
	if xforward != "" {
		return strings.TrimSpace(xforward)
	}
	addr := strings.TrimSpace(c.R.RemoteAddr)
	if addr != "" {
		if ip, _, err := net.SplitHostPort(addr); err == nil {
			return ip
		}
	}

	return addr
}
