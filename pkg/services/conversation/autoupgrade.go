package conversation

// gradeRank returns 0 / 1 / 2 for LGTM / DIGESTED / GRILLED so callers can
// compare floors monotonically. Any unrecognized grade returns -1 (never
// clears an unknown grade during auto-upgrade).
func gradeRank(g string) int {
	switch g {
	case "LGTM":
		return 0
	case "DIGESTED":
		return 1
	case "GRILLED":
		return 2
	default:
		return -1
	}
}

// autoFloorForCount maps a user-message count to the corresponding auto
// grade floor. Aligns with the v0.12 spec: 3+ → GRILLED, 1+ → DIGESTED,
// otherwise LGTM.
func autoFloorForCount(count int) string {
	switch {
	case count >= 3:
		return "GRILLED"
	case count >= 1:
		return "DIGESTED"
	default:
		return "LGTM"
	}
}
