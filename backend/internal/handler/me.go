package handler

import (
    "net/http"

    "github.com/gin-gonic/gin"
)

// MeHandler returns the current authenticated user info populated by RequireAuth
func MeHandler() gin.HandlerFunc {
    return func(c *gin.Context) {
        user, ok := CurrentUser(c)
        if !ok {
            WriteError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing session")
            return
        }

        c.JSON(http.StatusOK, gin.H{"user": gin.H{"id": user.ID, "username": user.Username, "role": user.Role}})
    }
}
