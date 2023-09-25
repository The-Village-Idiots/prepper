package main

import (
	"errors"
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

	if s.SignedIn {
		c.Redirect(http.StatusFound, "/dashboard/")
		return
	}

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
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	frm := struct {
		Username string `form:"username" binding:"required"`
		Password string `form:"password" binding:"required"`
	}{}
	err := c.Bind(&frm)
	if err != nil {
		c.String(http.StatusBadRequest, "Bad Inputs")
		return
	}

	us, err := data.GetUserByName(Database, frm.Username)
	if err != nil {
		// SQL error
		if !errors.Is(err, data.ErrUserNotFound) {
			internalError(c, err)
			return
		}

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