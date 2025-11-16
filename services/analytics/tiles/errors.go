package tiles

type TimelineFetchError struct{}

func NewTimelineFetchError() TimelineFetchError {
	return TimelineFetchError{}
}

func (e TimelineFetchError) Error() string {
	return "failed to fetch timeline data"
}
