package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/pritunl/pritunl-zero/cookie"
	"github.com/pritunl/pritunl-zero/database"
	"github.com/pritunl/pritunl-zero/errortypes"
	"github.com/pritunl/pritunl-zero/session"
	"github.com/pritunl/pritunl-zero/settings"
	"github.com/pritunl/pritunl-zero/user"
	"gopkg.in/mgo.v2/bson"
)

type authStateData struct {
	Providers []*authStateProviderData `json:"providers"`
}

type authStateProviderData struct {
	Id    bson.ObjectId `json:"id"`
	Type  string        `json:"type"`
	Label string        `json:"label"`
}

func authStateGet(c *gin.Context) {
	data := &authStateData{
		Providers: []*authStateProviderData{},
	}

	for _, provider := range settings.Auth.Providers {
		providerData := &authStateProviderData{
			Id:    provider.Id,
			Type:  provider.Type,
			Label: provider.Label,
		}
		data.Providers = append(data.Providers, providerData)
	}

	c.JSON(200, data)
}

type authData struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func authSessionPost(c *gin.Context) {
	db := c.MustGet("db").(*database.Database)
	data := &authData{}

	err := c.Bind(data)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}

	usr, err := user.GetUsername(db, user.Local, data.Username)
	if err != nil {
		switch err.(type) {
		case *database.NotFoundError:
			c.JSON(401, &errortypes.ErrorData{
				Error:   "auth_invalid",
				Message: "Authencation credentials are invalid",
			})
			break
		default:
			c.AbortWithError(500, err)
		}
		return
	}

	valid := usr.CheckPassword(data.Password)
	if !valid {
		c.JSON(401, &errortypes.ErrorData{
			Error:   "auth_invalid",
			Message: "Authencation credentials are invalid",
		})
		return
	}

	if usr.Administrator != "super" {
		c.JSON(401, &errortypes.ErrorData{
			Error:   "unauthorized",
			Message: "Not authorized",
		})
		return
	}

	cook := cookie.New(c)

	_, err = cook.NewSession(db, usr.Id, true)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}

	c.Status(200)
}

func logoutGet(c *gin.Context) {
	db := c.MustGet("db").(*database.Database)
	sess := c.MustGet("session").(*session.Session)

	err := sess.Remove(db)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}

	c.Redirect(302, "/login")
}
