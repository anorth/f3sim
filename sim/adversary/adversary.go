package adversary

import "github.com/filecoin-project/go-f3/gpbft"

type Receiver interface {
	gpbft.Receiver
	ID() gpbft.ActorID
	AllowMessage(from gpbft.ActorID, to gpbft.ActorID, msg gpbft.GMessage) bool
}

// Endpoint with which the adversary can control the network
type Host interface {
	gpbft.Host
	// Sends a message to all other participants, immediately.
	// Note that the adversary can subsequently delay delivery to some participants,
	// before messages are actually received.
	RequestSynchronousBroadcast(mb *gpbft.MessageBuilder) error
}

type Generator func(gpbft.ActorID, Host) *Adversary

type Adversary struct {
	Receiver
	Power *gpbft.StoragePower
}
