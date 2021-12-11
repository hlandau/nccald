// Package types provides supplemental types for nccald.
package types

import "time"

// ExtraNameInfo provides extra information corresponding to a single name_show
// result.
type ExtraNameInfo struct {
	EstimatedExpiryTime time.Time
	ExpiryHeight        int32
}
