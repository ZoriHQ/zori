package services

import (
	"context"
	"fmt"
	"net/http"
	"time"
	"zori/internal/config"
	"zori/internal/ctx"
	"zori/internal/storage/postgres/models"
	"zori/internal/utils"
	"zori/services/payments/data"
	"zori/services/payments/types"
	projectsData "zori/services/projects/data"

	"github.com/labstack/echo/v4"
	"github.com/stripe/stripe-go/v82"
)

type ProviderManager struct {
	data            *data.PaymentProviderData
	projectData     *projectsData.ProjectData
	encryptor       *utils.Encryptor
	backfillService *BackfillService
	config          *config.Config
}

func NewProviderManager(
	data *data.PaymentProviderData,
	projectData *projectsData.ProjectData,
	encryptor *utils.Encryptor,
	backfillService *BackfillService,
	cfg *config.Config,
) *ProviderManager {
	return &ProviderManager{
		data:            data,
		projectData:     projectData,
		encryptor:       encryptor,
		backfillService: backfillService,
		config:          cfg,
	}
}

// @Summary Create a new payment provider
// @Description Connect a new payment provider to a project
// @Tags Payment Providers
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param request body types.CreatePaymentProviderRequest true "Payment provider details"
// @Success 201 {object} types.PaymentProviderResponse "Created payment provider"
// @Failure 400 {object} map[string]interface{} "Invalid request or validation failed"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 404 {object} map[string]interface{} "Project not found"
// @Failure 409 {object} map[string]interface{} "Provider already exists for this project"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/payment-providers [post]
func (pm *ProviderManager) CreateProvider(c *ctx.Ctx) (*types.PaymentProviderResponse, error) {
	var req types.CreatePaymentProviderRequest
	if err := c.Echo.Bind(&req); err != nil {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	if err := utils.ValidateStruct(req); err != nil {
		return nil, echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if req.ProviderType == models.ProviderTypeStripe && pm.config.ZoriStripeApp && !pm.config.ZoriOSS {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "Manual Stripe provider creation is not allowed. Please use Stripe App OAuth flow via /api/v1/payment-providers/instructions?provider_type=stripe")
	}

	project, err := pm.projectData.GetProject(c.Echo.Request().Context(), req.ProjectID, c.OrgID())
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusNotFound, "Project not found")
	}

	existing, err := pm.data.GetProviderByProjectAndType(c.Echo.Request().Context(), req.ProjectID, req.ProviderType)
	if err == nil && existing != nil {
		return nil, echo.NewHTTPError(http.StatusConflict, "Provider already exists for this project")
	}

	encryptedAPIKey, err := pm.encryptor.Encrypt(req.APIKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt API key: %w", err)
	}

	encryptedWebhookSecret, err := pm.encryptor.Encrypt(req.WebhookSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt webhook secret: %w", err)
	}

	var accountID string
	if req.ProviderType == models.ProviderTypeStripe {
		providerClient := stripe.NewClient(req.APIKey)
		account, err := providerClient.V1Accounts.Retrieve(c, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve Stripe account: %w", err)
		}
		accountID = account.ID
	}

	provider := &models.PaymentProvider{
		ProjectID:              project.ID,
		AccountID:              accountID,
		OrganizationID:         project.OrganizationID,
		ProviderType:           req.ProviderType,
		APIKeyEncrypted:        encryptedAPIKey,
		WebhookSecretEncrypted: encryptedWebhookSecret,
		IsActive:               true,
	}

	if err := pm.data.CreateProvider(c.Echo.Request().Context(), provider); err != nil {
		return nil, fmt.Errorf("failed to create provider: %w", err)
	}

	if req.ProviderType == models.ProviderTypeStripe && pm.backfillService != nil {
		go func() {
			ctx := context.Background()
			startDate := time.Now().AddDate(0, -3, 0)

			if err := pm.backfillService.BackfillStripePayments(ctx, provider, project, startDate); err != nil {
				fmt.Printf("Backfill failed for provider %s: %v\n", provider.ID, err)
			}
		}()
	}

	c.Echo.Response().Status = http.StatusCreated
	return types.ToPaymentProviderResponse(provider), nil
}

// @Summary List payment providers
// @Description Get all payment providers for the organization
// @Tags Payment Providers
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param project_id query string false "Filter by project ID"
// @Success 200 {object} types.ListPaymentProvidersResponse "List of payment providers"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/payment-providers [get]
func (pm *ProviderManager) ListProviders(c *ctx.Ctx) (*types.ListPaymentProvidersResponse, error) {
	projectID := c.Echo.QueryParam("project_id")

	var providers []*models.PaymentProvider
	var err error

	if projectID != "" {
		providers, err = pm.data.ListProvidersByProject(c.Echo.Request().Context(), projectID, c.OrgID())
	} else {
		providers, err = pm.data.ListProvidersByOrganization(c.Echo.Request().Context(), c.OrgID())
	}

	if err != nil {
		return nil, fmt.Errorf("failed to list providers: %w", err)
	}

	responses := make([]*types.PaymentProviderResponse, len(providers))
	for i, provider := range providers {
		responses[i] = types.ToPaymentProviderResponse(provider)
	}

	return &types.ListPaymentProvidersResponse{
		Providers: responses,
		Total:     len(responses),
	}, nil
}

