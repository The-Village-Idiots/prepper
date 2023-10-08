package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/ejv2/prepper/data"
	"github.com/gin-gonic/gin"
)

// handleInventory is the handler for "/inventory/".
func handleInventory(c *gin.Context) {
	eq, err := data.GetEquipment(Database)
	if err != nil {
		internalError(c, err)
		return
	}

	for i, e := range eq {
		b, err := e.Bookings(time.Now(), time.Now().Add(24*time.Hour))
		if err != nil {
			internalError(c, err)
			return
		}
		u, err := e.DailyUsage(time.Now())
		if err != nil {
			internalError(c, err)
			return
		}

		fmt.Fprintf(c.Writer, "#%d\t%v %v [IN TOTAL TODAY: %d]\n", i, e, b, u)
	}

	c.String(http.StatusOK, "---END INVENTORY REPORT---")
}
