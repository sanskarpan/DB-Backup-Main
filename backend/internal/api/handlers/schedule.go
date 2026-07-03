package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/sanskarpan/db-backup/internal/scheduler"
)

type ScheduleHandler struct {
	scheduler *scheduler.Scheduler
}

func NewScheduleHandler(sched *scheduler.Scheduler) *ScheduleHandler {
	return &ScheduleHandler{
		scheduler: sched,
	}
}

// HandleListSchedules lists all schedules.
func (h *ScheduleHandler) HandleListSchedules(c *gin.Context) {
	schedules := h.scheduler.ListJobs()
	c.JSON(http.StatusOK, gin.H{
		"schedules": schedules,
		"total":     len(schedules),
	})
}

// HandleCreateSchedule creates a new schedule.
func (h *ScheduleHandler) HandleCreateSchedule(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Schedule creation via API not yet implemented"})
}

// HandleGetSchedule retrieves a schedule by ID.
func (h *ScheduleHandler) HandleGetSchedule(c *gin.Context) {
	id := c.Param("id")

	job, err := h.scheduler.GetJob(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Schedule not found"})
		return
	}

	c.JSON(http.StatusOK, job)
}

// HandleUpdateSchedule updates a schedule.
func (h *ScheduleHandler) HandleUpdateSchedule(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Schedule update via API not yet implemented"})
}

// HandleDeleteSchedule deletes a schedule.
func (h *ScheduleHandler) HandleDeleteSchedule(c *gin.Context) {
	id := c.Param("id")

	if err := h.scheduler.RemoveJob(id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Schedule not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Schedule deleted successfully"})
}

// HandleEnableSchedule enables a schedule.
func (h *ScheduleHandler) HandleEnableSchedule(c *gin.Context) {
	id := c.Param("id")

	if err := h.scheduler.EnableJob(id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Schedule not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Schedule enabled"})
}

// HandleDisableSchedule disables a schedule.
func (h *ScheduleHandler) HandleDisableSchedule(c *gin.Context) {
	id := c.Param("id")

	if err := h.scheduler.DisableJob(id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Schedule not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Schedule disabled"})
}

// HandleRunSchedule manually triggers a schedule.
func (h *ScheduleHandler) HandleRunSchedule(c *gin.Context) {
	id := c.Param("id")

	result, err := h.scheduler.RunJobNow(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Schedule not found or execution failed", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Schedule triggered successfully", "result": result})
}
