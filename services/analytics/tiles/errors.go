package tiles

type TimelineFetchError struct{}

func NewTimelineFetchError() TimelineFetchError {
	return TimelineFetchError{}
}

func (e TimelineFetchError) Error() string {
	return "failed to fetch timeline data"
}

type LLMCostFetchError struct{}

func NewLLMCostFetchError() LLMCostFetchError {
	return LLMCostFetchError{}
}

func (e LLMCostFetchError) Error() string {
	return "failed to fetch LLM cost data"
}

type LLMModelsCostFetchError struct{}

func NewLLMModelsCostFetchError() LLMModelsCostFetchError {
	return LLMModelsCostFetchError{}
}

func (e LLMModelsCostFetchError) Error() string {
	return "failed to fetch LLM models cost data"
}
