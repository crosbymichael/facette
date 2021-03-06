// Package catalog implements the service catalog handling immutable data: origins, sources and metrics.
package catalog

import (
	"fmt"

	"github.com/facette/facette/pkg/logger"
)

// Catalog represents the main structure of a catalog instance.
type Catalog struct {
	Origins    map[string]*Origin
	RecordChan chan *Record
}

// Record represents a catalog record.
type Record struct {
	Origin         string
	Source         string
	Metric         string
	OriginalOrigin string
	OriginalSource string
	OriginalMetric string
	Connector      interface{}
}

func (r Record) String() string {
	return fmt.Sprintf("{Origin: \"%s\", Source: \"%s\", Metric: \"%s\"}", r.Origin, r.Source, r.Metric)
}

// NewCatalog creates a new instance of catalog.
func NewCatalog() *Catalog {
	return &Catalog{
		Origins:    make(map[string]*Origin),
		RecordChan: make(chan *Record),
	}
}

// Insert inserts a new record in the catalog.
func (catalog *Catalog) Insert(record *Record) {
	logger.Log(
		logger.LevelDebug,
		"catalog",
		"appending metric `%s' to source `%s' via origin `%s'",
		record.Metric,
		record.Source,
		record.Origin,
	)

	if _, ok := catalog.Origins[record.Origin]; !ok {
		catalog.Origins[record.Origin] = NewOrigin(
			record.Origin,
			record.OriginalOrigin,
			catalog,
		)
	}

	if _, ok := catalog.Origins[record.Origin].Sources[record.Source]; !ok {
		catalog.Origins[record.Origin].Sources[record.Source] = NewSource(
			record.Source,
			record.OriginalSource,
			catalog.Origins[record.Origin],
		)
	}

	if _, ok := catalog.Origins[record.Origin].Sources[record.Source].Metrics[record.Metric]; !ok {
		catalog.Origins[record.Origin].Sources[record.Source].Metrics[record.Metric] = NewMetric(
			record.Metric,
			record.OriginalMetric,
			catalog.Origins[record.Origin].Sources[record.Source],
			record.Connector,
		)
	}
}

// GetMetric returns an existing metric entry based on its origin, source and name.
func (catalog *Catalog) GetMetric(origin, source, name string) *Metric {
	if _, ok := catalog.Origins[origin]; !ok {
		return nil
	} else if _, ok := catalog.Origins[origin].Sources[source]; !ok {
		return nil
	} else if _, ok := catalog.Origins[origin].Sources[source].Metrics[name]; !ok {
		return nil
	}

	return catalog.Origins[origin].Sources[source].Metrics[name]
}

// Close closes a catalog instance.
func (catalog *Catalog) Close() error {
	close(catalog.RecordChan)

	return nil
}
