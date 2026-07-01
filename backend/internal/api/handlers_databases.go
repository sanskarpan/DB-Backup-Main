package api

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sanskarpan/db-backup/internal/database"
	"github.com/sanskarpan/db-backup/internal/dbregistry"
)

// databaseStoreReady reports whether the registry store is configured, and if
// not, writes a 503 response.
func (s *Server) databaseStoreReady(c *gin.Context) bool {
	if s.dbStore == nil {
		s.respondError(c, http.StatusServiceUnavailable,
			errors.New("database registry is not configured"),
			"Database registry is unavailable")
		return false
	}
	return true
}

// handleListDatabases handles GET /api/v1/databases.
func (s *Server) handleListDatabases(c *gin.Context) {
	if !s.databaseStoreReady(c) {
		return
	}
	dbs, err := s.dbStore.List()
	if err != nil {
		s.respondError(c, http.StatusInternalServerError, err, "Failed to list databases")
		return
	}
	// Return the raw array to match the client contract.
	c.JSON(http.StatusOK, dbs)
}

// handleGetDatabase handles GET /api/v1/databases/:id.
func (s *Server) handleGetDatabase(c *gin.Context) {
	if !s.databaseStoreReady(c) {
		return
	}
	db, err := s.dbStore.Get(c.Param("id"))
	if err != nil {
		if errors.Is(err, dbregistry.ErrNotFound) {
			s.respondError(c, http.StatusNotFound, err, "Database not found")
			return
		}
		s.respondError(c, http.StatusInternalServerError, err, "Failed to get database")
		return
	}
	c.JSON(http.StatusOK, db)
}

// handleCreateDatabase handles POST /api/v1/databases.
func (s *Server) handleCreateDatabase(c *gin.Context) {
	if !s.databaseStoreReady(c) {
		return
	}
	var req dbregistry.CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.respondError(c, http.StatusBadRequest, err, "Invalid request body")
		return
	}
	db, err := s.dbStore.Create(&req)
	if err != nil {
		var verr *dbregistry.ValidationError
		if errors.As(err, &verr) {
			s.respondError(c, http.StatusBadRequest, err, "Validation failed")
			return
		}
		s.respondError(c, http.StatusInternalServerError, err, "Failed to create database")
		return
	}
	c.JSON(http.StatusCreated, db)
}

// handleUpdateDatabase handles PUT /api/v1/databases/:id.
func (s *Server) handleUpdateDatabase(c *gin.Context) {
	if !s.databaseStoreReady(c) {
		return
	}
	var req dbregistry.UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.respondError(c, http.StatusBadRequest, err, "Invalid request body")
		return
	}
	db, err := s.dbStore.Update(c.Param("id"), &req)
	if err != nil {
		var verr *dbregistry.ValidationError
		switch {
		case errors.Is(err, dbregistry.ErrNotFound):
			s.respondError(c, http.StatusNotFound, err, "Database not found")
		case errors.As(err, &verr):
			s.respondError(c, http.StatusBadRequest, err, "Validation failed")
		default:
			s.respondError(c, http.StatusInternalServerError, err, "Failed to update database")
		}
		return
	}
	c.JSON(http.StatusOK, db)
}

// handleDeleteDatabase handles DELETE /api/v1/databases/:id.
func (s *Server) handleDeleteDatabase(c *gin.Context) {
	if !s.databaseStoreReady(c) {
		return
	}
	if err := s.dbStore.Delete(c.Param("id")); err != nil {
		if errors.Is(err, dbregistry.ErrNotFound) {
			s.respondError(c, http.StatusNotFound, err, "Database not found")
			return
		}
		s.respondError(c, http.StatusInternalServerError, err, "Failed to delete database")
		return
	}
	c.Status(http.StatusNoContent)
}

// handleTestDatabase handles POST /api/v1/databases/:id/test. It performs a
// real connection attempt using the stored credentials and the driver
// registry, measuring latency. If no driver is registered for the type it
// returns success=false with an honest message (never a faked success).
func (s *Server) handleTestDatabase(c *gin.Context) {
	if !s.databaseStoreReady(c) {
		return
	}
	db, err := s.dbStore.Get(c.Param("id"))
	if err != nil {
		if errors.Is(err, dbregistry.ErrNotFound) {
			s.respondError(c, http.StatusNotFound, err, "Database not found")
			return
		}
		s.respondError(c, http.StatusInternalServerError, err, "Failed to get database")
		return
	}

	resp := s.testConnection(c.Request.Context(), db)
	c.JSON(http.StatusOK, resp)
}

// testConnection attempts to connect to the given database using the driver
// registry and returns a ConnectionTestResponse. It always closes the driver.
func (s *Server) testConnection(ctx context.Context, db *dbregistry.Database) dbregistry.ConnectionTestResponse {
	driver, err := database.CreateDriver(database.DatabaseType(db.Type))
	if err != nil {
		// No driver registered for this type — be honest, do not fake success.
		return dbregistry.ConnectionTestResponse{
			Success: false,
			Message: "connection test not supported for database type '" + db.Type + "': " + err.Error(),
		}
	}

	cfg := &database.ConnectionConfig{
		Type:              database.DatabaseType(db.Type),
		Host:              db.Host,
		Port:              db.Port,
		Username:          db.Username,
		Password:          db.Password,
		Database:          db.Database,
		SSLMode:           db.SSLMode,
		Options:           db.Options,
		ConnectionTimeout: 10 * time.Second,
	}

	testCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	start := time.Now()
	if err := driver.Connect(testCtx, cfg); err != nil {
		return dbregistry.ConnectionTestResponse{
			Success: false,
			Message: "connection failed: " + err.Error(),
		}
	}
	latency := float64(time.Since(start).Microseconds()) / 1000.0
	// Best-effort cleanup; the connection already succeeded so a disconnect
	// error does not change the test result, but we surface it if we can.
	if derr := driver.Disconnect(); derr != nil && s.logger != nil {
		s.logger.Warn("failed to disconnect after connection test", map[string]interface{}{"error": derr.Error()})
	}

	return dbregistry.ConnectionTestResponse{
		Success: true,
		Message: "connection successful",
		Latency: latency,
	}
}
