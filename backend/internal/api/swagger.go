package api

import (
	"net/http"
	"os"
	"path/filepath"

	httpSwagger "github.com/swaggo/http-swagger"
)

// SwaggerHandler serves the Swagger UI.
func SwaggerHandler() http.Handler {
	// Configure Swagger UI to load the spec from /swagger.yaml endpoint
	return httpSwagger.Handler(
		httpSwagger.URL("/swagger.yaml"),
		httpSwagger.DeepLinking(true),
		httpSwagger.DocExpansion("list"),
		httpSwagger.DomID("swagger-ui"),
	)
}

// getSwaggerFilePath returns the path to the swagger.yaml file.
func getSwaggerFilePath() string {
	// Try different possible locations
	paths := []string{
		"./docs/swagger.yaml",
		"../docs/swagger.yaml",
		"../../docs/swagger.yaml",
		"/etc/db-backup/docs/swagger.yaml",
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	// Default to docs/swagger.yaml
	return "./docs/swagger.yaml"
}

// SwaggerFileHandler serves the swagger.yaml file.
func SwaggerFileHandler(w http.ResponseWriter, r *http.Request) {
	swaggerFile := getSwaggerFilePath()

	// Check if file exists
	if _, err := os.Stat(swaggerFile); os.IsNotExist(err) {
		http.Error(w, "Swagger specification not found", http.StatusNotFound)
		return
	}

	// Serve the file
	http.ServeFile(w, r, swaggerFile)
}

// SetupSwaggerRoutes adds Swagger routes to the router.
func SetupSwaggerRoutes(mux *http.ServeMux) {
	// Serve Swagger UI
	mux.Handle("/swagger/", httpSwagger.Handler(
		httpSwagger.URL("/swagger.yaml"),
	))

	// Serve the swagger.yaml file
	mux.HandleFunc("/swagger.yaml", SwaggerFileHandler)

	// Serve OpenAPI JSON (alternative format)
	mux.HandleFunc("/openapi.json", func(w http.ResponseWriter, r *http.Request) {
		// For now, redirect to YAML
		// In the future, you could convert YAML to JSON
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotImplemented)
		if _, err := w.Write([]byte(`{"error":"OpenAPI JSON endpoint not yet implemented. Please use /swagger.yaml"}`)); err != nil {
			return
		}
	})
}

// GetSwaggerSpec reads and returns the Swagger specification.
func GetSwaggerSpec() ([]byte, error) {
	swaggerFile := getSwaggerFilePath()
	absPath, err := filepath.Abs(swaggerFile)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, err
	}

	return data, nil
}
