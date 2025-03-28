package log

import (
	"fmt"

	"github.com/drycc/logger/storage"
)

// NewAggregator returns a pointer to an appropriate implementation of the Aggregator interface, as
// determined by the aggregatorType string it is passed.
func NewAggregator(aggregatorType string, storageAdapter storage.Adapter) (Aggregator, error) {
	if aggregatorType == "valkey" {
		return newValkeyAggregator(storageAdapter), nil
	}
	return nil, fmt.Errorf("unrecognized aggregator type: '%s'", aggregatorType)
}
