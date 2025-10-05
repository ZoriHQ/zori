package projects_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"zori/di"
	"zori/internal/storage/postgres/models"
	"zori/services/auth/services"
	"zori/services/projects/types"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type ProjectTestResponse struct {
	*models.Project
}

type ListProjectsTestResponse struct {
	Projects []*models.Project `json:"projects"`
	Total    int               `json:"total"`
}

func setupTestUser(t *testing.T, tc *di.TestContainer) *services.AuthResponse {
	randomEmail := fmt.Sprintf("test-%d@example.com", time.Now().UnixNano())

	registerUser := services.RegisterRequest{
		Email:            randomEmail,
		Password:         "ValidPass123!",
		FirstName:        "Test",
		LastName:         "User",
		OrganizationName: "Test Organization",
	}

	reqBody, _ := json.Marshal(registerUser)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBuffer(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	tc.Server.Echo.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var authResponse services.AuthResponse
	err := json.Unmarshal(rec.Body.Bytes(), &authResponse)
	require.NoError(t, err)

	return &authResponse
}

func TestProjectService_CreateProject(t *testing.T) {
	tc := di.NewTestContainer(t)
	defer tc.Cleanup()

	authResponse := setupTestUser(t, tc)

	tests := []struct {
		name           string
		request        types.CreateProjectRequest
		expectedStatus int
		checkResponse  func(t *testing.T, response *ProjectTestResponse)
		expectError    bool
	}{
		{
			name: "successful project creation",
			request: types.CreateProjectRequest{
				Name:           "Test Project",
				WebsiteURL:     "https://example.com",
				AllowLocalHost: false,
			},
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, response *ProjectTestResponse) {
				assert.NotEmpty(t, response.ID)
				assert.Equal(t, "Test Project", response.Name)
				assert.Equal(t, "https://example.com", response.Domain)
				assert.False(t, response.AllowLocalHost)
				assert.NotEmpty(t, response.ProjectToken)
				assert.Equal(t, authResponse.Organization.ID, response.OrganizationID)
				assert.Nil(t, response.FirstEventReceivedAt)
			},
			expectError: false,
		},
		{
			name: "project with localhost allowed",
			request: types.CreateProjectRequest{
				Name:           "Localhost Project",
				WebsiteURL:     "http://localhost:3000",
				AllowLocalHost: true,
			},
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, response *ProjectTestResponse) {
				assert.Equal(t, "Localhost Project", response.Name)
				assert.Equal(t, "http://localhost:3000", response.Domain)
				assert.True(t, response.AllowLocalHost)
			},
			expectError: false,
		},
		{
			name: "missing project name",
			request: types.CreateProjectRequest{
				WebsiteURL:     "https://example.com",
				AllowLocalHost: false,
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name: "missing website URL",
			request: types.CreateProjectRequest{
				Name:           "Test Project",
				AllowLocalHost: false,
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name: "invalid website URL",
			request: types.CreateProjectRequest{
				Name:           "Test Project",
				WebsiteURL:     "not-a-valid-url",
				AllowLocalHost: false,
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqBody, err := json.Marshal(tt.request)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/projects", bytes.NewBuffer(reqBody))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			req.Header.Set("Authorization", "Bearer "+authResponse.AccessToken)
			rec := httptest.NewRecorder()

			tc.Server.Echo.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Logf("Unexpected status code. Response body: %s", rec.Body.String())
			}
			assert.Equal(t, tt.expectedStatus, rec.Code)

			if !tt.expectError {
				var response ProjectTestResponse
				err := json.Unmarshal(rec.Body.Bytes(), &response)
				require.NoError(t, err)

				if tt.checkResponse != nil {
					tt.checkResponse(t, &response)
				}

				ctx := context.Background()
				var project models.Project
				err = tc.DB.DB.NewSelect().
					Model(&project).
					Where("id = ?", response.ID).
					Scan(ctx)
				require.NoError(t, err)
				assert.Equal(t, tt.request.Name, project.Name)
				assert.Equal(t, tt.request.WebsiteURL, project.Domain)
				assert.Equal(t, tt.request.AllowLocalHost, project.AllowLocalHost)
				assert.Equal(t, authResponse.Organization.ID, project.OrganizationID)
			}
		})
	}
}

