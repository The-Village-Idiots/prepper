package main

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/ejv2/prepper/data"
	"github.com/ejv2/prepper/session"
)

// handleActivities is the handler for "/activity/".
//
// Shows a table of registered activities, along with a search feature reused
// from the booking menu.
func handleActivities(c *gin.Context) {
	s := Sessions.Start(c)
	defer s.Update()

	ddat, err := NewDashboardData(s)
	if err != nil {
		internalError(c, err)
		return
	}

	act, err := data.GetPermanentActivities(Database)
	if err != nil {
		internalError(c, err)
		return
	}

	delname, deleted := c.GetQuery("deleted")
	_, saved := c.GetQuery("saved")

	dat := struct {
		DashboardData
		Activities  []data.Activity
		Deleted     bool
		DeletedName string
		Saved       bool
	}{ddat, act, deleted, delname, saved}

	c.HTML(http.StatusOK, "activities.gohtml", dat)
}

// activityEditor shows the activity editor HTML page.
func activityEditor(c *gin.Context, activity data.Activity, s session.Session) {
	ddat, err := NewDashboardData(s)
	if err != nil {
		internalError(c, err)
		return
	}

	eq, err := data.GetEquipment(Database)
	if err != nil {
		internalError(c, err)
		return
	}

	c.HTML(http.StatusOK, "activity-edit.gohtml", struct {
		DashboardData
		Activity  data.Activity
		Equipment []data.EquipmentItem
	}{ddat, activity, eq})
}

// handleActivityNew is the handler for "/activity/new".
func handleActivityNew(c *gin.Context) {
	s := Sessions.Start(c)
	defer s.Update()

	newact, err := data.NewActivity(Database, s.UserID)
	if err != nil {
		internalError(c, err)
		return
	}

	activityEditor(c, newact, s)
}

// handleActivityEdit is the handler for GET "/activity/[ID]/edit".
func handleActivityEdit(c *gin.Context) {
	s := Sessions.Start(c)
	defer s.Update()

	actsid := c.Param("activity")
	actid, err := strconv.ParseUint(actsid, 10, 32)
	if err != nil {
		c.String(http.StatusBadRequest, "Bad Activity ID")
		return
	}

	act, err := data.GetActivity(Database, uint(actid))
	if err != nil {
		c.String(http.StatusBadRequest, "Activity Not Found")
		return
	}

	activityEditor(c, act, s)
}

// handleActivityDoEdit is the handler for POST "/activity/[ID]/edit".
//
// This is the form handler for the edit form.
func handleActivityDoEdit(c *gin.Context) {
	actsid := c.Param("activity")
	actid, err := strconv.ParseUint(actsid, 10, 32)
	if err != nil {
		c.String(http.StatusBadRequest, "Bad Activity ID")
		return
	}

	act, err := data.GetActivity(Database, uint(actid))
	if err != nil {
		c.String(http.StatusNotFound, "Activity Not Found")
		return
	}

	c.MultipartForm()
	set, err := NewPostItemInformation(c.Request)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid item set")
		return
	}

	sub := struct {
		Title       string `form:"title"`
		Description string `form:"description"`
		Category    string `form:"category"`
	}{act.Title, act.Description, act.Category}
	err = c.Bind(&sub)
	if err != nil {
		c.String(http.StatusBadRequest, "Recieved bad data")
	}

	// Process equipment sets
	// We later use this to determine what to delete
	for i := range act.Equipment {
		act.Equipment[i].Quantity = 0
	}

	set.Copy(&act)
	act.Title = sub.Title
	act.Description = sub.Description
	act.Category = sub.Category
	if err := Database.Updates(&act).Error; err != nil {
		internalError(c, err)
		return
	}
	for _, eq := range act.Equipment {
		if eq.Quantity == 0 {
			if err := Database.Model(&eq).Where(&eq).Delete(&eq).Error; err != nil {
				internalError(c, err)
				return
			}

			continue
		}

		if err := Database.Updates(&eq).Error; err != nil {
			internalError(c, err)
			return
		}
	}

	c.Redirect(http.StatusFound, "/activity/?saved")
}

// handleActivityDelete is the handler for "/activity/[ID]/delete".
//
// This shows a confirmation screen given no query arguments, and performs the
// deletion if given them.
func handleActivityDelete(c *gin.Context) {
	s := Sessions.Start(c)
	defer s.Update()

	ddat, err := NewDashboardData(s)
	if err != nil {
		internalError(c, err)
		return
	}

	actsid := c.Param("activity")
	actid, err := strconv.ParseUint(actsid, 10, 32)
	if err != nil {
		c.String(http.StatusBadRequest, "Bad Activity ID")
		return
	}

	act, err := data.GetActivity(Database, uint(actid))
	if err != nil {
		c.String(http.StatusBadRequest, "Activity Not Found")
		return
	}

	// Actually do deletion when confirmed.
	_, conf := c.GetQuery("confirm")
	if conf {
		// If err isn't nil, GORM will rollback this transaction.
		err := Database.Transaction(func(tx *gorm.DB) error {
			acts := make([]data.Activity, 0, 10)
			if err := tx.Where(data.Activity{CopiedFrom: act.ID}).Find(&acts).Error; err != nil {
				internalError(c, err)
				return err
			}

			// Delete child activities and bookings.
			for _, a := range acts {
				// Delete equipment sets
				if err := tx.Model(data.EquipmentSet{}).Where(&data.EquipmentSet{ActivityID: a.ID}).Delete(&data.EquipmentSet{}).Error; err != nil {
					return err
				}

				if err := tx.Model(data.Booking{}).Where(&data.Booking{ActivityID: a.ID}).Delete(&data.Booking{}).Error; err != nil {
					return err
				}

				if err := tx.Delete(&a).Error; err != nil {
					return err
				}
			}

			if err := tx.Model(data.EquipmentSet{}).Where(&data.EquipmentSet{ActivityID: act.ID}).Delete(&data.EquipmentSet{}).Error; err != nil {
				return err
			}

			// Delete parent activity.
			if err := tx.Where(&act).Delete(&act).Error; err != nil {
				return err
			}

			return nil
		})

		if err != nil {
			internalError(c, err)
			return
		}

		c.Redirect(http.StatusFound, fmt.Sprint("/activity/?deleted=", url.QueryEscape(act.Title)))
		return
	}

	numbook := int64(0)
	if err := Database.Model(&data.Activity{}).Where(&data.Activity{CopiedFrom: act.ID}).Count(&numbook).Error; err != nil {
		internalError(c, err)
		return
	}

	c.HTML(http.StatusOK, "activity-delete.gohtml", struct {
		DashboardData
		data.Activity
		BookingCount int64
	}{ddat, act, numbook})
}
