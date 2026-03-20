package jobs

func priorityToQueue(priority int) string {
	switch {
	case priority >= 90:
		return "critical"
	case priority >= 50:
		return "default"
	default:
		return "low"
	}
}
