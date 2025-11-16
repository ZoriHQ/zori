package tiles

func calculateTotalVisits(dataPoints []TimelineTileData) uint64 {
	var total uint64
	for _, dp := range dataPoints {
		total += dp.NumDesktopVisits + dp.NumMobileVisits + dp.NumUnknownVisits
	}
	return total
}
