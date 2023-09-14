package watch

// EventProcessor to process data
type EventProcessor interface {
	ProcessEventData(data []byte)
}
