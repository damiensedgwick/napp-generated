package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"os"
	"time"

	"github.com/gorilla/sessions"
	"github.com/joho/godotenv"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Template struct {
	tmpl *template.Template
}

func newTemplate() *Template {
	return &Template{
		tmpl: template.Must(template.ParseGlob("template/*.html")),
	}
}

func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.tmpl.ExecuteTemplate(w, name, data)
}

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		fmt.Println("error loading godotenv")
	}

	e := echo.New()

	e.Static("/static", "static")
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.Secure())

	store := sessions.NewCookieStore([]byte(os.Getenv("NAPP_GENERATED_COOKIE_STORE_SECRET")))
	e.Use(session.Middleware(store))

	e.Renderer = newTemplate()

	db, err := gorm.Open(sqlite.Open(os.Getenv("NAPP_GENERATED_DB_PATH")), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	db.AutoMigrate(&Lead{}, &User{})

	e.GET("/", func(c echo.Context) error {
		sess, _ := session.Get("session", c)

		if sess.Values["user"] != nil {
			var user User

			err := json.Unmarshal(sess.Values["user"].([]byte), &user)
			if err != nil {
				fmt.Println("error unmarshalling user value")
				return err
			}

			return c.Render(200, "index", newPageData(user, newLeadFormData()))
		}

		return c.Render(200, "index", newPageData(newUser(), newLeadFormData()))
	})

	e.POST("/join-waitlist", func(c echo.Context) error {
		email := c.FormValue("email")

		if leadExists(email, db) {
			leadFormData := LeadFormData{
				Errors: map[string]string{
					"email": "Email is already subscribed",
				},
				Values: map[string]string{
					"email": email,
				},
			}

			return c.Render(422, "waitlist", leadFormData)
		}

		db.Create(&Lead{
			Email: email,
		})

		return c.Render(200, "waitlist", newLeadFormData())
	})

	e.Logger.Fatal(e.Start(":8080"))
}

type PageData struct {
	User     User
	LeadForm LeadFormData
}

func newPageData(user User, leadForm LeadFormData) PageData {
	return PageData{
		User:     user,
		LeadForm: leadForm,
	}
}

type Lead struct {
	gorm.Model
	Email     string
	CreatedAt time.Time
	UpdatedAt *time.Time
}

type LeadFormData struct {
	Errors map[string]string
	Values map[string]string
}

func newLeadFormData() LeadFormData {
	return LeadFormData{
		Errors: map[string]string{},
		Values: map[string]string{},
	}
}

func leadExists(email string, db *gorm.DB) bool {
	var lead Lead
	err := db.First(&lead, "email = ?", email).Error
	if err == gorm.ErrRecordNotFound {
		return false
	}

	return true
}

type User struct {
	gorm.Model
	Name      string
	Email     string
	Password  string
	CreatedAt time.Time
	UpdatedAt *time.Time
}

func newUser() User {
	return User{}
}
