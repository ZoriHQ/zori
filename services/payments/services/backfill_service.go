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

	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/client"
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

	if err := bs.backfillCharges(ctx, sc, provider, project, startDate); err != nil {
		return fmt.Errorf("failed to backfill charges: %w", err)
	}

	if err := bs.backfillPaymentIntents(ctx, sc, provider, project, startDate); err != nil {
		return fmt.Errorf("failed to backfill payment intents: %w", err)
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
			Metadata:         charge.Metadata,
		}

		if visitorID, ok := charge.Metadata["zori_visitor_id"]; ok && visitorID != "" {
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

func (bs *BackfillService) backfillPaymentIntents(
	ctx context.Context,
	sc *client.API,
	provider *models.PaymentProvider,
	project *models.Project,
	startDate time.Time,
) error {
	params := &stripe.PaymentIntentListParams{
		ListParams: stripe.ListParams{
			Limit: stripe.Int64(100),
		},
	}
	params.Filters.AddFilter("created", "gte", fmt.Sprintf("%d", startDate.Unix()))

	count := 0
	iter := sc.PaymentIntents.List(params)
	for iter.Next() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		pi := iter.PaymentIntent()

		if pi.Status != stripe.PaymentIntentStatusSucceeded {
			continue
		}

		paymentFrame := &types.PaymentEventFrame{
			PaymentID:        pi.ID,
			ProviderType:     "stripe",
			PaymentStatus:    "succeeded",
			ProductName:      getPaymentIntentDescription(pi),
			Amount:           pi.Amount,
			Currency:         string(pi.Currency),
			PaymentTimestamp: time.Unix(pi.Created, 0).UTC(),
			ProjectID:        project.ID,
			OrganizationID:   project.OrganizationID,
			Metadata:         pi.Metadata,
		}

		if visitorID, ok := pi.Metadata["zori_visitor_id"]; ok && visitorID != "" {
			paymentFrame.VisitorID = &visitorID
		}

		paymentFrameBytes, err := json.Marshal(paymentFrame)
		if err != nil {
			log.Printf("Failed to marshal payment frame for payment intent %s: %v", pi.ID, err)
			continue
		}

		if err := bs.natsStream.GetConnection().Publish("payments:raw", paymentFrameBytes); err != nil {
			log.Printf("Failed to publish payment event for payment intent %s: %v", pi.ID, err)
			continue
		}

		count++
	}

	if err := iter.Err(); err != nil {
		return fmt.Errorf("error iterating payment intents: %w", err)
	}

	log.Printf("Backfilled %d payment intents", count)
	return nil
}

func getChargeDescription(charge *stripe.Charge) string {
	if charge.Description != "" {
		return charge.Description
	}
	return "Unknown Product"
}

func getPaymentIntentDescription(pi *stripe.PaymentIntent) string {
	if pi.Description != "" {
		return pi.Description
	}
	return "Unknown Product"
}
