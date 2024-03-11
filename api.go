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

type clashReference struct {
	EquipmentName   string `json:"equipment_name"`
	TotalQuantity   uint   `json:"total_quantity"`
	NetQuantity     int    `json:"net_quantity"`
	YouQuantity     uint   `json:"you_quantity"`
	ClashQuantity   uint   `json:"clash_quantity"`
	BookingID       uint   `json:"booking_id"`
	BookingUser     string `json:"booking_user"`
	BookingActivity string `json:"booking_activity"`
	BookingStarts   string `json:"booking_starts"`
	BookingEnds     string `json:"booking_ends"`
}

// handleAPIClashes is the handler for "/api/clashes".
//
// Returns a JSON array of all the clashes which are detected for the two query
// parameter datetimes.
func handleAPIClashes(c *gin.Context) {
	clashes := make([]clashReference, 0, 5)

	var err error
	var date time.Time

	_, manual := c.GetQuery("manual")
	if manual {
		sdate := c.Query("date")
		date, err = time.Parse(dateFormat, sdate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Bad Date Format",
				"message": err.Error(),
			})
			return
		}
	} else {
		wcp := c.Query("week_commencing")
		wstart, err := time.Parse(dateFormat, wcp)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Bad date format",
				"message": err.Error(),
			})
			return
		}
		wday := parseDay(c.Query("day"))

		date = weekCommencing(wstart).Truncate(24 * time.Hour).Add((24 * time.Hour) * time.Duration(wday))
	}

	sstime, setime := c.Query("start_time"), c.Query("end_time")
	stime, err := time.Parse(timeFormat, sstime)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad start time format",
			"message": err.Error(),
		})
		return
	}

	etime, err := time.Parse(timeFormat, setime)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad end time format",
			"message": err.Error(),
		})
		return
	}

	// Handle zone offsets as HTML does not supply them
	_, off := time.Now().Zone()
	stime = stime.Add(time.Duration(off) * -time.Second).In(time.Local).Add(2 * time.Minute)
	etime = etime.Add(time.Duration(off) * -time.Second).In(time.Local).Add(2 * time.Minute)

	// Start/end time is the date + the offset from the day boundary in stime/etime.
	start := date.Add(time.Hour * time.Duration(stime.Hour())).Add(time.Minute * time.Duration(stime.Minute()))
	end := date.Add(time.Hour * time.Duration(etime.Hour())).Add(time.Minute * time.Duration(etime.Minute()))

	set, err := NewItemInformation(c.Request)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Error",
			"message": err.Error(),
		})
		return
	}

	for _, i := range set {
		i.Item, err = data.GetEquipmentItem(Database, i.Item.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Internal Error",
				"message": err.Error(),
			})
			return
		}

		qty, err := i.Item.NetQuantity(start, end)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Internal Error",
				"message": "SQL Error" + err.Error(),
			})
			return
		}

		// Clash detected!
		if qty < 0 || uint(qty) < i.Quantity {
			if qty >= 0 {
				qty -= int(i.Quantity)
			}

			bks, err := i.Item.Bookings(start, end)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error":   "Internal Error",
					"message": "SQL Error" + err.Error(),
				})
				return
			}

			for _, b := range bks {
				cl := clashReference{
					EquipmentName:   i.Item.Name,
					TotalQuantity:   i.Item.Quantity,
					NetQuantity:     qty,
					YouQuantity:     i.Quantity,
					BookingID:       b.ID,
					BookingUser:     b.Owner.Username,
					BookingActivity: b.Activity.Parent(Database).Title,
					BookingStarts:   b.StartTime.Format(time.TimeOnly),
					BookingEnds:     b.EndTime.Format(time.TimeOnly),
				}

				for _, eq := range b.Activity.Equipment {
					if eq.ItemID == i.Item.ID {
						cl.ClashQuantity = eq.Quantity
					}
				}

				clashes = append(clashes, cl)
			}
		}
	}

	c.JSON(http.StatusOK, clashes)
}
