package main

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/ejv2/prepper/data"
	"github.com/ejv2/prepper/notifications"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type formattedNotification struct {
	notifications.Notification
	FmtTime string `json:"fmt_time"`
}

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

// handleAPICreateItem is the handler for "/api/item/create"
//
// Returns the JSON-encoded new item details.
func handleAPICreateItem(c *gin.Context) {
	s := Sessions.Start(c)
	defer s.Update()

	us, err := data.GetUser(Database, s.UserID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "Access Denied",
			"message": "Authentication Failure",
		})
		return
	}

	if !us.Can(data.CapManageInventory) {
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "Access Denied",
			"message": "Insufficient Privilege Level",
		})
		return
	}

	dat := data.EquipmentItem{Model: &gorm.Model{}}
	err = c.BindJSON(&dat)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Body Syntax",
			"message": "Malformed JSON: " + err.Error(),
		})
		return
	}

	if dat.ID != 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid Item Specification",
			"message": "Refusing to create with specific ID",
		})
		return
	}

	if err := Database.Create(&dat).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Database Server Error",
			"message": "Database SQL Error: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dat)
}

// handleAPIBadEditItem is the handler for "/api/item/edit".
//
// This route is designed to catch bad requests from incorrectly created items
// on the client.
func handleAPIBadEditItem(c *gin.Context) {
	c.JSON(http.StatusBadRequest, gin.H{
		"error":   "Bad Request",
		"message": "Incorrect URL format (need item ID to edit): want /api/item/[ID]/edit",
	})
}

// handleAPICreateItem is the handler for "/api/item/[ID]/edit"
//
// Returns the JSON-encoded new item details.
func handleAPIEditItem(c *gin.Context) {
	s := Sessions.Start(c)
	defer s.Update()

	us, err := data.GetUser(Database, s.UserID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "Access Denied",
			"message": "Authentication Failure",
		})
		return
	}

	if !us.Can(data.CapManageInventory) {
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "Access Denied",
			"message": "Insufficient Privilege Level",
		})
		return
	}

	suid := c.Param("id")
	luid, err := strconv.ParseUint(suid, 10, 32)
	id := uint(luid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Invalid Item ID",
			"message": "Item ID Parse Error: " + err.Error(),
		})
		return
	}

	i, err := data.GetEquipmentItem(Database, id)
	oldid := i.ID
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Item Not Found",
			"message": "Item " + strconv.FormatUint(uint64(id), 10) + " does not exist",
		})
	}

	err = c.BindJSON(&i)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Body Syntax",
			"message": "Malformed JSON: " + err.Error(),
		})
		return
	}

	// Enforce that IDs are read only
	if oldid != i.ID {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid Item Specification",
			"message": "Item ID may not be modified",
		})
		return
	}

	log.Printf("user %s (%d) updates item ID %d: new record: %v", us.DisplayName(), us.ID, id, i)

	err = Database.Updates(&i).
		Update("hazard_voltage", i.HazardVoltage).
		Update("hazard_lazer", i.HazardLazer).
		Update("hazard_toxic", i.HazardToxic).
		Update("hazard_misc", i.HazardMisc).
		Update("available", i.Available).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Database Server Error",
			"message": "Database SQL Error: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, i)
}

// handleAPIDashboard is the handler for "/api/dashboard".
func handleAPIDashboard(c *gin.Context) {
	s := Sessions.Start(c)
	defer s.Update()

	if !s.SignedIn {
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "Access Denied",
			"message": "Please authenticate first",
		})
		return
	}

	// Pull out maximum of 5 notifications, or the length of the queue (whichever is lower)
	a := make([]formattedNotification, 0, 5)
	for i := 0; i < 5; i++ {
		n, err := Notifications.PopUser(s.UserID)
		if err != nil {
			break
		}

		a = append(a, formattedNotification{n, n.Time.Format(time.Kitchen)})
	}

	c.JSON(http.StatusOK, gin.H{
		"time":          time.Now().Format(time.Kitchen),
		"notifications": a,
	})
}

func handleAPIPeriod(c *gin.Context) {
	s := Sessions.Start(c)
	defer s.Update()

	if !s.SignedIn {
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "Access Denied",
			"message": "Please authenticate first",
		})
		return
	}

	ts, ok := c.GetQuery("time")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Need a time parameter",
		})
	}

	t, err := time.Parse("15:04", ts)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Bad time format: " + err.Error(),
		})
	}

	p := Config.TimetableLayout.FindPeriod(t)
	if p == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Not Found",
			"message": "No matching period",
		})
	}

	c.JSON(http.StatusOK, p)
}
