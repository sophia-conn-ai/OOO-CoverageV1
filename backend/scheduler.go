package main

import (
	"fmt"
	"log"
	"time"
)

// ScheduleSyncs fires a background Sync at 12:00am, 8:00am, 4:00pm PST.
func ScheduleSyncs(gh *GreenhouseClient) {
	for {
		next := nextSyncTime()
		label := next.In(pstLocation()).Format("3:04pm MST")
		dur := time.Until(next)
		log.Printf("Next scheduled sync: %s (in %s)\n", label, dur.Round(time.Minute))

		time.Sleep(dur)

		log.Printf("Scheduled sync triggered (%s)", label)
		go gh.Sync()
	}
}

func nextSyncTime() time.Time {
	now := time.Now().UTC()
	pst := now.In(pstLocation())
	h := pst.Hour()*60 + pst.Minute()

	for _, syncH := range syncHoursPST {
		if syncH*60 > h {
			return time.Date(pst.Year(), pst.Month(), pst.Day(), syncH, 0, 0, 0, pstLocation()).UTC()
		}
	}
	// All today's syncs passed — first sync tomorrow
	tomorrow := pst.AddDate(0, 0, 1)
	return time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), syncHoursPST[0], 0, 0, 0, pstLocation()).UTC()
}

func pstLocation() *time.Location {
	loc, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		// Fallback to fixed PST offset if timezone data not available
		loc = time.FixedZone("PST", pstOffsetHours*3600)
	}
	return loc
}

// Ensure fmt is used
var _ = fmt.Sprintf
