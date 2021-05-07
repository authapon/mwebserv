# mwebserv (the tiny web framework)
*This framework is made for personal use, you may take it at your own risk*

## Example

```golang
package main

import (
	m "github.com/authapon/mwebserv"
)

func main() {
	web := m.New()
	web.Get("/", wserve)
	web.Post("/", pserve)
	web.Serve(":9900")
}

func wserve(c *m.MContext) {
	c.WriteString("Hello, world! in GET Method")
}

func pserve(c *m.MContext) {
	c.WriteString("Hello, world! in POST Method")
}
```

## Static file

```golang
func main() {
	web := m.New()
	web.Static("static")
	...
}
```

## Embeded Static file

```golang
func main() {
	web := m.New()
	web.SetAsset(Asset, AssetNames) // This method must call before "StaticBindata"
	web.StaticBindata("static")	
	...
}
``` 

## Route Variable

```golang
func main() {
	web := m.New()
	web.Get("/hello/:name", hello)
	...
}

func hello(c *m.MContext) {
	c.WriteString("Hello, " + c.V["name"])
}
```

## Query Variable

**example query** http://example.com/send?name=jonny&food=apple
```golang
func hello(c *m.MContext) {
	c.WriteString("Hello, " + c.Q.Get("name") + " and you eat " + c.Q.Get("food"))
}
```

## Middleware

```golang
func main() {
	web := m.New()
	web.Use(log)
	...
}

func log(c *m.MContext) {
	fmt.Printf("URL : %s\n", c.R.URL.Path)
	c.Next()
}
```

## JSON read and write

```golang
type DataStruct struct {
	Name string  `json:"name"`
	Food string  `json:"food"`
}

func hello(c *m.MContext) {
	var data DataStruct
	c.ReadJSON(&data)
	...
	...
	...
	c.WriteJSON(&DataStruct{Name: "jonny", Food: "apple"})
}
```

## Render from template

```golang
func main() {
	web := m.New()
	web.View("template") // Add template folder, allow only .html file template with html/template
	web.Get("/", startPage)
	...
}

func startPage(c *m.MContext) {
	data := make(map[string]string)
	data["name"] = "jonny"
	data["food"] = "apple"
	c.Render("index.html", data)
}
```

**File: template/index.html**
```html
{{ define "index.html" }}
<html>
<body>
Hello, {{ .name }} and you eat {{ .food }}
</body>
</html>
{{ end }}
```

## Render from embeded template 

```golang
func main() {
	web := m.New()
	web.SetAsset(Asset, AssetNames) // This method must call before "ViewBindata"
	web.ViewBindata("template")	
	...	
}
```

## Redirect

```golang
func page1(c *m.MContext) {
	c.Redirect("/hello")
}
```

## Get Remote Address IP

```golang
func page(c *m.MContext) {
	ip := c.RemoteAddr()
	...
}
```
