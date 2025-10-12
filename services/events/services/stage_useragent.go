package services

import (
	"zori/services/ingestion/types"

	"github.com/Cleverse/go-utilities/nullable"
	"github.com/medama-io/go-useragent"
)

type StageUserAgent struct {
	ua *useragent.Parser
}

func NewStageUserAgent() ProcessorStage {
	return StageUserAgent{
		ua: useragent.NewParser(),
	}
}

// ProcessFrame for StageUserAgent parses the request user-agent header and determines OS and browser information
func (s StageUserAgent) ProcessFrame(event *types.ClientEventFrameV1) error {
	if event.UserAgent == "" {
		return nil
	}

	parsedUserAgent := s.ua.Parse(event.UserAgent)

	event.BrowserName = nullable.FromString(parsedUserAgent.Browser().String()).Ptr()
	event.DeviceType = nullable.FromString(parsedUserAgent.Device().String()).Ptr()
	event.OsName = nullable.FromString(parsedUserAgent.OS().String()).Ptr()

	return nil
}
