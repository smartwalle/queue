package delay

const (
	kModeTypeDefault = 0
	kModeTypeReadAll = 1
)

type mode[T any] interface {
	dequeue(dq *delayQueue[T]) (T, int64)

	close(dq *delayQueue[T])
}

func getMode[T any](mType int) mode[T] {
	switch mType {
	case kModeTypeReadAll:
		return newReadAllMode[T]()
	default:
		return newDefaultMode[T]()
	}
}
