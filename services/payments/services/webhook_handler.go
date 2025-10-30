package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"zori/internal/config"
	"zori/internal/natsstream"
	"zori/internal/storage/postgres/models"
	"zori/services/payments/types"
	"zori/services/projects/data"

	"github.com/labstack/echo/v4"
	"github.com/stripe/stripe-go/v83"
	"github.com/stripe/stripe-go/v83/client"
	"github.com/stripe/stripe-go/v83/webhook"
)

const (
	paymentEventsStream  = "payments:raw"
	paymentEventsSubject = "payments:raw"
)

type WebhookHandler struct {
	natsStream      *natsstream.Stream
	projectData     *data.ProjectData
	providerManager *ProviderManager
	config          *config.Config
}

func NewWebhookHandler(
	natsStream *natsstream.Stream,
	projectData *data.ProjectData,
	providerManager *ProviderManager,
	conf *config.Config,
) *WebhookHandler {
	return &WebhookHandler{
		natsStream:      natsStream,
		config:          conf,
		projectData:     projectData,
		providerManager: providerManager,
	}
}

// HandleStripeConnectWebhook handles webhook for cloud version of Zori
// For regular self-hosted versions this can be enabled, but generally not used, not recommended
// As you'd need to have two separate Stripe accounts to make this work.
func (wh *WebhookHandler) HandleStripeConnectWebhook(c echo.Context) error {
	signature := c.Request().Header.Get("Stripe-Signature")
	if signature == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Missing Stripe-Signature header")
	}

	payload, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Failed to read request body")
	}

	webhookSecret := wh.config.ZoriStripeConnectWebhookSecret

	event, err := webhook.ConstructEvent(payload, signature, webhookSecret)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, fmt.Sprintf("Webhook signature verification failed: %v", err))
	}

	paymentProvider, err := wh.providerManager.GetProviderByAccountID(c.Request().Context(), event.Account)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Account not found")
	}

	project, err := wh.projectData.GetProjectByID(c.Request().Context(), paymentProvider.ProjectID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Project not found")
	}

	paymentFrame, err := wh.extractStripePaymentData(&event, project, paymentProvider)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Failed to extract payment data: %v", err))
	}

	paymentFrameBytes, err := json.Marshal(paymentFrame)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to serialize payment event")
	}

	if err := wh.natsStream.GetConnection().Publish(paymentEventsSubject, paymentFrameBytes); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to publish payment event")
	}

	return c.String(http.StatusOK, "Payment event received and queued for processing")
}

func (wh *WebhookHandler) HandleStripeWebhook(c echo.Context) error {
	projectID := c.Param("project_id")

	project, err := wh.projectData.GetProjectByID(context.Background(), projectID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Project not found")
	}

	provider, err := wh.providerManager.GetProviderByProjectAndType(context.Background(), projectID, models.ProviderTypeStripe)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Stripe integration not found for this project")
	}

	if !provider.IsActive {
		return echo.NewHTTPError(http.StatusBadRequest, "Stripe integration is not active")
	}

	signature := c.Request().Header.Get("Stripe-Signature")
	if signature == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Missing Stripe-Signature header")
	}

	webhookSecret, err := wh.providerManager.DecryptWebhookSecret(provider.WebhookSecretEncrypted)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to decrypt webhook secret")
	}

	payload, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Failed to read request body")
	}

	event, err := webhook.ConstructEvent(payload, signature, webhookSecret)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, fmt.Sprintf("Webhook signature verification failed: %v", err))
	}

	switch event.Type {
	case "charge.succeeded", "invoice.payment_succeeded":
	default:
		return c.String(http.StatusOK, fmt.Sprintf("Event type %s acknowledged but not processed", event.Type))
	}

	paymentFrame, err := wh.extractStripePaymentData(&event, project, provider)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Failed to extract payment data: %v", err))
	}

	paymentFrameBytes, err := json.Marshal(paymentFrame)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to serialize payment event")
	}

	if err := wh.natsStream.GetConnection().Publish(paymentEventsSubject, paymentFrameBytes); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to publish payment event")
	}

	return c.String(http.StatusOK, "Payment event received and queued for processing")
}

func mergeSubscriptionMetadata(existingMetadata, subscriptionMetadata map[string]string) map[string]string {
	if existingMetadata == nil {
		existingMetadata = make(map[string]string)
	}

	for key, value := range subscriptionMetadata {
		if _, exists := existingMetadata[key]; !exists && value != "" {
			existingMetadata[key] = value
		}
	}

	return existingMetadata
}

func extractSubscriptionMetadata(invoice *stripe.Invoice) map[string]string {
	if invoice != nil &&
		invoice.Parent != nil &&
		invoice.Parent.SubscriptionDetails != nil &&
		invoice.Parent.SubscriptionDetails.Metadata != nil {
		return invoice.Parent.SubscriptionDetails.Metadata
	}
	return nil
}

