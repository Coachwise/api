package views

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Version is the release this binary was built from. CI sets it from the git tag
// (-ldflags "-X coachwise/src/app/views.Version=1.2.0"); a local build says dev.
var Version = "dev"

func Init(r *gin.Engine) {

	r.GET("health", health)

	authGroup(r)
	userGroup(r)
	connectionGroup(r)
	coachGroup(r)
	rootGroup(r)
	exerciseGroup(r)
	plansGroup(r)
	planScheduleGroup(r)
	packagesGroup(r)
	billingGroup(r)
	walletGroup(r)
	testsGroup(r)
	achievementGroup(r)
	notificationGroup(r)
	workoutsGroup(r)
	workoutLogGroup(r)
	tagGroup(r)
	feedsGroup(r)
	mediaGroup(r)
	messageGroup(r)
}

func health(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"service":   "coachwise-api",
		"version":   Version,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}
