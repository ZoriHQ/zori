package services

import (
	"context"
	"encoding/json"
	"time"

	"zori/internal/natsstream"
	"zori/internal/telemetry"
	"zori/services/llmtraces/data"
	"zori/services/llmtraces/types"

	"github.com/nats-io/nats.go"
	"github.com/shopspring/decimal"
)

type LLMTraceProcessor struct {
	natsStream    *natsstream.Stream
	repository    *data.LLMTraceRepository
	tokenCounter  *TokenCounter
	priceFetcher  *PriceFetcher
	logger        *telemetry.Logger
	subscription  *nats.Subscription
}

func NewLLMTraceProcessor(
	natsStream *natsstream.Stream,
	repository *data.LLMTraceRepository,
	tokenCounter *TokenCounter,
	priceFetcher *PriceFetcher,
	logger *telemetry.Logger,
) *LLMTraceProcessor {
	return &LLMTraceProcessor{
		natsStream:   natsStream,
		repository:   repository,
		tokenCounter: tokenCounter,
		priceFetcher: priceFetcher,
		logger:       logger,
	}
}

func (p *LLMTraceProcessor) Start(ctx context.Context) error {
	err := p.natsStream.UpsertJetStream(natsstream.LLMTracesStreamName, natsstream.LLMTracesRawSubject)
	if err != nil {
		return err
	}

	sub, err := p.natsStream.GetConnection().Subscribe(natsstream.LLMTracesRawSubject, func(msg *nats.Msg) {
		p.processMessage(ctx, msg)
	})
	if err != nil {
		return err
	}

	p.subscription = sub
	p.logger.Info("LLM trace processor started", telemetry.String("subject", natsstream.LLMTracesRawSubject))
	return nil
}

func (p *LLMTraceProcessor) Stop() error {
	if p.subscription != nil {
		return p.subscription.Unsubscribe()
	}
	return nil
}

func (p *LLMTraceProcessor) processMessage(ctx context.Context, msg *nats.Msg) {
	var frame types.LLMTraceFrame
	if err := json.Unmarshal(msg.Data, &frame); err != nil {
		p.logger.Error("Failed to unmarshal LLM trace frame", telemetry.Error(err))
		return
	}

	trace, err := p.buildTrace(ctx, &frame)
	if err != nil {
		p.logger.Error("Failed to build LLM trace", telemetry.Error(err))
		return
	}

	if err := p.repository.Insert(ctx, trace); err != nil {
		p.logger.Error("Failed to insert LLM trace", telemetry.Error(err))
		return
	}
}

func (p *LLMTraceProcessor) buildTrace(ctx context.Context, frame *types.LLMTraceFrame) (*types.LLMTrace, error) {
	inputTokens := uint32(0)
	outputTokens := uint32(0)

	if frame.InputTokens != nil {
		inputTokens = *frame.InputTokens
	} else if frame.InputPrompt != nil {
		counted, err := p.tokenCounter.CountTokens(*frame.InputPrompt, frame.Model)
		if err != nil {
			p.logger.Warn("Failed to count input tokens, using 0", telemetry.Error(err))
		} else {
			inputTokens = counted
		}
	}

	if frame.OutputTokens != nil {
		outputTokens = *frame.OutputTokens
	} else if frame.OutputPrompt != nil {
		counted, err := p.tokenCounter.CountTokens(*frame.OutputPrompt, frame.Model)
		if err != nil {
			p.logger.Warn("Failed to count output tokens, using 0", telemetry.Error(err))
		} else {
			outputTokens = counted
		}
	}

	priceInfo, err := p.priceFetcher.GetPrice(ctx, frame.Provider, frame.Model)
	if err != nil {
		p.logger.Warn("Failed to get price info, using defaults", telemetry.Error(err))
		priceInfo = &types.PriceInfo{
			InputPricePerToken:  decimal.NewFromFloat(0.000001),
			OutputPricePerToken: decimal.NewFromFloat(0.000002),
		}
	}

	inputCost, outputCost, totalCost := p.priceFetcher.CalculateCost(priceInfo, inputTokens, outputTokens)

	var extraFee *decimal.Decimal
	if frame.ExtraFee != nil {
		fee, err := decimal.NewFromString(*frame.ExtraFee)
		if err == nil {
			extraFee = &fee
			totalCost = totalCost.Add(fee)
		}
	}

	clientTimestamp := time.Now().UTC()
	if frame.Timestamp != nil {
		clientTimestamp = time.UnixMilli(*frame.Timestamp).UTC()
	}

	return &types.LLMTrace{
		VisitorID:          frame.VisitorID,
		CustomerID:         frame.CustomerID,
		Provider:           frame.Provider,
		Model:              frame.Model,
		InputTokens:        inputTokens,
		OutputTokens:       outputTokens,
		InputPrompt:        frame.InputPrompt,
		OutputPrompt:       frame.OutputPrompt,
		InputCost:          inputCost,
		OutputCost:         outputCost,
		ExtraFee:           extraFee,
		TotalCost:          totalCost,
		ProjectID:          frame.ProjectID,
		OrganizationID:     frame.OrganizationID,
		ClientTimestampUTC: clientTimestamp,
		ServerTimestampUTC: time.Now().UTC(),
	}, nil
}
