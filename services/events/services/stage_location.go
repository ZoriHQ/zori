package services

import (
	"log"
	"net/netip"
	"zori/services/ingestion/types"

	"github.com/Cleverse/go-utilities/nullable"
	"github.com/oschwald/maxminddb-golang/v2"
)

type StageLocation struct {
	maxMindDb *maxminddb.Reader
}

func NewStageLocation() StageLocation {
	maxMindDB, err := maxminddb.Open("./ipdb.mmdb")
	if err != nil {
		log.Fatal(err)
	}

	return StageLocation{
		maxMindDb: maxMindDB,
	}
}

// ProcessFrame for StageLocation parses the IP with MaxMindDB and extracts approximate location information (city and country)
func (s StageLocation) ProcessFrame(event *types.ClientEventFrameV1) error {
	if event.IP == "" {
		return nil
	}

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

		event.LocationCountryISO = nullable.FromString(countryCode).Ptr()
		event.LocationCity = nullable.FromString(cityName).Ptr()
	}

	return nil
}
