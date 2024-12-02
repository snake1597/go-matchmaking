package enum

type BroadcastAction int

const (
	BroadcastActionUnspecified BroadcastAction = iota
	BroadcastActionPublic
	BroadcastActionJoin
)
