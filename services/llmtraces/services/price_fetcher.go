package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"zori/internal/cache"
	"zori/services/llmtraces/types"

	"github.com/shopspring/decimal"
)

const (
	PriceCacheTTL = 30 * time.Second
)

type PriceFetcher struct {
	cacheService *cache.CacheService
	fallbackPrices map[string]types.PriceInfo
}

func NewPriceFetcher(cacheService *cache.CacheService) *PriceFetcher {
	pf := &PriceFetcher{
		cacheService:   cacheService,
		fallbackPrices: make(map[string]types.PriceInfo),
	}
	pf.initFallbackPrices()
	return pf
}

func (pf *PriceFetcher) GetPrice(ctx context.Context, provider, model string) (*types.PriceInfo, error) {
	cacheKey := cache.LLMPriceCacheKey.FromValue(fmt.Sprintf("%s:%s", provider, model))

	cachedPrice, err := pf.cacheService.Get(ctx, cacheKey)
	if err == nil && cachedPrice != nil {
		var priceInfo types.PriceInfo
		if err := json.Unmarshal([]byte(*cachedPrice), &priceInfo); err == nil {
			return &priceInfo, nil
		}
	}

	priceInfo := pf.getFallbackPrice(provider, model)

	if err := pf.cacheService.Set(ctx, cacheKey, priceInfo, PriceCacheTTL); err != nil {
		return priceInfo, nil
	}

	return priceInfo, nil
}

func (pf *PriceFetcher) CalculateCost(priceInfo *types.PriceInfo, inputTokens, outputTokens uint32) (inputCost, outputCost, totalCost decimal.Decimal) {
	inputCost = priceInfo.InputPricePerToken.Mul(decimal.NewFromInt(int64(inputTokens)))
	outputCost = priceInfo.OutputPricePerToken.Mul(decimal.NewFromInt(int64(outputTokens)))
	totalCost = inputCost.Add(outputCost)
	return
}

func (pf *PriceFetcher) getFallbackPrice(provider, model string) *types.PriceInfo {
	key := fmt.Sprintf("%s:%s", strings.ToLower(provider), strings.ToLower(model))

	if price, ok := pf.fallbackPrices[key]; ok {
		return &price
	}

	providerKey := strings.ToLower(provider) + ":"
	for k, price := range pf.fallbackPrices {
		if strings.HasPrefix(k, providerKey) && strings.Contains(strings.ToLower(model), strings.TrimPrefix(k, providerKey)) {
			return &price
		}
	}

	return &types.PriceInfo{
		Provider:            types.LLMProvider(provider),
		Model:               model,
		InputPricePerToken:  decimal.NewFromFloat(0.000001),
		OutputPricePerToken: decimal.NewFromFloat(0.000002),
		UpdatedAt:           time.Now(),
	}
}

