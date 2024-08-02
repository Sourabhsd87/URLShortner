package routes

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"
	"urlshortner/database"
	"urlshortner/helpers"

	"github.com/asaskevich/govalidator"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"

	"github.com/gin-gonic/gin"
)

type request struct {
	URL         string        `json:"url"`
	Customshort string        `json:"short"`
	Expiry      time.Duration `json:"expiry"`
}

type response struct {
	URL             string        `json:"url"`
	Customshort     string        `json:"short"`
	Expiry          time.Duration `json:"expiry"`
	XRateRemaining  int           `json:"rate_limit"`
	XRateLimitReset time.Duration `json:"rate_limit_reset"`
}

func ShortenURL(c *gin.Context) {

	var body request
	err := c.ShouldBindJSON(&body)
	if err != nil {
		c.JSON(http.StatusBadRequest, map[string]string{"message": "error  binding data"})
	}

	//implement rate limiting
	r2 := database.CreateClient(1)
	defer r2.Close()
	fmt.Println("---IP: ", c.ClientIP(), "---")
	val, err := r2.Get(database.Ctx, c.ClientIP()).Result()
	if err == redis.Nil {
		_ = r2.Set(database.Ctx, c.ClientIP(), os.Getenv("API_QUOTA"), 30*time.Minute).Err()
	} else {
		// value, _ := r2.Get(database.Ctx, c.ClientIP()).Result()
		intval, _ := strconv.Atoi(val)
		if intval <= 0 {
			limit, _ := r2.TTL(database.Ctx, c.ClientIP()).Result()
			c.JSON(http.StatusServiceUnavailable, map[string]interface{}{"error": "rate limit exceeded", "rate_limit_reset": (limit / time.Nanosecond) / time.Minute})
		}
	}

	//check if input is an actual URL
	if !govalidator.IsURL(body.URL) {
		c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid URL"})
	}

	//Check for domain error
	if helpers.RemoveDomainError(body.URL) {
		c.JSON(http.StatusServiceUnavailable, map[string]string{"message": "Domain error"})
	}

	//enforce hhtps,SSL
	body.URL = helpers.EnforceHTTP(body.URL)

	var id string
	if body.Customshort == "" {
		id = uuid.New().String()[:6]
	} else {
		id = body.Customshort
	}

	r := database.CreateClient(0)
	defer r.Close()

	value, _ := r.Get(database.Ctx, id).Result()
	if value != "" {
		c.JSON(http.StatusForbidden, map[string]interface{}{"error": "URL custon short is already in use"})
	}

	if body.Expiry == 0 {
		body.Expiry = 24
	}

	err = r.Set(database.Ctx, id, body.URL, body.Expiry*3600*time.Second).Err()
	if err != nil {
		c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": "unable to connect server"})
		return
	}

	resp := response{
		URL:             body.URL,
		Customshort:     "",
		Expiry:          body.Expiry,
		XRateRemaining:  10,
		XRateLimitReset: 30,
	}

	r2.Decr(database.Ctx, c.ClientIP())

	val, _ = r2.Get(database.Ctx, c.ClientIP()).Result()
	resp.XRateRemaining, _ = strconv.Atoi(val)

	ttl, _ := r2.TTL(database.Ctx, c.ClientIP()).Result()
	resp.XRateLimitReset = (ttl / time.Nanosecond) / time.Minute

	resp.Customshort = os.Getenv("DOMAIN") + "/" + id
	c.JSON(http.StatusOK, map[string]interface{}{"data": resp})
}