func (wh *WebhookHandler) createStripeClient(provider *models.PaymentProvider) (*client.API, error) {
	sc := &client.API{}

	if wh.config.ZoriStripeConnect && !wh.config.ZoriOSS {
		sc.Init(wh.config.ZoriStripeConnectSecretKey, &stripe.Backends{
			API: stripe.GetBackendWithConfig(stripe.APIBackend, &stripe.BackendConfig{
				URL: stripe.String(stripe.APIURL),
			}),
		})
	} else {
		apiKey, err := wh.providerManager.DecryptAPIKey(provider.APIKeyEncrypted)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt API key: %w", err)
		}
		sc.Init(apiKey, nil)
	}

	return sc, nil
}

func (wh *WebhookHandler) fetchChargeWithInvoice(chargeID string, provider *models.PaymentProvider) (*stripe.Charge, error) {
	sc, err := wh.createStripeClient(provider)
	if err != nil {
		return nil, err
	}

	params := &stripe.ChargeParams{}
	params.AddExpand("invoice")

	if wh.config.ZoriStripeConnect && !wh.config.ZoriOSS {
		// For Connect, specify the account
		params.SetStripeAccount(provider.AccountID)
	}

	return sc.Charges.Get(chargeID, params)
}

func (wh *WebhookHandler) fetchPaymentIntentWithInvoice(paymentIntentID string, provider *models.PaymentProvider) (*stripe.PaymentIntent, error) {
	sc, err := wh.createStripeClient(provider)
	if err != nil {
		return nil, err
	}

	params := &stripe.PaymentIntentParams{}
	params.AddExpand("invoice")

	if wh.config.ZoriStripeConnect && !wh.config.ZoriOSS {
		// For Connect, specify the account
		params.SetStripeAccount(provider.AccountID)
	}

	return sc.PaymentIntents.Get(paymentIntentID, params)
}

func (wh *WebhookHandler) extractStripePaymentData(event *stripe.Event, project *models.Project, provider *models.PaymentProvider) (*types.PaymentEventFrame, error) {
	var paymentID string
	var amount int64
	var currency string
	var productName string
	var created int64
	var metadata map[string]string
	var invoice *stripe.Invoice

	switch event.Type {
	case stripe.EventTypeChargeSucceeded:
		var charge stripe.Charge
		if err := json.Unmarshal(event.Data.Raw, &charge); err != nil {
			return nil, fmt.Errorf("failed to parse charge: %w", err)
		}

		paymentID = charge.ID
		amount = charge.Amount
		currency = string(charge.Currency)
		productName = charge.Description
		created = charge.Created
		metadata = charge.Metadata

		fullCharge, err := wh.fetchChargeWithInvoice(charge.ID, provider)
		if err == nil && fullCharge != nil {
			_ = fullCharge
		}

	case stripe.EventTypeInvoicePaymentSucceeded:
		var invoiceData stripe.Invoice
		if err := json.Unmarshal(event.Data.Raw, &invoiceData); err != nil {
			return nil, fmt.Errorf("failed to parse invoice: %w", err)
		}
		paymentID = invoiceData.ID
		amount = invoiceData.Total
		currency = string(invoiceData.Currency)
		productName = invoiceData.Description
		created = invoiceData.Created
		metadata = invoiceData.Metadata
		invoice = &invoiceData

		if subscriptionMetadata := extractSubscriptionMetadata(invoice); subscriptionMetadata != nil {
			metadata = mergeSubscriptionMetadata(metadata, subscriptionMetadata)
		}

	default:
		return nil, fmt.Errorf("unsupported event type: %s", event.Type)
	}

	if paymentID == "" {
		return nil, fmt.Errorf("missing payment ID")
	}

	if currency == "" {
		currency = "usd"
	}

	if productName == "" {
		productName = "Unknown Product"
	}

	var visitorID *string
	if metadata != nil {
		if zoriVisitorID, ok := metadata["zori_visitor_id"]; ok && zoriVisitorID != "" {
			visitorID = &zoriVisitorID
		}
	}

	paymentTimestamp := time.Unix(created, 0).UTC()

	return &types.PaymentEventFrame{
		PaymentID:        paymentID,
		VisitorID:        visitorID,
		ProviderType:     "stripe",
		PaymentStatus:    "succeeded",
		ProductName:      productName,
		Amount:           amount,
		Currency:         strings.ToUpper(currency),
		PaymentTimestamp: paymentTimestamp,
		ProjectID:        project.ID,
		OrganizationID:   project.OrganizationID,
		Metadata:         metadata,
	}, nil
}
