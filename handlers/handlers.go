package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"coutoffer/models"
	"coutoffer/scheduler"
)

type Handler struct {
	db *gorm.DB
}

func New(db *gorm.DB) *Handler {
	return &Handler{db: db}
}

func (h *Handler) Index(c *gin.Context) {
	var subs []models.Subscription
	h.db.Where("active = ?", true).Find(&subs)
	c.HTML(http.StatusOK, "index.html", gin.H{
		"subscriptions": subs,
	})
}

func (h *Handler) Subscribe(c *gin.Context) {
	email := c.PostForm("email")
	company := c.PostForm("company")
	atsSlug := c.PostForm("ats_slug")

	if email == "" || company == "" || atsSlug == "" {
		var subs []models.Subscription
		h.db.Where("active = ?", true).Find(&subs)
		c.HTML(http.StatusBadRequest, "index.html", gin.H{
			"subscriptions": subs,
			"error":         "email, company, and ATS slug are required",
		})
		return
	}

	sub := models.Subscription{
		Email:   email,
		Company: company,
		ATSType: models.ATSType(c.PostForm("ats_type")),
		ATSSlug: atsSlug,
		Roles:   c.PostForm("roles"),
		Active:  true,
	}
	h.db.Create(&sub)
	c.Redirect(http.StatusSeeOther, "/")
}

func (h *Handler) Unsubscribe(c *gin.Context) {
	id := c.Param("id")
	h.db.Model(&models.Subscription{}).Where("id = ?", id).Update("active", false)
	c.Redirect(http.StatusSeeOther, "/")
}

func (h *Handler) ManualCheck(c *gin.Context) {
	go scheduler.CheckJobs(h.db)
	c.Redirect(http.StatusSeeOther, "/")
}