func TestProjectService_ListProjects(t *testing.T) {
	tc := di.NewTestContainer(t)
	defer tc.Cleanup()

	authResponse := setupTestUser(t, tc)

	projectRequests := []types.CreateProjectRequest{
		{
			Name:           "Project 1",
			WebsiteURL:     "https://project1.com",
			AllowLocalHost: false,
		},
		{
			Name:           "Project 2",
			WebsiteURL:     "https://project2.com",
			AllowLocalHost: true,
		},
	}

	createdProjects := make([]*ProjectTestResponse, 0, len(projectRequests))
	for _, projectReq := range projectRequests {
		reqBody, _ := json.Marshal(projectReq)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/projects", bytes.NewBuffer(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		req.Header.Set("Authorization", "Bearer "+authResponse.AccessToken)
		rec := httptest.NewRecorder()

		tc.Server.Echo.ServeHTTP(rec, req)
		require.Equal(t, http.StatusCreated, rec.Code)

		var response ProjectTestResponse
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)
		createdProjects = append(createdProjects, &response)
	}

	t.Run("list all projects", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/list", nil)
		req.Header.Set("Authorization", "Bearer "+authResponse.AccessToken)
		rec := httptest.NewRecorder()

		tc.Server.Echo.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var response ListProjectsTestResponse
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, 2, response.Total)
		assert.Len(t, response.Projects, 2)

		projectNames := make([]string, len(response.Projects))
		for i, project := range response.Projects {
			projectNames[i] = project.Name
		}
		assert.Contains(t, projectNames, "Project 1")
		assert.Contains(t, projectNames, "Project 2")
	})

	t.Run("unauthorized request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/list", nil)
		rec := httptest.NewRecorder()

		tc.Server.Echo.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})
}

func TestProjectService_GetProject(t *testing.T) {
	tc := di.NewTestContainer(t)
	defer tc.Cleanup()

	authResponse := setupTestUser(t, tc)

	projectReq := types.CreateProjectRequest{
		Name:           "Get Test Project",
		WebsiteURL:     "https://gettest.com",
		AllowLocalHost: false,
	}

	reqBody, _ := json.Marshal(projectReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects", bytes.NewBuffer(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("Authorization", "Bearer "+authResponse.AccessToken)
	rec := httptest.NewRecorder()

	tc.Server.Echo.ServeHTTP(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)

	var createdProject ProjectTestResponse
	err := json.Unmarshal(rec.Body.Bytes(), &createdProject)
	require.NoError(t, err)

	t.Run("get existing project", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+createdProject.ID, nil)
		req.Header.Set("Authorization", "Bearer "+authResponse.AccessToken)
		rec := httptest.NewRecorder()

		tc.Server.Echo.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var response ProjectTestResponse
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, createdProject.ID, response.ID)
		assert.Equal(t, "Get Test Project", response.Name)
		assert.Equal(t, "https://gettest.com", response.Domain)
		assert.False(t, response.AllowLocalHost)
	})

	t.Run("get non-existent project", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/non-existent-id", nil)
		req.Header.Set("Authorization", "Bearer "+authResponse.AccessToken)
		rec := httptest.NewRecorder()

		tc.Server.Echo.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("unauthorized request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+createdProject.ID, nil)
		rec := httptest.NewRecorder()

		tc.Server.Echo.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})
}

