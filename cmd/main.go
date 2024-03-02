package main

import (
	"fmt"
	"html/template"
	"io"
	"os"

	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	_ "github.com/mattn/go-sqlite3"
)
	
type Template struct {
	tmpl *template.Template
}
	
func newTemplate() *Template {
	return &Template{
		tmpl: template.Must(template.ParseGlob("templates/*.html")),
	}
}

func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.tmpl.ExecuteTemplate(w, name, data)
}

var db *sqlx.DB

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		fmt.Println("error loading godotenv")
	}

	db = sqlx.MustConnect("sqlite3", os.Getenv("AUTH_DIARIES_DB_PATH"))

	var message string
	err = db.Ping()
	if err == nil {
		message = "Successfully connected to DB"
	}

	e := echo.New()

	e.Static("/static", "static")
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.Secure())

	e.Renderer = newTemplate()

	e.GET("/", func(c echo.Context) error {
		return c.Render(200, "index", newPageData(message))
	})

	e.Logger.Fatal(e.Start(":8080"))
}

type PageData struct {
	Message string
}

func newPageData(message string) PageData {
	return PageData{
		Message: message,
	}
}
