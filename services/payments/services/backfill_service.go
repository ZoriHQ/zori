package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"
	"zori/internal/natsstream"
	"zori/internal/storage/postgres/models"
	"zori/internal/utils"
	"zori/services/payments/types"

	"github.com/stripe/stripe-go/v83"
	"github.com/stripe/stripe-go/v83/client"
)

type BackfillService struct {
	natsStream *natsstream.Stream
	encryptor  *utils.Encryptor
}

func NewBackfillService(
	natsStream *natsstream.Stream,
	encryptor *utils.Encryptor,
) *BackfillService {
	return &BackfillService{
		natsStream: natsStream,
		encryptor:  encryptor,
	}
}

func (bs *BackfillService) BackfillStripePayments(
	ctx context.Context,
	provider *models.PaymentProvider,
	project *models.Project,
	startDate time.Time,
) error {
	apiKey, err := bs.encryptor.Decrypt(provider.APIKeyEncrypted)
	if err != nil {
		return fmt.Errorf("failed to decrypt API key: %w", err)
	}

	sc := &client.API{}
	sc.Init(apiKey, nil)

	log.Printf("Starting backfill for provider %s from %s", provider.ID, startDate.Format(time.RFC3339))

	if err := bs.backfillInvoices(ctx, sc, provider, project, startDate); err != nil {
		return fmt.Errorf("failed to backfill invoices: %w", err)
	}

	if err := bs.backfillCharges(ctx, sc, provider, project, startDate); err != nil {
		return fmt.Errorf("failed to backfill charges: %w", err)
	}

	log.Printf("Completed backfill for provider %s", provider.ID)

	return nil
}

func (bs *BackfillService) backfillCharges(
	ctx context.Context,
	sc *client.API,
	provider *models.PaymentProvider,
	project *models.Project,
	startDate time.Time,
) error {
	params := &stripe.ChargeListParams{
		ListParams: stripe.ListParams{
			Limit: stripe.Int64(100),
		},
	}
	params.Filters.AddFilter("created", "gte", fmt.Sprintf("%d", startDate.Unix()))

	count := 0
	iter := sc.Charges.List(params)
	for iter.Next() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		charge := iter.Charge()

		if charge.Status != "succeeded" {
			continue
		}

		metadata := charge.Metadata

		paymentFrame := &types.PaymentEventFrame{
			PaymentID:        charge.ID,
			ProviderType:     "stripe",
			PaymentStatus:    "succeeded",
			ProductName:      getChargeDescription(charge),
			Amount:           charge.Amount,
			Currency:         string(charge.Currency),
			PaymentTimestamp: time.Unix(charge.Created, 0).UTC(),
			ProjectID:        project.ID,
			OrganizationID:   project.OrganizationID,
			Metadata:         metadata,
		}

		if visitorID, ok := metadata["zori_visitor_id"]; ok && visitorID != "" {
			paymentFrame.VisitorID = &visitorID
		}

		paymentFrameBytes, err := json.Marshal(paymentFrame)
		if err != nil {
			log.Printf("Failed to marshal payment frame for charge %s: %v", charge.ID, err)
			continue
		}

		if err := bs.natsStream.GetConnection().Publish("payments:raw", paymentFrameBytes); err != nil {
			log.Printf("Failed to publish payment event for charge %s: %v", charge.ID, err)
			continue
		}

		count++
	}

	if err := iter.Err(); err != nil {
		return fmt.Errorf("error iterating charges: %w", err)
	}

	log.Printf("Backfilled %d charges", count)
	return nil
}

func (bs *BackfillService) backfillInvoices(
	ctx context.Context,
	sc *client.API,
	provider *models.PaymentProvider,
	project *models.Project,
	startDate time.Time,
) error {
	params := &stripe.InvoiceListParams{
		ListParams: stripe.ListParams{
			Limit: stripe.Int64(100),
		},
		Status: stripe.String("paid"),
	}
	params.Filters.AddFilter("created", "gte", fmt.Sprintf("%d", startDate.Unix()))

	count := 0
	iter := sc.Invoices.List(params)
	for iter.Next() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		invoice := iter.Invoice()

		metadata := invoice.Metadata

		subscriptionMetadata := extractSubscriptionMetadataFromInvoice(invoice)
		if subscriptionMetadata != nil {
			metadata = mergeMetadata(metadata, subscriptionMetadata)
		}

		paymentFrame := &types.PaymentEventFrame{
			PaymentID:        invoice.ID,
			ProviderType:     "stripe",
			PaymentStatus:    "succeeded",
			ProductName:      getInvoiceDescription(invoice),
			Amount:           invoice.Total,
			Currency:         string(invoice.Currency),
			PaymentTimestamp: time.Unix(invoice.Created, 0).UTC(),
			ProjectID:        project.ID,
			OrganizationID:   project.OrganizationID,
			Metadata:         metadata,
		}

		if visitorID, ok := metadata["zori_visitor_id"]; ok && visitorID != "" {
			paymentFrame.VisitorID = &visitorID
		}

		paymentFrameBytes, err := json.Marshal(paymentFrame)
		if err != nil {
			log.Printf("Failed to marshal payment frame for invoice %s: %v", invoice.ID, err)
			continue
		}

		if err := bs.natsStream.GetConnection().Publish("payments:raw", paymentFrameBytes); err != nil {
			log.Printf("Failed to publish payment event for invoice %s: %v", invoice.ID, err)
			continue
		}

		count++
	}

	if err := iter.Err(); err != nil {
		return fmt.Errorf("error iterating invoices: %w", err)
	}

	log.Printf("Backfilled %d subscription invoices", count)
	return nil
}

func getChargeDescription(charge *stripe.Charge) string {
	if charge.Description != "" {
		return charge.Description
	}
	return "Unknown Product"
}

func getInvoiceDescription(invoice *stripe.Invoice) string {
	if invoice.Description != "" {
		return invoice.Description
	}
	return "Subscription Payment"
}

func extractSubscriptionMetadataFromInvoice(invoice *stripe.Invoice) map[string]string {
	if invoice != nil &&
		invoice.Parent != nil &&
		invoice.Parent.SubscriptionDetails != nil &&
		invoice.Parent.SubscriptionDetails.Metadata != nil {
		return invoice.Parent.SubscriptionDetails.Metadata
	}
	return nil
}

func mergeMetadata(existingMetadata, subscriptionMetadata map[string]string) map[string]string {
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
