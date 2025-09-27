package main

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
	"gopkg.in/cas.v2"
)

// Your CAS SERVER URL
var casURL = "https://cas.example.com"

type templateBinding struct {
	Username   string
	Attributes cas.UserAttributes
}

func main() {
	u, err := url.Parse(casURL)
	if err != nil {
		log.Fatalf("failed to parse CAS URL: %v", err)
	}

	client := cas.NewClient(&cas.Options{URL: u, CasVersion: cas.CASVERSION3})

	router := gin.Default()
	router.Use(casGinMiddleware(client))

	router.GET("/", func(c *gin.Context) {
		tmpl, err := template.New("index.html").Parse(indexHTML)
		if err != nil {
			c.String(http.StatusInternalServerError, fmt.Sprintf(error500, err))
			return
		}

		binding := &templateBinding{
			Username:   cas.Username(c.Request),
			Attributes: cas.Attributes(c.Request),
		}

		buf := new(bytes.Buffer)
		if err := tmpl.Execute(buf, binding); err != nil {
			c.String(http.StatusInternalServerError, fmt.Sprintf(error500, err))
			return
		}

		c.Data(http.StatusOK, "text/html; charset=utf-8", buf.Bytes())
	})

	server := &http.Server{
		Addr:    ":9999",
		Handler: client.Handle(router),
	}

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}

func casGinMiddleware(client *cas.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		handler := client.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c.Request = r
			c.Next()
		}))

		handler.ServeHTTP(c.Writer, c.Request)

	}
}

const indexHTML = `<!DOCTYPE html>
<html>
  <head>
    <title>Welcome {{.Username}}</title>
  </head>
  <body>
    <h1>Welcome {{.Username}} <a href="/logout">Logout</a></h1>
    <p>Your attributes are:</p>
    <ul>{{range $key, $values := .Attributes}}
      <li>{{$len := len $values}}{{$key}}:{{if gt $len 1}}
        <ul>{{range $values}}
          <li>{{.}}</li>{{end}}
        </ul>
      {{else}} {{index $values 0}}{{end}}</li>{{end}}
    </ul>
  </body>
</html>
`

const error500 = `<!DOCTYPE html>
<html>
  <head>
    <title>Error 500</title>
  </head>
  <body>
    <h1>Error 500</h1>
    <p>%v</p>
  </body>
</html>
`
