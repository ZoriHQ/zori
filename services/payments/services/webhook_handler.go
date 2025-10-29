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
	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/webhook"
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

	var rawStripeEvent stripe.Event
	if err := c.Bind(&rawStripeEvent); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Failed to bind request body")
	}

	webhookSecret := wh.config.ZoriStripeConnectWebhookSecret
	if !rawStripeEvent.Livemode {
		webhookSecret = wh.config.ZoriStripeConnectWebhookSandboxSecret
	}

	payload, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Failed to read request body")
	}

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

	paymentFrame, err := wh.extractStripePaymentData(&event, project)
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
	case "charge.succeeded", "payment_intent.succeeded", "invoice.payment_succeeded":
	default:
		return c.String(http.StatusOK, fmt.Sprintf("Event type %s acknowledged but not processed", event.Type))
	}

	paymentFrame, err := wh.extractStripePaymentData(&event, project)
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

func (wh *WebhookHandler) extractStripePaymentData(event *stripe.Event, project *models.Project) (*types.PaymentEventFrame, error) {
	var paymentID string
	var amount int64
	var currency string
	var productName string
	var created int64
	var metadata map[string]string

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

	case stripe.EventTypePaymentIntentSucceeded:
		var paymentIntent stripe.PaymentIntent
		if err := json.Unmarshal(event.Data.Raw, &paymentIntent); err != nil {
			return nil, fmt.Errorf("failed to parse payment intent: %w", err)
		}
		paymentID = paymentIntent.ID
		amount = paymentIntent.Amount
		currency = string(paymentIntent.Currency)
		productName = paymentIntent.Description
		created = paymentIntent.Created
		metadata = paymentIntent.Metadata

	case stripe.EventTypeInvoicePaymentSucceeded:
		var invoice stripe.Invoice
		if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
			return nil, fmt.Errorf("failed to parse invoice: %w", err)
		}
		paymentID = invoice.ID
		amount = invoice.Total
		currency = string(invoice.Currency)
		productName = invoice.Description
		created = invoice.Created
		metadata = invoice.Metadata

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
