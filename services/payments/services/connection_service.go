package services

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
	"zori/internal/config"
	"zori/internal/ctx"
	"zori/internal/storage/postgres/models"
	"zori/internal/utils"
	"zori/services/payments/data"
	"zori/services/payments/types"
	projectsData "zori/services/projects/data"

	"github.com/labstack/echo/v4"
)

type ConnectionService struct {
	config          *config.Config
	data            *data.PaymentProviderData
	projectData     *projectsData.ProjectData
	encryptor       *utils.Encryptor
	backfillService *BackfillService
}

func NewConnectionService(
	cfg *config.Config,
	data *data.PaymentProviderData,
	projectData *projectsData.ProjectData,
	encryptor *utils.Encryptor,
	backfillService *BackfillService,
) *ConnectionService {
	return &ConnectionService{
		config:          cfg,
		data:            data,
		projectData:     projectData,
		encryptor:       encryptor,
		backfillService: backfillService,
	}
}

// @Summary Get provider connection instructions
// @Description Get instructions for connecting a payment provider (OAuth URL or manual credentials)
// @Tags Payment Providers
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param provider_type query string true "Provider type" Enums(stripe, paddle, paypal, lemon_squeezy, square)
// @Param project_id query string false "Project ID (required for Stripe Connect)"
// @Success 200 {object} types.ProviderInstructionsResponse "Connection instructions"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Router /api/v1/payment-providers/instructions [get]
func (cs *ConnectionService) GetProviderInstructions(c *ctx.Ctx) (*types.ProviderInstructionsResponse, error) {
	providerType := c.Echo.QueryParam("provider_type")
	if providerType == "" {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "provider_type query parameter is required")
	}

	// Validate provider type
	validProviders := map[string]bool{
		"stripe":        true,
		"paddle":        true,
		"paypal":        true,
		"lemon_squeezy": true,
		"square":        true,
	}
	if !validProviders[providerType] {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "Invalid provider type")
	}

	if providerType == "stripe" && cs.config.ZoriStripeApp && !cs.config.ZoriOSS {
		projectID := c.Echo.QueryParam("project_id")
		if projectID == "" {
			return nil, echo.NewHTTPError(http.StatusBadRequest, "project_id query parameter is required for Stripe Apps")
		}
		return cs.getStripeAppInstructions(c, projectID)
	}

	return cs.getManualConnectionInstructions(providerType)
}

func (cs *ConnectionService) getStripeAppInstructions(c *ctx.Ctx, projectID string) (*types.ProviderInstructionsResponse, error) {
	_, err := cs.projectData.GetProject(c.Echo.Request().Context(), projectID, c.OrgID())
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusNotFound, "Project not found or does not belong to organization")
	}

	state, err := generateSecureState()
	if err != nil {
		return nil, fmt.Errorf("failed to generate state: %w", err)
	}

	stateData := map[string]string{
		"org_id":     c.OrgID(),
		"project_id": projectID,
		"random":     state,
		"timestamp":  fmt.Sprintf("%d", time.Now().Unix()),
	}
	stateJSON, _ := json.Marshal(stateData)
	encodedState := base64.URLEncoding.EncodeToString(stateJSON)

	installLink := cs.config.ZoriStripeAppInstallLink
	if installLink == "" {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Stripe App installation link not configured")
	}

	oauthURL := fmt.Sprintf("%s&state=%s", installLink, url.QueryEscape(encodedState))

	return &types.ProviderInstructionsResponse{
		ProviderType:     models.ProviderTypeStripe,
		ConnectionMethod: "oauth",
		OAuthURL:         oauthURL,
		Instructions:     "Click the link below to install the Stripe App. You will be redirected to Stripe to authorize access.",
	}, nil
}

func (cs *ConnectionService) getManualConnectionInstructions(providerType string) (*types.ProviderInstructionsResponse, error) {
	fields := []types.ProviderField{}

	switch providerType {
	case "stripe":
		fields = []types.ProviderField{
			{
				Name:        "api_key",
				Label:       "Secret Key",
				Type:        "password",
				Required:    true,
				Placeholder: "sk_live_xxxxx or sk_test_xxxxx",
				HelpText:    "Your Stripe secret key from the Stripe Dashboard (Developers > API keys)",
			},
			{
				Name:        "webhook_secret",
				Label:       "Webhook Signing Secret",
				Type:        "password",
				Required:    true,
				Placeholder: "whsec_xxxxx",
				HelpText:    fmt.Sprintf("Create a webhook endpoint at %s/webhooks/stripe/:project_id and copy the signing secret", cs.config.ZoriAPIHost),
			},
		}
	case "paddle":
		fields = []types.ProviderField{
			{
				Name:        "api_key",
				Label:       "API Key",
				Type:        "password",
				Required:    true,
				Placeholder: "your-paddle-api-key",
				HelpText:    "Your Paddle API key",
			},
			{
				Name:        "webhook_secret",
				Label:       "Webhook Secret",
				Type:        "password",
				Required:    true,
				Placeholder: "your-webhook-secret",
				HelpText:    "Your Paddle webhook secret",
			},
		}
	default:
		fields = []types.ProviderField{
			{
				Name:        "api_key",
				Label:       "API Key",
				Type:        "password",
				Required:    true,
				Placeholder: "your-api-key",
				HelpText:    fmt.Sprintf("Your %s API key", providerType),
			},
			{
				Name:        "webhook_secret",
				Label:       "Webhook Secret",
				Type:        "password",
				Required:    true,
				Placeholder: "your-webhook-secret",
				HelpText:    fmt.Sprintf("Your %s webhook secret", providerType),
			},
		}
	}

	return &types.ProviderInstructionsResponse{
		ProviderType:     models.ProviderType(providerType),
		ConnectionMethod: "manual",
		Fields:           fields,
		Instructions:     fmt.Sprintf("Enter your %s credentials below to connect your account", providerType),
	}, nil
}

