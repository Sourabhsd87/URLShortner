package routes

import (
	"net/http"
	"urlshortner/database"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

func ResolveURL(c *gin.Context) {
	url := c.Param("url")

	r := database.CreateClient(0)
	defer r.Close()

	value, err := r.Get(database.Ctx, url).Result()
	if err == redis.Nil {
		c.JSON(http.StatusNotFound, map[string]string{"error": "short not found"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, map[string]string{"error": "error connecting to database"})
		return
	}

	rIncr := database.CreateClient(1)
	defer rIncr.Close()

	_ = rIncr.Incr(database.Ctx, "counter")

	c.Redirect(301, value)
}
