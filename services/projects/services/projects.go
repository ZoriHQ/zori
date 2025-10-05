package services

import (
	"marker/internal/ctx"
	"marker/services/projects/data"
)

type ProjectService struct {
	data *data.ProjectData
}

func NewProjectService(data *data.ProjectData) *ProjectService {
	return &ProjectService{data: data}
}

func (s *ProjectService) ListOrganizationProjects(c *ctx.Ctx) ([]*data.ProjectData, error) {
	return s.data.ListOrganizationProjects(c.OrgID())
}

func (s *ProjectService) ListProjects(c *ctx.Ctx) ([]*data.ProjectData, error) {
	projects, err := s.data.ListOrganizationProjects(c.OrgID())
	if err != nil {
		return nil, err
	}
	return projects, nil
}

func (s *ProjectService) CreateProject(c *ctx.Ctx) (*data.ProjectData, error) {
	return nil, nil
}
