package pipelinedb

type PayoutGroupStatus string

const (
	PayoutGroupSkipped  = PayoutGroupStatus("skipped")
	PayoutGroupComplete = PayoutGroupStatus("complete")
)
