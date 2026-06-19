package vmstats

const (
	nanosecondsPerSecond float64 = 1_000_000_000
	bytesPerKibibyte     float64 = 1024
)

func nanosecondsToSeconds(ns uint64) float64 {
	return float64(ns) / nanosecondsPerSecond
}

func kibibytesToBytes(kibibytes uint64) float64 {
	return float64(kibibytes) * bytesPerKibibyte
}

func humanReadableVCPUState(state int) string {
	switch state {
	case VCPUOffline:
		return "offline"
	case VCPUBlocked:
		return "blocked"
	case VCPURunning:
		return "running"
	default:
		return "unknown"
	}
}
