package services

import (
	"log"
	"net/netip"
	"strings"
	"zori/services/ingestion/types"

	"github.com/Cleverse/go-utilities/nullable"
	"github.com/oschwald/maxminddb-golang/v2"
)

type StageLocation struct {
	maxMindDb *maxminddb.Reader
}

func NewStageLocation() StageLocation {
	maxMindDB, err := maxminddb.Open("./ipdb.mmdb")

	// We are not panicking here, but logging the error
	// For a few reasons, we might want to disable location enrichment
	// Or we might be running tests
	if err != nil {
		log.Printf("Warning: Could not open ipdb.mmdb: %v. Location enrichment will be disabled.", err)
		return StageLocation{
			maxMindDb: nil,
		}
	}

	return StageLocation{
		maxMindDb: maxMindDB,
	}
}

func (s StageLocation) ProcessFrame(event *types.ClientEventFrameV1) error {
	if event.IP == "" || s.maxMindDb == nil {
		return nil
	}

	// When reading IP from headers, we could receive multiple IPs, for headers like X-Forwarded-For
	ipList := strings.Split(event.IP, ",")
	event.IP = ipList[0]

	if event.IP != "" {
		parsedIp, err := netip.ParseAddr(event.IP)
		if err != nil {
			return err
		}

		var countryCode string
		err = s.maxMindDb.Lookup(parsedIp).DecodePath(&countryCode, "country", "iso_code")
		if err != nil {
			return err
		}

		var cityName string
		err = s.maxMindDb.Lookup(parsedIp).DecodePath(&cityName, "city", "names", "en")
		if err != nil {
			return err
		}

		var lat float64
		err = s.maxMindDb.Lookup(parsedIp).DecodePath(&lat, "location", "latitude")
		if err != nil {
			return err
		}

		var lng float64
		err = s.maxMindDb.Lookup(parsedIp).DecodePath(&lng, "location", "longitude")
		if err != nil {
			return err
		}

		event.LocationCountryISO = nullable.FromString(countryCode).Ptr()
		event.LocationCity = nullable.FromString(cityName).Ptr()
		event.LocationLatitude = nullable.FromFloat64(lat).Ptr()
		event.LocationLongitude = nullable.FromFloat64(lng).Ptr()
	}

	return nil
}
