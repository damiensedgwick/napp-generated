package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/mail"
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
	e.Renderer = newTemplate()
	e.Static("/static", "static")
	e.Use(middleware.Recover())
	e.Use(middleware.Secure())
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: "method=${method}, uri=${uri}, status=${status}\n",
	}))
	store := sessions.NewCookieStore([]byte(os.Getenv("NAPP_GENERATED_COOKIE_STORE_SECRET")))
	e.Use(session.Middleware(store))

	db, err := gorm.Open(sqlite.Open(os.Getenv("NAPP_GENERATED_DB_PATH")), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}
	db.AutoMigrate(&Lead{}, &User{})

	e.GET("/", homepageHandler())
	e.POST("/join-waitlist", joinWaitlistHandler(db))

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

func homepageHandler() echo.HandlerFunc {
	return func(c echo.Context) error {
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

func joinWaitlistHandler(db *gorm.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		email := c.FormValue("email")
		_, err := mail.ParseAddress(email)
		if err != nil {
			return c.Render(422, "waitlist", LeadFormData{
				Errors: map[string]string{
					"email": "Oops! That email appears to be invalid",
				},
				Values: map[string]string{
					"email": email,
				},
			})
		}

		if leadExists(email, db) {
			return c.Render(422, "waitlist", LeadFormData{
				Errors: map[string]string{
					"email": "Oops! It appears you are already subscribed",
				},
				Values: map[string]string{
					"email": email,
				},
			})
		}

		lead := Lead{
			Email: email,
		}

		if err := db.Create(&lead).Error; err != nil {
			return c.Render(500, "waitlist", LeadFormData{
				Errors: map[string]string{
					"email": "Oops! It appears we have had an error",
				},
				Values: map[string]string{},
			})
		}

		return c.Render(200, "waitlist", newLeadFormData())
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
