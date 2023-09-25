package main

import (
	"net/http"
	"strconv"

	"github.com/ejv2/prepper/data"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// handleAPIRoot is the handler for "/api/".
//
// Returns a bad usage error and some help for whoever is posting here for some
// reason.
func handleAPIRoot(c *gin.Context) {
	c.JSON(http.StatusNotFound, gin.H{
		"success": false,
		"error":   "This is the Prepper API endpoint URL. This is the root and does not accept any data. Please request a specific endpoint.",
	})
}

// handleAPIEditUser is the handler for POST @ "/api/user/edit/[ID]".
//
// NOTE: This endpoint is used separately for the creation of users.
func handleAPIEditUser(c *gin.Context) {
	s := Sessions.Start(c)
	defer s.Update()

	if !s.SignedIn {
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "Access Denied",
			"message": "Please authenticate first",
		})
		return
	}

	us, err := data.GetUser(Database, s.UserID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "Access Denied",
			"message": "Authentication Failure",
		})
		return
	}

	suid := c.Param("id")
	uid, err := strconv.ParseUint(suid, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Malformed User ID",
		})
		return
	}

	if uint(uid) != s.UserID && !us.Can(data.CapManageUsers) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Access Denied",
			"message": "Insufficient Privilege Level",
		})
		return
	}

	u := struct {
		data.User
		PostPassword string `json:"password"`
	}{
		data.User{Model: &gorm.Model{ID: uint(uid)}},
		"",
	}

	if err = Database.Find(&u.User).First(&u.User).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Database Error",
			"message": "Internal Database Error" + err.Error(),
		})
		return
	}

	err = c.BindJSON(&u)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Body Syntax",
			"message": "Malformed JSON: " + err.Error(),
		})
		return
	}

	if u.PostPassword != "" && us.Can(data.CapResetPassword) {
		if err = u.SetPassword(u.PostPassword); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Bad Password",
				"message": "Password does not meet minimum complexity requirements",
			})
			return
		}
	}

	if err = Database.Updates(&u.User).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Database Server Error",
			"message": "Database SQL Error: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, u.User)
}
