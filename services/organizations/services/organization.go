package services

import (
	"zori/internal/ctx"
	"zori/internal/storage/postgres/models"
	"zori/services/organizations/data"
)

type OrganizationService struct {
	data *data.OrganizationData
}

func NewOrganizationService(data *data.OrganizationData) *OrganizationService {
	return &OrganizationService{
		data: data,
	}
}

func (s *OrganizationService) GetOrganization(c *ctx.Ctx) (*models.Organization, error) {
	orgId := c.OrgID()
	return s.GetOrganizationByID(c, orgId)
}

func (s *OrganizationService) GetOrganizationByID(c *ctx.Ctx, id string) (*models.Organization, error) {
	return s.data.GetOrganizationByID(c, id)
}