// @Summary Get a payment provider
// @Description Get a single payment provider by ID
// @Tags Payment Providers
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param id path string true "Provider ID"
// @Success 200 {object} types.PaymentProviderResponse "Payment provider details"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 404 {object} map[string]interface{} "Provider not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/payment-providers/{id} [get]
func (pm *ProviderManager) GetProvider(c *ctx.Ctx) (*types.PaymentProviderResponse, error) {
	providerID := c.Echo.Param("id")
	if providerID == "" {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "Provider ID is required")
	}

	provider, err := pm.data.GetProvider(c.Echo.Request().Context(), providerID, c.OrgID())
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusNotFound, "Provider not found")
	}

	return types.ToPaymentProviderResponse(provider), nil
}

// @Summary Update a payment provider
// @Description Update payment provider credentials or settings
// @Tags Payment Providers
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param id path string true "Provider ID"
// @Param request body types.UpdatePaymentProviderRequest true "Update details"
// @Success 200 {object} types.PaymentProviderResponse "Updated payment provider"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 404 {object} map[string]interface{} "Provider not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/payment-providers/{id} [put]
func (pm *ProviderManager) UpdateProvider(c *ctx.Ctx) (*types.PaymentProviderResponse, error) {
	providerID := c.Echo.Param("id")
	if providerID == "" {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "Provider ID is required")
	}

	var req types.UpdatePaymentProviderRequest
	if err := c.Echo.Bind(&req); err != nil {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	if err := utils.ValidateStruct(req); err != nil {
		return nil, echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	exists, err := pm.data.ProviderExists(c.Echo.Request().Context(), providerID, c.OrgID())
	if err != nil {
		return nil, fmt.Errorf("failed to check provider existence: %w", err)
	}
	if !exists {
		return nil, echo.NewHTTPError(http.StatusNotFound, "Provider not found")
	}

	updates := make(map[string]interface{})

	if req.APIKey != nil {
		encrypted, err := pm.encryptor.Encrypt(*req.APIKey)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt API key: %w", err)
		}
		updates["api_key_encrypted"] = encrypted
	}

	if req.WebhookSecret != nil {
		encrypted, err := pm.encryptor.Encrypt(*req.WebhookSecret)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt webhook secret: %w", err)
		}
		updates["webhook_secret_encrypted"] = encrypted
	}

	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}

	provider, err := pm.data.UpdateProvider(c.Echo.Request().Context(), providerID, c.OrgID(), updates)
	if err != nil {
		return nil, fmt.Errorf("failed to update provider: %w", err)
	}

	return types.ToPaymentProviderResponse(provider), nil
}

// @Summary Delete a payment provider
// @Description Disconnect a payment provider from a project
// @Tags Payment Providers
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param id path string true "Provider ID"
// @Success 200 {object} map[string]string "Deletion confirmation"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 404 {object} map[string]interface{} "Provider not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/payment-providers/{id} [delete]
func (pm *ProviderManager) DeleteProvider(c *ctx.Ctx) (map[string]string, error) {
	providerID := c.Echo.Param("id")
	if providerID == "" {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "Provider ID is required")
	}

	exists, err := pm.data.ProviderExists(c.Echo.Request().Context(), providerID, c.OrgID())
	if err != nil {
		return nil, fmt.Errorf("failed to check provider existence: %w", err)
	}
	if !exists {
		return nil, echo.NewHTTPError(http.StatusNotFound, "Provider not found")
	}

	if err := pm.data.DeleteProvider(c.Echo.Request().Context(), providerID, c.OrgID()); err != nil {
		return nil, fmt.Errorf("failed to delete provider: %w", err)
	}

	return map[string]string{
		"message": "Payment provider deleted successfully",
	}, nil
}

func (pm *ProviderManager) DecryptAPIKey(encrypted string) (string, error) {
	return pm.encryptor.Decrypt(encrypted)
}

func (pm *ProviderManager) DecryptWebhookSecret(encrypted string) (string, error) {
	return pm.encryptor.Decrypt(encrypted)
}

func (pm *ProviderManager) GetProviderByProjectAndType(ctx context.Context, projectID string, providerType models.ProviderType) (*models.PaymentProvider, error) {
	return pm.data.GetProviderByProjectAndType(ctx, projectID, providerType)
}

func (pm *ProviderManager) GetProviderByAccountID(ctx context.Context, accountID string) (*models.PaymentProvider, error) {
	return pm.data.GetProviderByAccountID(ctx, accountID)
}

func (pm *ProviderManager) DeleteProviderByID(ctx context.Context, providerID string, orgID string) error {
	return pm.data.DeleteProvider(ctx, providerID, orgID)
}