// @Summary Handle Stripe App OAuth callback
// @Description Handle the OAuth redirect from Stripe App installation and create payment provider
// @Tags Payment Providers
// @Accept json
// @Produce json
// @Param code query string true "Authorization code"
// @Param state query string true "State parameter for CSRF protection"
// @Success 200 {object} types.StripeConnectCallbackResponse "Connection successful"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/payment-providers/stripe/app/callback [get]
func (cs *ConnectionService) HandleStripeAppCallback(c echo.Context) error {
	code := c.QueryParam("code")
	state := c.QueryParam("state")

	if code == "" || state == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Missing required parameters (code, state)")
	}

	stateData, err := cs.validateState(state)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid state parameter: %v", err))
	}

	orgID := stateData["org_id"]
	if orgID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid state: missing organization ID")
	}

	projectID := stateData["project_id"]
	if projectID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid state: missing project ID")
	}

	redirectWithError := func(errorMsg string) error {
		redirectURL := fmt.Sprintf(
			"https://app.zorihq.com/projects/%s?status=error&provider=stripe&error=%s",
			url.QueryEscape(projectID),
			url.QueryEscape(errorMsg),
		)
		return c.Redirect(http.StatusFound, redirectURL)
	}

	project, err := cs.projectData.GetProject(c.Request().Context(), projectID, orgID)
	if err != nil {
		return redirectWithError("Project not found")
	}

	existing, err := cs.data.GetProviderByProjectAndType(c.Request().Context(), projectID, models.ProviderTypeStripe)
	if err == nil && existing != nil {
		return redirectWithError("Stripe provider already exists for this project")
	}

	tokenData, err := cs.exchangeCodeForToken(code)
	if err != nil {
		return redirectWithError("Failed to connect to Stripe")
	}

	accessToken := tokenData["access_token"].(string)
	stripeAccountID := tokenData["stripe_user_id"].(string)
	refreshToken := ""
	if rt, ok := tokenData["refresh_token"].(string); ok {
		refreshToken = rt
	}

	encryptedAccessToken, err := cs.encryptor.Encrypt(accessToken)
	if err != nil {
		return redirectWithError("Failed to encrypt credentials")
	}

	webhookSecret := cs.config.ZoriStripeAppWebhookSecret
	encryptedWebhookSecret, err := cs.encryptor.Encrypt(webhookSecret)
	if err != nil {
		return redirectWithError("Failed to encrypt credentials")
	}

	provider := &models.PaymentProvider{
		ProjectID:              project.ID,
		AccountID:              stripeAccountID,
		OrganizationID:         project.OrganizationID,
		ProviderType:           models.ProviderTypeStripe,
		APIKeyEncrypted:        encryptedAccessToken,
		WebhookSecretEncrypted: encryptedWebhookSecret,
		IsActive:               true,
	}

	if err := cs.data.CreateProvider(c.Request().Context(), provider); err != nil {
		return redirectWithError("Failed to save provider")
	}

	if cs.backfillService != nil {
		go func() {
			ctx := context.Background()
			startDate := time.Now().AddDate(0, -3, 0) // Last 3 months

			if err := cs.backfillService.BackfillStripePayments(ctx, provider, project, startDate); err != nil {
				fmt.Printf("Backfill failed for provider %s: %v\n", provider.ID, err)
			}
		}()
	}

	_ = refreshToken

	redirectURL := fmt.Sprintf(
		"https://app.zorihq.com/projects/%s?status=success&provider=stripe",
		url.QueryEscape(projectID),
	)
	return c.Redirect(http.StatusFound, redirectURL)
}

func (cs *ConnectionService) exchangeCodeForToken(code string) (map[string]interface{}, error) {
	secretKey := cs.config.ZoriStripeAppSecretKey

	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)

	req, err := http.NewRequest("POST", "https://api.stripe.com/v1/oauth/token", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.URL.RawQuery = data.Encode()
	req.SetBasicAuth(secretKey, "")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("stripe returned error: %s", string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result, nil
}

func (cs *ConnectionService) validateState(encodedState string) (map[string]string, error) {
	stateJSON, err := base64.URLEncoding.DecodeString(encodedState)
	if err != nil {
		return nil, fmt.Errorf("invalid state encoding: %w", err)
	}

	var stateData map[string]string
	if err := json.Unmarshal(stateJSON, &stateData); err != nil {
		return nil, fmt.Errorf("invalid state format: %w", err)
	}

	return stateData, nil
}

func generateSecureState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