func TestProjectService_UpdateProject(t *testing.T) {
	tc := di.NewTestContainer(t)
	defer tc.Cleanup()

	authResponse := setupTestUser(t, tc)

	projectReq := types.CreateProjectRequest{
		Name:           "Update Test Project",
		WebsiteURL:     "https://updatetest.com",
		AllowLocalHost: false,
	}

	reqBody, _ := json.Marshal(projectReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects", bytes.NewBuffer(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("Authorization", "Bearer "+authResponse.AccessToken)
	rec := httptest.NewRecorder()

	tc.Server.Echo.ServeHTTP(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)

	var createdProject ProjectTestResponse
	err := json.Unmarshal(rec.Body.Bytes(), &createdProject)
	require.NoError(t, err)

	tests := []struct {
		name           string
		projectID      string
		request        types.UpdateProjectRequest
		expectedStatus int
		checkResponse  func(t *testing.T, response *ProjectTestResponse)
		expectError    bool
	}{
		{
			name:      "successful project update",
			projectID: createdProject.ID,
			request: types.UpdateProjectRequest{
				Name:           "Updated Project Name",
				WebsiteURL:     "https://updated.com",
				AllowLocalHost: true,
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, response *ProjectTestResponse) {
				assert.Equal(t, createdProject.ID, response.ID)
				assert.Equal(t, "Updated Project Name", response.Name)
				assert.Equal(t, "https://updated.com", response.Domain)
				assert.True(t, response.AllowLocalHost)
			},
			expectError: false,
		},
		{
			name:      "update non-existent project",
			projectID: "non-existent-id",
			request: types.UpdateProjectRequest{
				Name: "Should Not Work",
			},
			expectedStatus: http.StatusNotFound,
			expectError:    true,
		},
		{
			name:      "invalid website URL",
			projectID: createdProject.ID,
			request: types.UpdateProjectRequest{
				WebsiteURL: "not-a-valid-url",
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqBody, err := json.Marshal(tt.request)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPut, "/api/v1/projects/"+tt.projectID, bytes.NewBuffer(reqBody))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			req.Header.Set("Authorization", "Bearer "+authResponse.AccessToken)
			rec := httptest.NewRecorder()

			tc.Server.Echo.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Logf("Unexpected status code. Response body: %s", rec.Body.String())
			}
			assert.Equal(t, tt.expectedStatus, rec.Code)

			if !tt.expectError {
				var response ProjectTestResponse
				err := json.Unmarshal(rec.Body.Bytes(), &response)
				require.NoError(t, err)

				if tt.checkResponse != nil {
					tt.checkResponse(t, &response)
				}

				ctx := context.Background()
				var project models.Project
				err = tc.DB.DB.NewSelect().
					Model(&project).
					Where("id = ?", response.ID).
					Scan(ctx)
				require.NoError(t, err)
				assert.Equal(t, tt.request.Name, project.Name)
				if tt.request.WebsiteURL != "" {
					assert.Equal(t, tt.request.WebsiteURL, project.Domain)
				}
				assert.Equal(t, tt.request.AllowLocalHost, project.AllowLocalHost)
			}
		})
	}
}

func TestProjectService_DeleteProject(t *testing.T) {
	tc := di.NewTestContainer(t)
	defer tc.Cleanup()

	authResponse := setupTestUser(t, tc)

	projectReq := types.CreateProjectRequest{
		Name:           "Delete Test Project",
		WebsiteURL:     "https://deletetest.com",
		AllowLocalHost: false,
	}

	reqBody, _ := json.Marshal(projectReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects", bytes.NewBuffer(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("Authorization", "Bearer "+authResponse.AccessToken)
	rec := httptest.NewRecorder()

	tc.Server.Echo.ServeHTTP(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)

	var createdProject ProjectTestResponse
	err := json.Unmarshal(rec.Body.Bytes(), &createdProject)
	require.NoError(t, err)

	t.Run("successful project deletion", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/projects/"+createdProject.ID, nil)
		req.Header.Set("Authorization", "Bearer "+authResponse.AccessToken)
		rec := httptest.NewRecorder()

		tc.Server.Echo.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var response map[string]string
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "Project deleted successfully", response["message"])

		ctx := context.Background()
		var project models.Project
		err = tc.DB.DB.NewSelect().
			Model(&project).
			Where("id = ?", createdProject.ID).
			Scan(ctx)
		assert.Error(t, err)
	})

	t.Run("delete non-existent project", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/projects/non-existent-id", nil)
		req.Header.Set("Authorization", "Bearer "+authResponse.AccessToken)
		rec := httptest.NewRecorder()

		tc.Server.Echo.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("unauthorized request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/projects/some-id", nil)
		rec := httptest.NewRecorder()

		tc.Server.Echo.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})
}
