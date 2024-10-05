package internal

import "github.com/vingarcia/insights/internal/adapters/evaluator"

type DataSourceRepo interface {
	FindByName(name string) DataSource
}

type DataSource struct {
	Name string
	Type string
	Read func() map[string]any
}

type Query struct {
	From    string
	Where   evaluator.Expression
	GroupBy GroupBy
}

type GroupBy struct {
	Keys         []string
	Aggregations []evaluator.Expression
}
