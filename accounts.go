package main

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/ejv2/prepper/data"
	"github.com/ejv2/prepper/isams"
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

// handleChangePassword is the handler for "/account/password".
//
// This only allows changes for the current user and such is not an
// authenticated route.
func handleChangePassword(c *gin.Context) {
	s := Sessions.Start(c)
	defer s.Update()

	ddat, err := NewDashboardData(s)
	if err != nil {
		internalError(c, err)
		return
	}

	_, erro := c.GetQuery("error")
	_, success := c.GetQuery("success")
	dat := struct {
		DashboardData
		Error   bool
		Success bool
	}{ddat, erro, success}

	c.HTML(http.StatusOK, "password.gohtml", dat)
}

// handleChangePasswordAttempt is the handler for POST @ "/account/password".
//
// This route actually changes the password given by the user.
func handleChangePasswordAttempt(c *gin.Context) {
	s := Sessions.Start(c)
	defer s.Update()

	ddat, err := NewDashboardData(s)
	if err != nil {
		internalError(c, err)
		return
	}

	oldPass, ok := c.GetPostForm("old_password")
	if !ok {
		c.Redirect(http.StatusFound, "/account/password?error")
		return
	}
	newpass, ok := c.GetPostForm("new_password")
	if !ok {
		c.Redirect(http.StatusFound, "/account/password?error")
		return
	}

	if !ddat.User.Password.Matches(oldPass) {
		c.Redirect(http.StatusFound, "/account/password?error")
		return
	}

	if ddat.User.SetPassword(newpass) != nil {
		c.Redirect(http.StatusFound, "/account/password?error")
		return
	}

	if Database.Updates(&ddat.User).Error != nil {
		internalError(c, err)
		return
	}

	c.Redirect(http.StatusFound, "/account/password?success")
}

// handleAccountTimetable is the handler for "/account/[ID]/timetable"
//
// This allows changes to a user's timetable. Changes to the current user's
// timetable are allowed unauthenticated, but to change others the user must be
// at least a technician.
func handleAccountTimetable(c *gin.Context) {
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

	// Need to be technician to manage another's timetable.
	if uint(uid) != s.UserID && !ddat.User.Can(data.CapManageTimetable) {
		c.String(http.StatusForbidden, "Permission Denied")
		return
	}

	usr, err := data.GetUser(Database, uint(uid))
	if err != nil {
		c.String(http.StatusNotFound, "User not Found")
		return
	}

	var iusr *isams.User
	var iusrs []isams.User
	if Config.HasISAMS() {
		if usr.IsamsID != nil {
			// NOTE: deliberately ignoring error here to use nil as a
			// sentinel. very naughty!
			iusr, _ = ISAMS.FindUser(*usr.IsamsID)
		}

		iusrs = ISAMS.Users
	}

	dat := struct {
		DashboardData
		TargetUser   data.User
		ISAMSEnabled bool
		ISAMSUser    *isams.User
		ISAMSUsers   []isams.User
	}{ddat, usr, Config.HasISAMS(), iusr, iusrs}

	c.HTML(http.StatusOK, "link.gohtml", dat)
}

// handleAccountUnlink is the handler for "/account/[ID]/unlink".
//
// Simply removes the iSAMS ID field from a user's database record. This is
// harder using GORM than I thought...
func handleAccountUnlink(c *gin.Context) {
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

	// Need to be technician to link another's timetable
	if uint(uid) != s.UserID && !ddat.User.Can(data.CapManageTimetable) {
		c.String(http.StatusForbidden, "Permission Denied")
		return
	}

	usr, err := data.GetUser(Database, uint(uid))
	if err != nil {
		c.String(http.StatusNotFound, "User not Found")
		return
	}

	usr.IsamsID = nil
	if err := Database.Model(&usr).Where(&usr).Update("isams_id", nil).Error; err != nil {
		internalError(c, err)
		return
	}

	c.Redirect(http.StatusFound, "/account/"+suid+"/timetable")
}

// handleAccountLink is the handler for "/account/[ID]/link".
//
// One GET parameter is expected containing the iSAMS UserCode which can be
// added to the user's account.
func handleAccountLink(c *gin.Context) {
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

	// Need to be technician to link another's timetable
	if uint(uid) != s.UserID && !ddat.User.Can(data.CapManageTimetable) {
		c.String(http.StatusForbidden, "Permission Denied")
		return
	}

	usr, err := data.GetUser(Database, uint(uid))
	if err != nil {
		c.String(http.StatusNotFound, "User not Found")
		return
	}

	if usr.IsamsID != nil {
		c.String(http.StatusForbidden, "User already has iSAMS ID. Must unlink existing first!")
		return
	}

	id, ok := c.GetQuery("id")
	if !ok {
		c.String(http.StatusBadRequest, "Expected ID parameter")
		return
	}

	iusr, err := ISAMS.FindUser(id)
	if err != nil {
		c.String(http.StatusNotFound, "No such iSAMS user")
		return
	}

	usr.IsamsID = &iusr.UserCode
	if err = Database.Updates(&usr).Error; err != nil {
		internalError(c, err)
		return
	}

	c.Redirect(http.StatusFound, "/account/"+suid+"/timetable")
}
