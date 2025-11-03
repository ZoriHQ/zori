package projects_test

import (
	"context"
	"net/http/httptest"
	"testing"

	"zori/di"
	"zori/internal/ctx"
	"zori/internal/storage/postgres/models"
	"zori/services/projects/data"
	"zori/services/projects/services"
	"zori/services/projects/types"
	"zori/testutil/fixtures"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createMockContext(account *fixtures.AccountFixture) *ctx.Ctx {
	e := echo.New()
	req := httptest.NewRequest("POST", "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	mockCtx := ctx.NewCtx(c)
	mockCtx.SetUser(&models.Account{
		ID:    account.AccountID,
		Email: account.Email,
	})
	mockCtx.SetOrgID(account.OrgID)

	return mockCtx
}

func TestProjectService_CreateProject(t *testing.T) {
	tc := di.NewTestContainer(t)
	defer tc.Cleanup()

	account := fixtures.CreateAccount(t, tc)

	projectData := data.NewProjectData(tc.DB)
	mockCtx := createMockContext(account)

	tests := []struct {
		name          string
		request       types.CreateProjectRequest
		checkResponse func(t *testing.T, response *models.Project, err error)
	}{
		{
			name: "successful project creation",
			request: types.CreateProjectRequest{
				Name:           "Test Project",
				WebsiteURL:     "https://example.com",
				AllowLocalHost: false,
			},
			checkResponse: func(t *testing.T, response *models.Project, err error) {
				require.NoError(t, err)
				assert.NotEmpty(t, response.ID)
				assert.Equal(t, "Test Project", response.Name)
				assert.Equal(t, "https://example.com", response.Domain)
				assert.False(t, response.AllowLocalHost)
				assert.NotEmpty(t, response.ProjectToken)
				assert.Equal(t, account.OrgID, response.OrganizationID)
				assert.Nil(t, response.FirstEventReceivedAt)

				dbCtx := context.Background()
				var project models.Project
				err = tc.DB.DB.NewSelect().
					Model(&project).
					Where("id = ?", response.ID).
					Scan(dbCtx)
				require.NoError(t, err)
				assert.Equal(t, "Test Project", project.Name)
				assert.Equal(t, "https://example.com", project.Domain)
				assert.Equal(t, account.OrgID, project.OrganizationID)
			},
		},
		{
			name: "project with localhost allowed",
			request: types.CreateProjectRequest{
				Name:           "Localhost Project",
				WebsiteURL:     "http://localhost:3000",
				AllowLocalHost: true,
			},
			checkResponse: func(t *testing.T, response *models.Project, err error) {
				require.NoError(t, err)
				assert.Equal(t, "Localhost Project", response.Name)
				assert.Equal(t, "http://localhost:3000", response.Domain)
				assert.True(t, response.AllowLocalHost)
			},
		},
		{
			name: "missing project name",
			request: types.CreateProjectRequest{
				WebsiteURL:     "https://example.com",
				AllowLocalHost: false,
			},
			checkResponse: func(t *testing.T, response *models.Project, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "missing website URL",
			request: types.CreateProjectRequest{
				Name:           "Test Project",
				AllowLocalHost: false,
			},
			checkResponse: func(t *testing.T, response *models.Project, err error) {
				require.NoError(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response, err := projectData.CreateProject(mockCtx, &tt.request)
			tt.checkResponse(t, response, err)
		})
	}
}

func TestProjectService_ListProjects(t *testing.T) {
	tc := di.NewTestContainer(t)
	defer tc.Cleanup()

	account := fixtures.CreateAccount(t, tc)

	projectData := data.NewProjectData(tc.DB)
	projectService := services.NewProjectService(projectData)

	mockCtx := createMockContext(account)

	project1, err := projectData.CreateProject(mockCtx, &types.CreateProjectRequest{
		Name:           "Project 1",
		WebsiteURL:     "https://project1.com",
		AllowLocalHost: false,
	})
	require.NoError(t, err)

	project2, err := projectData.CreateProject(mockCtx, &types.CreateProjectRequest{
		Name:           "Project 2",
		WebsiteURL:     "https://project2.com",
		AllowLocalHost: true,
	})
	require.NoError(t, err)

	t.Run("list all projects", func(t *testing.T) {
		response, err := projectService.ListProjects(mockCtx)

		require.NoError(t, err)
		assert.Equal(t, 2, response.Total)
		assert.Len(t, response.Projects, 2)

		projectNames := make([]string, len(response.Projects))
		for i, project := range response.Projects {
			projectNames[i] = project.Name
		}
		assert.Contains(t, projectNames, "Project 1")
		assert.Contains(t, projectNames, "Project 2")

		for _, project := range response.Projects {
			assert.Equal(t, account.OrgID, project.OrganizationID)
		}
	})

	t.Run("list projects for different org returns empty", func(t *testing.T) {
		differentOrgAccount := &fixtures.AccountFixture{
			AccountID: account.AccountID,
			Email:     account.Email,
			OrgID:     uuid.New().String(),
		}
		differentOrgCtx := createMockContext(differentOrgAccount)

		response, err := projectService.ListProjects(differentOrgCtx)

		require.NoError(t, err)
		assert.Equal(t, 0, response.Total)
		assert.Len(t, response.Projects, 0)
	})

	_, _ = projectData.DeleteProject(mockCtx, project1.ID)
	_, _ = projectData.DeleteProject(mockCtx, project2.ID)
}

func TestProjectService_GetProject(t *testing.T) {
	tc := di.NewTestContainer(t)
	defer tc.Cleanup()

	account := fixtures.CreateAccount(t, tc)

	projectData := data.NewProjectData(tc.DB)
	projectService := services.NewProjectService(projectData)

	mockCtx := createMockContext(account)

	createdProject, err := projectData.CreateProject(mockCtx, &types.CreateProjectRequest{
		Name:           "Get Test Project",
		WebsiteURL:     "https://gettest.com",
		AllowLocalHost: false,
	})
	require.NoError(t, err)

	t.Run("get existing project", func(t *testing.T) {
		response, err := projectService.GetProjectNoContext(context.Background(), createdProject.ID, account.OrgID)

		require.NoError(t, err)
		assert.Equal(t, createdProject.ID, response.ID)
		assert.Equal(t, "Get Test Project", response.Name)
		assert.Equal(t, "https://gettest.com", response.Domain)
		assert.False(t, response.AllowLocalHost)
	})

	t.Run("get non-existent project", func(t *testing.T) {
		nonExistentID := "00000000-0000-0000-0000-000000000000"
		response, err := projectService.GetProjectNoContext(context.Background(), nonExistentID, account.OrgID)

		require.Error(t, err)
		assert.Nil(t, response)
	})

	t.Run("get project with wrong org ID", func(t *testing.T) {
		wrongOrgID := uuid.New().String()
		response, err := projectService.GetProjectNoContext(context.Background(), createdProject.ID, wrongOrgID)

		require.Error(t, err)
		assert.Nil(t, response)
	})

	_, _ = projectData.DeleteProject(mockCtx, createdProject.ID)
}

func TestProjectService_UpdateProject(t *testing.T) {
	tc := di.NewTestContainer(t)
	defer tc.Cleanup()

	account := fixtures.CreateAccount(t, tc)

	projectData := data.NewProjectData(tc.DB)

	mockCtx := createMockContext(account)

	createdProject, err := projectData.CreateProject(mockCtx, &types.CreateProjectRequest{
		Name:           "Update Test Project",
		WebsiteURL:     "https://updatetest.com",
		AllowLocalHost: false,
	})
	require.NoError(t, err)

	tests := []struct {
		name          string
		projectID     string
		orgID         string
		request       types.UpdateProjectRequest
		checkResponse func(t *testing.T, response *services.ProjectResponse, err error)
	}{
		{
			name:      "successful project update",
			projectID: createdProject.ID,
			orgID:     account.OrgID,
			request: types.UpdateProjectRequest{
				Name:           "Updated Project Name",
				WebsiteURL:     "https://updated.com",
				AllowLocalHost: true,
			},
			checkResponse: func(t *testing.T, response *services.ProjectResponse, err error) {
				require.NoError(t, err)
				assert.Equal(t, createdProject.ID, response.ID)
				assert.Equal(t, "Updated Project Name", response.Name)
				assert.Equal(t, "https://updated.com", response.Domain)
				assert.True(t, response.AllowLocalHost)

				dbCtx := context.Background()
				var project models.Project
				err = tc.DB.DB.NewSelect().
					Model(&project).
					Where("id = ?", response.ID).
					Scan(dbCtx)
				require.NoError(t, err)
				assert.Equal(t, "Updated Project Name", project.Name)
			},
		},
		{
			name:      "update non-existent project",
			projectID: "00000000-0000-0000-0000-000000000000",
			orgID:     account.OrgID,
			request: types.UpdateProjectRequest{
				Name: "Should Not Work",
			},
			checkResponse: func(t *testing.T, response *services.ProjectResponse, err error) {
				require.Error(t, err)
				assert.Nil(t, response)
			},
		},
		{
			name:      "update project with wrong org ID",
			projectID: createdProject.ID,
			orgID:     uuid.New().String(),
			request: types.UpdateProjectRequest{
				Name: "Should Not Work",
			},
			checkResponse: func(t *testing.T, response *services.ProjectResponse, err error) {
				require.Error(t, err)
				assert.Nil(t, response)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exists, err := projectData.ProjectExists(context.Background(), tt.projectID, tt.orgID)
			if err != nil {
				tt.checkResponse(t, nil, err)
				return
			}
			if !exists {
				tt.checkResponse(t, nil, echo.NewHTTPError(404, "Project not found"))
				return
			}

			response, err := projectData.UpdateProject(context.Background(), tt.projectID, tt.orgID, &tt.request)
			if err != nil {
				tt.checkResponse(t, nil, err)
				return
			}

			tt.checkResponse(t, &services.ProjectResponse{Project: response}, nil)
		})
	}

	_, _ = projectData.DeleteProject(mockCtx, createdProject.ID)
}

func TestProjectService_DeleteProject(t *testing.T) {
	tc := di.NewTestContainer(t)
	defer tc.Cleanup()

	account := fixtures.CreateAccount(t, tc)

	projectData := data.NewProjectData(tc.DB)

	mockCtx := createMockContext(account)

	createdProject, err := projectData.CreateProject(mockCtx, &types.CreateProjectRequest{
		Name:           "Delete Test Project",
		WebsiteURL:     "https://deletetest.com",
		AllowLocalHost: false,
	})
	require.NoError(t, err)

	t.Run("successful project deletion", func(t *testing.T) {
		rowsAffected, err := projectData.DeleteProject(mockCtx, createdProject.ID)

		require.NoError(t, err)
		assert.Equal(t, int64(1), rowsAffected)

		dbCtx := context.Background()
		var project models.Project
		err = tc.DB.DB.NewSelect().
			Model(&project).
			Where("id = ?", createdProject.ID).
			Scan(dbCtx)
		assert.Error(t, err)
	})

	t.Run("delete non-existent project", func(t *testing.T) {
		nonExistentID := "00000000-0000-0000-0000-000000000000"
		rowsAffected, err := projectData.DeleteProject(mockCtx, nonExistentID)

		require.NoError(t, err)
		assert.Equal(t, int64(0), rowsAffected)
	})

	t.Run("delete project with wrong org ID", func(t *testing.T) {
		newProject, err := projectData.CreateProject(mockCtx, &types.CreateProjectRequest{
			Name:           "Another Delete Test",
			WebsiteURL:     "https://deletetest2.com",
			AllowLocalHost: false,
		})
		require.NoError(t, err)

		wrongOrgAccount := &fixtures.AccountFixture{
			AccountID: account.AccountID,
			Email:     account.Email,
			OrgID:     uuid.New().String(),
		}
		wrongOrgCtx := createMockContext(wrongOrgAccount)

		rowsAffected, err := projectData.DeleteProject(wrongOrgCtx, newProject.ID)

		require.NoError(t, err)
		assert.Equal(t, int64(0), rowsAffected)

		exists, err := projectData.ProjectExists(context.Background(), newProject.ID, account.OrgID)
		require.NoError(t, err)
		assert.True(t, exists)

		_, _ = projectData.DeleteProject(mockCtx, newProject.ID)
	})
}
