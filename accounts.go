package main

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/ejv2/prepper/data"
	"github.com/gin-gonic/gin"
)

// handleAccounts is the handler for "/account/".
func handleAccounts(c *gin.Context) {
	s := Sessions.Start(c)
	defer s.Update()

	ddat, err := NewDashboardData(s)
	if err != nil {
		internalError(c, err)
		return
	}

	if !ddat.User.Can(data.CapManageUsers) {
		c.String(http.StatusForbidden, "Access Denied")
		return
	}

	us, err := data.GetUsers(Database)
	if err != nil {
		internalError(c, err)
		return
	}

	dat := struct {
		DashboardData
		Users        []data.User
		SessionCount int
	}{ddat, us, Sessions.Len()}

	c.HTML(http.StatusOK, "accounts.gohtml", dat)
}

// handleEditAccount is the handler for "/account/[ID]".
//
// Returns the HTML editor page for the given account.
func handleEditAccount(c *gin.Context) {
	s := Sessions.Start(c)
	defer s.Update()

	ddat, err := NewDashboardData(s)
	if err != nil {
		internalError(c, err)
		return
	}

	suid := c.Param("id")
	if suid == "" {
		c.String(http.StatusBadRequest, "Invalid user ID")
		return
	}
	uid, err := strconv.ParseUint(suid, 10, 32)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid user ID")
		return
	}

	if uint(uid) != ddat.User.ID && !ddat.User.Can(data.CapManageUsers) {
		c.String(http.StatusForbidden, "Access Denied")
		return
	}

	us, err := data.GetUser(Database, uint(uid))
	if err != nil {
		if errors.Is(err, data.ErrUserNotFound) {
			c.String(http.StatusNotFound, "User Not Found")
			return
		}

		internalError(c, err)
		return
	}

	dat := struct {
		DashboardData
		TargetUser data.User
	}{ddat, us}
	c.HTML(http.StatusOK, "accounts-edit.gohtml", dat)
}

func handleNewAccount(c *gin.Context) {
	s := Sessions.Start(c)
	defer s.Update()

	role := data.UserTeacher
	if _, t := c.GetQuery("technician"); t {
		role = data.UserTechnician
	}
	if _, t := c.GetQuery("admin"); t {
		role = data.UserAdmin
	}

	us, err := data.GetUser(Database, s.UserID)
	if err != nil {
		internalError(c, err)
		return
	}

	if !us.Can(data.CapManageUsers) {
		c.String(http.StatusForbidden, "Access Denied")
		return
	}

	u, err := data.NewUser(Database, data.UserRole(role))
	if err != nil {
		internalError(c, err)
		return
	}

	c.Redirect(http.StatusFound, "/account/"+strconv.FormatUint(uint64(u.ID), 10))
}

// handleAccountSwitch is the handler for "/account/switch".
//
// Shows an HTML account selection page which allows the choice of which
// account to impersonate. The switch itself is done by another route.
func handleAccountSwitch(c *gin.Context) {
	s := Sessions.Start(c)
	defer s.Update()

	ddat, err := NewDashboardData(s)
	if err != nil {
		internalError(c, err)
		return
	}

	if !ddat.User.Can(data.CapImpersonate) {
		c.String(http.StatusForbidden, "Access Denied")
		return
	}

	if c.Query("user") != "" {
		suid := c.Query("user")
		uid, err := strconv.ParseUint(suid, 10, 32)
		if err != nil {
			c.Redirect(http.StatusFound, "/account/switch?error")
			return
		}

		u, err := data.GetUser(Database, uint(uid))
		if err != nil {
			c.Redirect(http.StatusFound, "/account/switch?error")
			return
		}

		s.SignIn(u.ID)
		ddat, err = NewDashboardData(s)
		if err != nil {
			c.Redirect(http.StatusFound, "/account/switch?error")
			return
		}

		c.HTML(http.StatusOK, "switch_success.gohtml", ddat)
		return
	}

	users, err := data.GetUsers(Database)
	if err != nil {
		internalError(c, err)
		return
	}

	_, e := c.GetQuery("error")
	dat := struct {
		DashboardData
		Users []data.User
		Error bool
	}{ddat, users, e}
	c.HTML(http.StatusOK, "switch.gohtml", dat)
}

func handleAccountTimetable(c *gin.Context) {
}
