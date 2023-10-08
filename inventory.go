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
	fmt.Fprintln(c.Writer, "---BEGIN INVENTORY REPORT---")

	eq, err := data.GetEquipment(Database)
	if err != nil {
		internalError(c, err)
		return
	}

	for i, e := range eq {
		b, err := e.DailyBookings(time.Now())
		if err != nil {
			internalError(c, err)
			return
		}
		u, err := e.DailyUsage(time.Now())
		if err != nil {
			internalError(c, err)
			return
		}

		fmt.Fprintf(c.Writer, "#%d\t%s -- %d in use by %d bookings, %d in total\n", i, e.Name, u, len(b), e.Quantity)
	}

	c.String(http.StatusOK, "---END INVENTORY REPORT---")
}
