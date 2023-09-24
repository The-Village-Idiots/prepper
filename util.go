package main

import "github.com/gin-gonic/gin"

// internalError prints an internal server error message for the user and gives
// instructions to report to the system administrator.
func internalError(c *gin.Context, err error) {
	c.String(500, "An internal system error has occurred. "+
		"Please report this issue (along with the error message) to your system administrator.\n\n"+
		"Error message: %s", err.Error())

	c.Abort()
}
