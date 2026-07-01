package views

import "github.com/gin-gonic/gin"

func Init(r *gin.Engine) {
	authGroup(r)
	userGroup(r)
	connectionGroup(r)
	coachGroup(r)
	rootGroup(r)
	exerciseGroup(r)
	plansGroup(r)
	planScheduleGroup(r)
	packagesGroup(r)
	testsGroup(r)
	achievementGroup(r)
	workoutsGroup(r)
	workoutLogGroup(r)
	tagGroup(r)
	feedsGroup(r)
	mediaGroup(r)
	messageGroup(r)
}
