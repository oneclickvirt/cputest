package cpu

import "time"

const (
	// DefaultBurnDuration is used when a burn has been requested without a
	// duration. MaxBurnDuration is the hard upper bound for every burn call.
	DefaultBurnDuration = 20 * time.Second
	MaxBurnDuration     = 20 * time.Second

	DefaultBurnMaxPrime    = 50000
	DefaultStructuredPrime = 10000
	// MaxPrimeLimit prevents int overflow and unbounded work from untrusted
	// or accidentally propagated configuration values.
	MaxPrimeLimit = 1000000
)

func normalizeMaxPrime(value, fallback int) int {
	if value <= 0 {
		value = fallback
	}
	if value > MaxPrimeLimit {
		return MaxPrimeLimit
	}
	return value
}
