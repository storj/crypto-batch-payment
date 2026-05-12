package payouts2

type InitUI interface {
	Started(StartedEvent)
	CSVLoaded(CSVLoadedEvent)
	RowAggregated(RowAggregatedEvent)
	RowSkipped(RowSkippedEvent)
	RowsAggregated(RowsAggregatedEvent)
	CSVsLoaded(CSVsLoadedEvent)
}

type StartedEvent struct {
	CSVPaths []string
}

type CSVLoadedEvent struct {
	CSVPath string
	Err     error
	NumRows int
}

type RowAggregatedEvent struct {
	CSVPath string
	Line    int
}

type RowSkippedEvent struct {
	CSVPath string
	Line    int
	Reason  RowSkipReason
}

type RowsAggregatedEvent struct {
	CSVPath string
}

type CSVsLoadedEvent struct {
	ByType ByType
}

type RowSkipReason int

const (
	RowInvalid RowSkipReason = iota
	RowSanctioned
	// RowSkipReasonMax must remain at the end.
	RowSkipReasonMax
)

func (r RowSkipReason) String() string {
	switch r {
	case RowInvalid:
		return "invalid"
	case RowSanctioned:
		return "sanctioned"
	default:
		return "unknown"
	}
}