func (pf *PriceFetcher) initFallbackPrices() {
	pf.fallbackPrices["openai:gpt-4o"] = types.PriceInfo{
		Provider:            types.ProviderOpenAI,
		Model:               "gpt-4o",
		InputPricePerToken:  decimal.NewFromFloat(0.0000025),
		OutputPricePerToken: decimal.NewFromFloat(0.00001),
		UpdatedAt:           time.Now(),
	}
	pf.fallbackPrices["openai:gpt-4o-mini"] = types.PriceInfo{
		Provider:            types.ProviderOpenAI,
		Model:               "gpt-4o-mini",
		InputPricePerToken:  decimal.NewFromFloat(0.00000015),
		OutputPricePerToken: decimal.NewFromFloat(0.0000006),
		UpdatedAt:           time.Now(),
	}
	pf.fallbackPrices["openai:gpt-4-turbo"] = types.PriceInfo{
		Provider:            types.ProviderOpenAI,
		Model:               "gpt-4-turbo",
		InputPricePerToken:  decimal.NewFromFloat(0.00001),
		OutputPricePerToken: decimal.NewFromFloat(0.00003),
		UpdatedAt:           time.Now(),
	}
	pf.fallbackPrices["openai:gpt-4"] = types.PriceInfo{
		Provider:            types.ProviderOpenAI,
		Model:               "gpt-4",
		InputPricePerToken:  decimal.NewFromFloat(0.00003),
		OutputPricePerToken: decimal.NewFromFloat(0.00006),
		UpdatedAt:           time.Now(),
	}
	pf.fallbackPrices["openai:gpt-3.5-turbo"] = types.PriceInfo{
		Provider:            types.ProviderOpenAI,
		Model:               "gpt-3.5-turbo",
		InputPricePerToken:  decimal.NewFromFloat(0.0000005),
		OutputPricePerToken: decimal.NewFromFloat(0.0000015),
		UpdatedAt:           time.Now(),
	}

	pf.fallbackPrices["anthropic:claude-3-5-sonnet"] = types.PriceInfo{
		Provider:            types.ProviderAnthropic,
		Model:               "claude-3-5-sonnet",
		InputPricePerToken:  decimal.NewFromFloat(0.000003),
		OutputPricePerToken: decimal.NewFromFloat(0.000015),
		UpdatedAt:           time.Now(),
	}
	pf.fallbackPrices["anthropic:claude-3-opus"] = types.PriceInfo{
		Provider:            types.ProviderAnthropic,
		Model:               "claude-3-opus",
		InputPricePerToken:  decimal.NewFromFloat(0.000015),
		OutputPricePerToken: decimal.NewFromFloat(0.000075),
		UpdatedAt:           time.Now(),
	}
	pf.fallbackPrices["anthropic:claude-3-sonnet"] = types.PriceInfo{
		Provider:            types.ProviderAnthropic,
		Model:               "claude-3-sonnet",
		InputPricePerToken:  decimal.NewFromFloat(0.000003),
		OutputPricePerToken: decimal.NewFromFloat(0.000015),
		UpdatedAt:           time.Now(),
	}
	pf.fallbackPrices["anthropic:claude-3-haiku"] = types.PriceInfo{
		Provider:            types.ProviderAnthropic,
		Model:               "claude-3-haiku",
		InputPricePerToken:  decimal.NewFromFloat(0.00000025),
		OutputPricePerToken: decimal.NewFromFloat(0.00000125),
		UpdatedAt:           time.Now(),
	}

	pf.fallbackPrices["google:gemini-1.5-pro"] = types.PriceInfo{
		Provider:            types.ProviderGoogle,
		Model:               "gemini-1.5-pro",
		InputPricePerToken:  decimal.NewFromFloat(0.00000125),
		OutputPricePerToken: decimal.NewFromFloat(0.000005),
		UpdatedAt:           time.Now(),
	}
	pf.fallbackPrices["google:gemini-1.5-flash"] = types.PriceInfo{
		Provider:            types.ProviderGoogle,
		Model:               "gemini-1.5-flash",
		InputPricePerToken:  decimal.NewFromFloat(0.000000075),
		OutputPricePerToken: decimal.NewFromFloat(0.0000003),
		UpdatedAt:           time.Now(),
	}

	pf.fallbackPrices["mistral:mistral-large"] = types.PriceInfo{
		Provider:            types.ProviderMistral,
		Model:               "mistral-large",
		InputPricePerToken:  decimal.NewFromFloat(0.000002),
		OutputPricePerToken: decimal.NewFromFloat(0.000006),
		UpdatedAt:           time.Now(),
	}
	pf.fallbackPrices["mistral:mistral-small"] = types.PriceInfo{
		Provider:            types.ProviderMistral,
		Model:               "mistral-small",
		InputPricePerToken:  decimal.NewFromFloat(0.000001),
		OutputPricePerToken: decimal.NewFromFloat(0.000003),
		UpdatedAt:           time.Now(),
	}
}
