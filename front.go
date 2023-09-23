package main

import (
	"log"
	"net/http"

	"github.com/ejv2/prepper/data"
	"github.com/gin-gonic/gin"
)

// handleRoot is the handler for "/"
//
// If the request comes from an authenticated user, the user is redirected to
// their dashboard. Else, the user is redirected to the login page.
func handleRoot(c *gin.Context) {
	s := Sessions.Start(c)
	defer s.Update()

	// Redirect based on login state
	if s.SignedIn {
		c.Redirect(http.StatusFound, "/dashboard/")
	} else {
		c.Redirect(http.StatusFound, "/login")
	}
}

// handleLogin is the handler for GET "/login"
//
// Returns the HTML login page. This *is not* the endpoint for POST request
// logins.
func handleLogin(c *gin.Context) {
	s := Sessions.Start(c)
	defer s.Update()

	_, fail := c.GetQuery("error")
	_, out := c.GetQuery("out")

	c.HTML(http.StatusOK, "login.gohtml", gin.H{
		"LoginFailed": fail,
		"LoggedOut":   out,
	})
}

// handleLoginAttempt is the handler for POST "/login"
//
// This is used to submit the form result and always returns status 302.
func handleLoginAttempt(c *gin.Context) {
	s := Sessions.Start(c)

	if s.SignedIn {
		c.Redirect(http.StatusFound, "/dashboard/")
		return
	}

	frm := struct {
		Username string `form:"username" binding:"required"`
		Password string `form:"password" binding:"required"`
	}{}
	c.Bind(&frm)

	us := data.User{Username: frm.Username}
	err := Database.Where(&us).First(&us).Error
	if err != nil {
		log.Print("Login attempt failed for username \"", frm.Username, "\" (bad username)")
		c.Redirect(http.StatusFound, "/login?error")
		return
	}

	if !us.Password.Matches(frm.Password) {
		log.Print("Login attempt failed for user \"", us.Username, "\" (bad password)")
		c.Redirect(http.StatusFound, "/login?error")
		return
	}

	s.SignIn(us.ID)
	s.Update()

	c.Redirect(http.StatusFound, "/dashboard/")
}

// handleLogout is the handler for "/logout"
//
// Resets the current session to defaults for a non-authenticated user.
func handleLogout(c *gin.Context) {
	s := Sessions.Start(c)

	s.Logout()
	s.Update()

	c.Redirect(http.StatusFound, "/login?out")
}
