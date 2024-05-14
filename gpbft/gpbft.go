package gpbft

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"math/big"
	"sort"
	"time"

	"github.com/filecoin-project/go-bitfield"
	rlepluslazy "github.com/filecoin-project/go-bitfield/rle"
	"golang.org/x/xerrors"
)

type Phase uint8

const (
	INITIAL_PHASE Phase = iota
	QUALITY_PHASE
	CONVERGE_PHASE
	PREPARE_PHASE
	COMMIT_PHASE
	DECIDE_PHASE
	TERMINATED_PHASE
)

func (p Phase) String() string {
	switch p {
	case INITIAL_PHASE:
		return "INITIAL"
	case QUALITY_PHASE:
		return "QUALITY"
	case CONVERGE_PHASE:
		return "CONVERGE"
	case PREPARE_PHASE:
		return "PREPARE"
	case COMMIT_PHASE:
		return "COMMIT"
	case DECIDE_PHASE:
		return "DECIDE"
	case TERMINATED_PHASE:
		return "TERMINATED"
	default:
		return "UNKNOWN"
	}
}

const DOMAIN_SEPARATION_TAG = "GPBFT"

// A message in the Granite protocol.
// The same message structure is used for all rounds and phases.
// Note that the message is self-attesting so no separate envelope or signature is needed.
// - The signature field fixes the included sender ID via the implied public key;
// - The signature payload includes all fields a sender can freely choose;
// - The ticket field is a signature of the same public key, so also self-attesting.
type GMessage struct {
	// ID of the sender/signer of this message (a miner actor ID).
	Sender ActorID
	// Vote is the payload that is signed by the signature
	Vote Payload
	// Signature by the sender's public key over Instance || Round || Step || Value.
	Signature []byte
	// VRF ticket for CONVERGE messages (otherwise empty byte array).
	Ticket Ticket
	// Justification for this message (some messages must be justified by a strong quorum of messages from some previous step).
	Justification *Justification
}

type Justification struct {
	// Vote is the payload that is signed by the signature
	Vote Payload
	// Indexes in the base power table of the signers (bitset)
	Signers bitfield.BitField
	// BLS aggregate signature of signers
	Signature []byte
}

// Fields of the message that make up the signature payload.
type Payload struct {
	// GossiPBFT instance (epoch) number.
	Instance uint64
	// GossiPBFT round number.
	Round uint64
	// GossiPBFT step name.
	Step Phase
	// Chain of tipsets proposed/voted for finalisation.
	// Always non-empty; the first entry is the base tipset finalised in the previous instance.
	Value ECChain
}

func (p Payload) Eq(other *Payload) bool {
	return p.Instance == other.Instance &&
		p.Round == other.Round &&
		p.Step == other.Step &&
		p.Value.Eq(other.Value)
}

func (p Payload) MarshalForSigning(nn NetworkName) []byte {
	var buf bytes.Buffer
	buf.WriteString(DOMAIN_SEPARATION_TAG)
	buf.WriteString(":")
	buf.WriteString(string(nn))
	buf.WriteString(":")
	_ = binary.Write(&buf, binary.BigEndian, p.Instance)
	_ = binary.Write(&buf, binary.BigEndian, p.Round)
	_ = binary.Write(&buf, binary.BigEndian, p.Step)
	for _, t := range p.Value {
		_ = binary.Write(&buf, binary.BigEndian, uint32(len(t)))
		buf.Write(t)
	}
	return buf.Bytes()
}

func (m GMessage) String() string {
	return fmt.Sprintf("%s{%d}(%d %s)", m.Vote.Step, m.Vote.Instance, m.Vote.Round, &m.Vote.Value)
}

// A single Granite consensus instance.
type instance struct {
	participant *Participant
	instanceID  uint64
	// The EC chain input to this instance.
	input ECChain
	// The power table for the base chain, used for power in this instance.
	powerTable PowerTable
	// The beacon value from the base chain, used for tickets in this instance.
	beacon []byte
	// Current round number.
	round uint64
	// Current phase in the round.
	phase Phase
	// Time at which the current phase can or must end.
	// For QUALITY, PREPARE, and COMMIT, this is the latest time (the phase can end sooner).
	// For CONVERGE, this is the exact time (the timeout solely defines the phase end).
	phaseTimeout time.Time
	// This instance's proposal for the current round. Never bottom.
	// This is set after the QUALITY phase, and changes only at the end of a full round.
	proposal ECChain
	// The value to be transmitted at the next phase, which may be bottom.
	// This value may change away from the proposal between phases.
	value ECChain
	// The set of values that are acceptable candidates to this instance.
	// This includes the base chain, all prefixes of proposal that found a strong quorum
	// of support in the QUALITY phase, and any chains that could possibly have been
	// decided by another participant.
	candidates []ECChain
	// The final termination value of the instance, for communication to the participant.
	// This field is an alternative to plumbing an optional decision value out through
	// all the method calls, or holding a callback handle to receive it here.
	terminationValue *Justification
	// Queue of messages to be synchronously processed before returning from top-level call.
	inbox []*GMessage
	// Quality phase state (only for round 0)
	quality *quorumState
	// State for each round of phases.
	// State from prior rounds must be maintained to provide justification for values in subsequent rounds.
	rounds map[uint64]*roundState
	// Decision state. Collects DECIDE messages until a decision can be made,
	// independently of protocol phases/rounds.
	decision *quorumState
	// tracer traces logic logs for debugging and simulation purposes.
	tracer Tracer
}

func newInstance(
	participant *Participant,
	instanceID uint64,
	input ECChain,
	powerTable PowerTable,
	beacon []byte) (*instance, error) {
	if input.IsZero() {
		return nil, fmt.Errorf("input is empty")
	}
	return &instance{
		participant: participant,
		instanceID:  instanceID,
		input:       input,
		powerTable:  powerTable,
		beacon:      beacon,
		round:       0,
		phase:       INITIAL_PHASE,
		proposal:    input,
		value:       ECChain{},
		candidates:  []ECChain{input.BaseChain()},
		quality:     newQuorumState(powerTable),
		rounds: map[uint64]*roundState{
			0: newRoundState(powerTable),
		},
		decision: newQuorumState(powerTable),
	}, nil
}

type roundState struct {
	converged *convergeState
	prepared  *quorumState
	committed *quorumState
}

func newRoundState(powerTable PowerTable) *roundState {
	return &roundState{
		converged: newConvergeState(),
		prepared:  newQuorumState(powerTable),
		committed: newQuorumState(powerTable),
	}
}

func (i *instance) Start() error {
	if err := i.beginQuality(); err != nil {
		return err
	}
	return i.drainInbox()
}

// Checks whether a message is valid.
// An invalid message can never become valid, so may be dropped.
// This method is read-only and inspects only immutable state, so should be safe to invoke
// concurrently.
func (i *instance) Validate(msg *GMessage) error {
	return i.validateMessage(msg)
}

// Receives a validated message.
// This method will not attempt to validate the message, the caller must ensure the message
// is valid before calling this method.
func (i *instance) Receive(msg *GMessage) error {
	if i.terminated() {
		return fmt.Errorf("senders message after decision")
	}
	if len(i.inbox) > 0 {
		return fmt.Errorf("senders message while already processing inbox")
	}

	// Enqueue the message for synchronous processing.
	i.enqueueInbox(msg)
	return i.drainInbox()
}

func (i *instance) ReceiveAlarm() error {
	if err := i.tryCompletePhase(); err != nil {
		return fmt.Errorf("failed completing protocol phase: %w", err)
	}

	// A phase may have been successfully completed.
	// Re-process any queued messages for the next phase.
	return i.drainInbox()
}

func (i *instance) Describe() string {
	return fmt.Sprintf("P%d{%d}, round %d, phase %s", i.participant.id, i.instanceID, i.round, i.phase)
}

func (i *instance) enqueueInbox(msg *GMessage) {
	i.inbox = append(i.inbox, msg)
}

func (i *instance) drainInbox() error {
	for len(i.inbox) > 0 {
		// Process one message.
		// Note the message being processed is left in the inbox until after processing,
		// as a signal that this loop is currently draining the inbox.
		if err := i.receiveOne(i.inbox[0]); err != nil {
			return fmt.Errorf("failed receiving message: %w", err)
		}
		i.inbox = i.inbox[1:]
	}

	return nil
}

// Processes a single message.
func (i *instance) receiveOne(msg *GMessage) error {
	if i.phase == TERMINATED_PHASE {
		return nil // No-op
	}
	round := i.roundState(msg.Vote.Round)

	switch msg.Vote.Step {
	case QUALITY_PHASE:
		// Receive each prefix of the proposal independently.
		i.quality.ReceiveEachPrefix(msg.Sender, msg.Vote.Value)
	case CONVERGE_PHASE:
		if err := round.converged.Receive(msg.Sender, msg.Vote.Value, msg.Ticket, msg.Justification); err != nil {
			return fmt.Errorf("failed processing CONVERGE message: %w", err)
		}
	case PREPARE_PHASE:
		round.prepared.Receive(msg.Sender, msg.Vote.Value, msg.Signature)
	case COMMIT_PHASE:
		round.committed.Receive(msg.Sender, msg.Vote.Value, msg.Signature)
		// The only justifications that need to be stored for future propagation are for COMMITs
		// to non-bottom values.
		// This evidence can be brought forward to justify a CONVERGE message in the next round.
		if !msg.Vote.Value.IsZero() {
			round.committed.ReceiveJustification(msg.Vote.Value, msg.Justification)
		}
	case DECIDE_PHASE:
		i.decision.Receive(msg.Sender, msg.Vote.Value, msg.Signature)
		if i.phase != DECIDE_PHASE {
			i.skipToDecide(msg.Vote.Value, msg.Justification)
		}
		if err := i.tryDecide(); err != nil {
			return fmt.Errorf("failed to decide: %w", err)
		}
	default:
		i.log("unexpected message %v", msg)
	}

	// Try to complete the current phase.
	// Every COMMIT phase stays open to new messages even after the protocol moves on to
	// a new round. Late-arriving COMMITS can still (must) cause a local decision, *in that round*.
	if msg.Vote.Step == COMMIT_PHASE && i.phase != DECIDE_PHASE {
		return i.tryCommit(msg.Vote.Round)
	}
	return i.tryCompletePhase()
}

// Attempts to complete the current phase and round.
func (i *instance) tryCompletePhase() error {
	i.log("try step %s", i.phase)
	switch i.phase {
	case QUALITY_PHASE:
		return i.tryQuality()
	case CONVERGE_PHASE:
		return i.tryConverge()
	case PREPARE_PHASE:
		return i.tryPrepare()
	case COMMIT_PHASE:
		return i.tryCommit(i.round)
	case DECIDE_PHASE:
		return i.tryDecide()
	case TERMINATED_PHASE:
		return nil // No-op
	default:
		return fmt.Errorf("unexpected phase %s", i.phase)
	}
}

// Checks message validity, including justification and signatures.
func (i *instance) validateMessage(msg *GMessage) error {
	// Check the message is for this instance.
	// The caller should ensure this is always the case.
	if msg.Vote.Instance != i.instanceID {
		return xerrors.Errorf("message for wrong instance %d, expected %d", msg.Vote.Instance, i.instanceID)
	}
	// Check sender is eligible.
	senderPower, senderPubKey := i.powerTable.Get(msg.Sender)
	if senderPower == nil || senderPower.Sign() == 0 {
		return xerrors.Errorf("sender with zero power or not in power table")
	}

	// Check that message value is a valid chain.
	if err := msg.Vote.Value.Validate(); err != nil {
		return xerrors.Errorf("invalid message vote value chain: %w", err)
	}
	// Check the value is acceptable.
	if !(msg.Vote.Value.IsZero() || msg.Vote.Value.HasBase(i.input.Base())) {
		return xerrors.Errorf("unexpected base %s", &msg.Vote.Value)
	}

	// Check phase-specific constraints.
	switch msg.Vote.Step {
	case INITIAL_PHASE:
		return xerrors.Errorf("invalid vote step: %v", INITIAL_PHASE)
	case QUALITY_PHASE:
		if msg.Vote.Round != 0 {
			return xerrors.Errorf("unexpected round %d for quality phase", msg.Vote.Round)
		}
		if msg.Vote.Value.IsZero() {
			return xerrors.Errorf("unexpected zero value for quality phase")
		}
	case CONVERGE_PHASE:
		if msg.Vote.Round == 0 {
			return xerrors.Errorf("unexpected round 0 for converge phase")
		}
		if msg.Vote.Value.IsZero() {
			return xerrors.Errorf("unexpected zero value for converge phase")
		}
		if !VerifyTicket(i.beacon, i.instanceID, msg.Vote.Round, senderPubKey, i.participant.host, msg.Ticket) {
			return xerrors.Errorf("failed to verify ticket from %v", msg.Sender)
		}
	case DECIDE_PHASE:
		if msg.Vote.Round != 0 {
			return xerrors.Errorf("unexpected non-zero round %d for decide phase", msg.Vote.Round)
		}
		if msg.Vote.Value.IsZero() {
			return xerrors.Errorf("unexpected zero value for decide phase")
		}
	case PREPARE_PHASE, COMMIT_PHASE:
		// No additional checks for PREPARE and COMMIT.
	default:
		return xerrors.Errorf("unknown vote step: %d", msg.Vote.Step)
	}

	// Check vote signature.
	sigPayload := msg.Vote.MarshalForSigning(i.participant.host.NetworkName())
	if err := i.participant.host.Verify(senderPubKey, sigPayload, msg.Signature); err != nil {
		return xerrors.Errorf("invalid signature on %v, %v", msg, err)
	}

	// Check justification
	needsJustification := !(msg.Vote.Step == QUALITY_PHASE ||
		(msg.Vote.Step == PREPARE_PHASE && msg.Vote.Round == 0) ||
		(msg.Vote.Step == COMMIT_PHASE && msg.Vote.Value.IsZero()))
	if needsJustification {
		if msg.Justification == nil {
			return fmt.Errorf("message for phase %v round %v has no justification", msg.Vote.Step, msg.Vote.Round)
		}
		// Check that the justification is for the same instance.
		if msg.Vote.Instance != msg.Justification.Vote.Instance {
			return fmt.Errorf("message with instanceID %v has evidence from instanceID: %v", msg.Vote.Instance, msg.Justification.Vote.Instance)
		}
		// Check that justification vote value is a valid chain.
		if err := msg.Justification.Vote.Value.Validate(); err != nil {
			return xerrors.Errorf("invalid justification vote value chain: %w", err)
		}

		// Check every remaining field of the justification, according to the phase requirements.
		// This map goes from the message phase to the expected justification phase(s),
		// to the required vote values for justification by that phase.
		// Anything else is disallowed.
		expectations := map[Phase]map[Phase]struct {
			Round uint64
			Value ECChain
		}{
			// CONVERGE is justified by a strong quorum of COMMIT for bottom,
			// or a strong quorum of PREPARE for the same value, from the previous round.
			CONVERGE_PHASE: {
				COMMIT_PHASE:  {msg.Vote.Round - 1, ECChain{}},
				PREPARE_PHASE: {msg.Vote.Round - 1, msg.Vote.Value},
			},
			// PREPARE is justified by the same rules as CONVERGE (in rounds > 0).
			PREPARE_PHASE: {
				COMMIT_PHASE:  {msg.Vote.Round - 1, ECChain{}},
				PREPARE_PHASE: {msg.Vote.Round - 1, msg.Vote.Value},
			},
			// COMMIT is justified by strong quorum of PREPARE from the same round with the same value.
			COMMIT_PHASE: {
				PREPARE_PHASE: {msg.Vote.Round, msg.Vote.Value},
			},
			// DECIDE is justified by strong quorum of COMMIT with the same value.
			// The DECIDE message doesn't specify a round.
			DECIDE_PHASE: {
				COMMIT_PHASE: {math.MaxUint64, msg.Vote.Value},
			},
		}

		if expectedPhases, ok := expectations[msg.Vote.Step]; ok {
			if expected, ok := expectedPhases[msg.Justification.Vote.Step]; ok {
				if msg.Justification.Vote.Round != expected.Round && expected.Round != math.MaxUint64 {
					return fmt.Errorf("message %v has justification from wrong round %d", msg, msg.Justification.Vote.Round)
				}
				if !msg.Justification.Vote.Value.Eq(expected.Value) {
					return fmt.Errorf("message %v has justification for a different value: %v", msg, msg.Justification.Vote.Value)
				}
			} else {
				return fmt.Errorf("message %v has justification with unexpected phase: %v", msg, msg.Justification.Vote.Step)
			}
		} else {
			return fmt.Errorf("message %v has unexpected phase for justification", msg)
		}

		// Check justification power and signature.
		justificationPower := NewStoragePower(0)
		signers := make([]PubKey, 0)
		if err := msg.Justification.Signers.ForEach(func(bit uint64) error {
			if int(bit) >= len(i.powerTable.Entries) {
				return fmt.Errorf("invalid signer index: %d", bit)
			}
			justificationPower.Add(justificationPower, i.powerTable.Entries[bit].Power)
			signers = append(signers, i.powerTable.Entries[bit].PubKey)
			return nil
		}); err != nil {
			return fmt.Errorf("failed to iterate over signers: %w", err)
		}

		if !hasStrongQuorum(justificationPower, i.powerTable.Total) {
			return fmt.Errorf("message %v has justification with insufficient power: %v", msg, justificationPower)
		}

		payload := msg.Justification.Vote.MarshalForSigning(i.participant.host.NetworkName())
		if err := i.participant.host.VerifyAggregate(payload, msg.Justification.Signature, signers); err != nil {
			return xerrors.Errorf("verification of the aggregate failed: %+v: %w", msg.Justification, err)
		}
	} else if msg.Justification != nil {
		return fmt.Errorf("message %v has unexpected justification", msg)
	}

	return nil
}

// Sends this node's QUALITY message and begins the QUALITY phase.
func (i *instance) beginQuality() error {
	if i.phase != INITIAL_PHASE {
		return fmt.Errorf("cannot transition from %s to %s", i.phase, QUALITY_PHASE)
	}
	// Broadcast input value and wait up to Δ to receive from others.
	i.phase = QUALITY_PHASE
	i.phaseTimeout = i.alarmAfterSynchrony()
	i.broadcast(i.round, QUALITY_PHASE, i.input, nil, nil)
	return nil
}

// Attempts to end the QUALITY phase and begin PREPARE based on current state.
func (i *instance) tryQuality() error {
	if i.phase != QUALITY_PHASE {
		return fmt.Errorf("unexpected phase %s, expected %s", i.phase, QUALITY_PHASE)
	}
	// Wait either for a strong quorum that agree on our proposal,
	// or for the timeout to expire.
	foundQuorum := i.quality.HasStrongQuorumFor(i.proposal.Key())
	timeoutExpired := atOrAfter(i.participant.host.Time(), i.phaseTimeout)

	if foundQuorum {
		// Keep current proposal.
	} else if timeoutExpired {
		strongQuora := i.quality.ListStrongQuorumValues()
		i.proposal = findFirstPrefixOf(i.proposal, strongQuora)
	}

	if foundQuorum || timeoutExpired {
		// Add prefixes with quorum to candidates (skipping base chain, which is already there).
		for l := range i.proposal {
			if l > 0 {
				i.candidates = append(i.candidates, i.proposal.Prefix(l))
			}
		}
		i.value = i.proposal
		i.log("adopting proposal/value %s", &i.proposal)
		i.beginPrepare(nil)
	}

	return nil
}

func (i *instance) beginConverge() {
	i.phase = CONVERGE_PHASE

	i.phaseTimeout = i.alarmAfterSynchrony()
	prevRoundState := i.roundState(i.round - 1)

	// Proposal was updated at the end of COMMIT phase to be some value for which
	// this node received a COMMIT message (bearing justification), if there were any.
	// If there were none, there must have been a strong quorum for bottom instead.
	var justification *Justification
	if quorum, ok := prevRoundState.committed.FindStrongQuorumFor(""); ok {
		// Build justification for strong quorum of COMMITs for bottom in the previous round.
		justification = i.buildJustification(quorum, i.round-1, COMMIT_PHASE, ECChain{})
	} else {
		// Extract the justification received from some participant (possibly this node itself).
		justification, ok = prevRoundState.committed.receivedJustification[i.proposal.Key()]
		if !ok {
			panic("beginConverge called but no justification for proposal")
		}
	}
	_, pubkey := i.powerTable.Get(i.participant.id)
	ticket, err := MakeTicket(i.beacon, i.instanceID, i.round, pubkey, i.participant.host)
	if err != nil {
		i.log("error while creating VRF ticket: %v", err)
		return
	}

	i.broadcast(i.round, CONVERGE_PHASE, i.proposal, ticket, justification)
}

// Attempts to end the CONVERGE phase and begin PREPARE based on current state.
func (i *instance) tryConverge() error {
	if i.phase != CONVERGE_PHASE {
		return fmt.Errorf("unexpected phase %s, expected %s", i.phase, CONVERGE_PHASE)
	}
	// The CONVERGE phase timeout doesn't wait to hear from >⅔ of power.
	timeoutExpired := atOrAfter(i.participant.host.Time(), i.phaseTimeout)
	if !timeoutExpired {
		return nil
	}

	possibleDecisionLastRound := !i.roundState(i.round - 1).committed.HasStrongQuorumFor("")
	winner := i.roundState(i.round).converged.FindMaxTicketProposal(i.powerTable)
	if winner.Chain.IsZero() {
		return fmt.Errorf("no values at CONVERGE")
	}
	justification := winner.Justification
	// If the winner is not a candidate but it could possibly have been decided by another participant
	// in the last round, consider it a candidate.
	if !i.isCandidate(winner.Chain) && winner.Justification.Vote.Step == PREPARE_PHASE && possibleDecisionLastRound {
		i.log("⚠️ swaying from %s to %s by CONVERGE", &i.proposal, &winner.Chain)
		i.candidates = append(i.candidates, winner.Chain)
	}
	if i.isCandidate(winner.Chain) {
		i.proposal = winner.Chain
		i.log("adopting proposal %s after converge", &winner.Chain)
	} else {
		// Else preserve own proposal.
		fallback, ok := i.roundState(i.round).converged.FindProposalFor(i.proposal)
		if !ok {
			panic("own proposal not found at CONVERGE")
		}
		justification = fallback.Justification
	}
	// NOTE: FIP-0086 says to loop to next lowest ticket, rather than fall back to own proposal.
	// But using own proposal is valid (the spec can't assume any others have been received),
	// considering others is an optimisation.

	i.value = i.proposal
	i.beginPrepare(justification)
	return nil
}

// Sends this node's PREPARE message and begins the PREPARE phase.
func (i *instance) beginPrepare(justification *Justification) {
	// Broadcast preparation of value and wait for everyone to respond.
	i.phase = PREPARE_PHASE
	i.phaseTimeout = i.alarmAfterSynchrony()
	i.broadcast(i.round, PREPARE_PHASE, i.value, nil, justification)
}

// Attempts to end the PREPARE phase and begin COMMIT based on current state.
func (i *instance) tryPrepare() error {
	if i.phase != PREPARE_PHASE {
		return fmt.Errorf("unexpected phase %s, expected %s", i.phase, PREPARE_PHASE)
	}

	prepared := i.roundState(i.round).prepared
	// Optimisation: we could advance phase once a strong quorum on our proposal is not possible.
	foundQuorum := prepared.HasStrongQuorumFor(i.proposal.Key())
	timedOut := atOrAfter(i.participant.host.Time(), i.phaseTimeout) && prepared.ReceivedFromStrongQuorum()

	if foundQuorum {
		i.value = i.proposal
	} else if timedOut {
		i.value = ECChain{}
	}

	if foundQuorum || timedOut {
		i.beginCommit()
	}

	return nil
}

func (i *instance) beginCommit() {
	i.phase = COMMIT_PHASE
	i.phaseTimeout = i.alarmAfterSynchrony()

	// The PREPARE phase exited either with i.value == i.proposal having a strong quorum agreement,
	// or with i.value == bottom otherwise.
	// No justification is required for committing bottom.
	var justification *Justification
	if !i.value.IsZero() {
		if quorum, ok := i.roundState(i.round).prepared.FindStrongQuorumFor(i.value.Key()); ok {
			// Found a strong quorum of PREPARE, build the justification for it.
			justification = i.buildJustification(quorum, i.round, PREPARE_PHASE, i.value)
		} else {
			panic("beginCommit with no strong quorum for non-bottom value")
		}
	}

	i.broadcast(i.round, COMMIT_PHASE, i.value, nil, justification)
}

func (i *instance) tryCommit(round uint64) error {
	// Unlike all other phases, the COMMIT phase stays open to new messages even after an initial quorum is reached,
	// and the algorithm moves on to the next round.
	// A subsequent COMMIT message can cause the node to decide, so there is no check on the current phase.
	committed := i.roundState(round).committed
	quorumValue, foundStrongQuorum := committed.FindStrongQuorumValue()
	timedOut := atOrAfter(i.participant.host.Time(), i.phaseTimeout) && committed.ReceivedFromStrongQuorum()

	if foundStrongQuorum && !quorumValue.IsZero() {
		// A participant may be forced to decide a value that's not its preferred chain.
		// The participant isn't influencing that decision against their interest, just accepting it.
		i.value = quorumValue
		i.beginDecide(round)
	} else if i.round == round && i.phase == COMMIT_PHASE && timedOut {
		if foundStrongQuorum {
			// If there is a strong quorum for bottom, carry forward the existing proposal.
		} else {
			// If there is no strong quorum for bottom, there must be a COMMIT for some other value.
			// There can only be one such value since it must be justified by a strong quorum of PREPAREs.
			// Some other participant could possibly have observed a strong quorum for that value,
			// since they might observe votes from ⅓ of honest power plus a ⅓ equivocating adversary.
			// Sway to consider that value as a candidate, even if it wasn't the local proposal.
			for _, v := range committed.ListAllValues() {
				if !v.IsZero() {
					if !i.isCandidate(v) {
						i.log("⚠️ swaying from %s to %s by COMMIT", &i.input, &v)
						i.candidates = append(i.candidates, v)
					}
					if !v.Eq(i.proposal) {
						i.proposal = v
						i.log("adopting proposal %s after commit", &i.proposal)
					}
					break
				}
			}
		}
		i.beginNextRound()
	}
	return nil
}

func (i *instance) beginDecide(round uint64) {
	i.phase = DECIDE_PHASE
	roundState := i.roundState(round)

	var justification *Justification
	// Value cannot be empty here.
	if quorum, ok := roundState.committed.FindStrongQuorumFor(i.value.Key()); ok {
		// Build justification for strong quorum of COMMITs for the value.
		justification = i.buildJustification(quorum, round, COMMIT_PHASE, i.value)
	} else {
		panic("beginDecide with no strong quorum for value")
	}

	// DECIDE messages always specify round = 0.
	// Extreme out-of-order message delivery could result in different nodes deciding
	// in different rounds (but for the same value).
	// Since each node sends only one DECIDE message, they must share the same vote
	// in order to be aggregated.
	i.broadcast(0, DECIDE_PHASE, i.value, nil, justification)
}

// Skips immediately to the DECIDE phase and sends a DECIDE message
// without waiting for a strong quorum of COMMITs in any round.
// The provided justification must justify the value being decided.
func (i *instance) skipToDecide(value ECChain, justification *Justification) {
	i.phase = DECIDE_PHASE
	i.proposal = value
	i.value = i.proposal
	i.broadcast(0, DECIDE_PHASE, i.value, nil, justification)
}

func (i *instance) tryDecide() error {
	quorumValue, ok := i.decision.FindStrongQuorumValue()
	if ok {
		if quorum, ok := i.decision.FindStrongQuorumFor(quorumValue.Key()); ok {
			decision := i.buildJustification(quorum, 0, DECIDE_PHASE, quorumValue)
			i.terminate(decision)
		} else {
			panic("tryDecide with no strong quorum for value")
		}
	}

	return nil
}

func (i *instance) roundState(r uint64) *roundState {
	round, ok := i.rounds[r]
	if !ok {
		round = newRoundState(i.powerTable)
		i.rounds[r] = round
	}
	return round
}

func (i *instance) beginNextRound() {
	i.round += 1
	i.log("moving to round %d with %s", i.round, i.proposal.String())
	i.beginConverge()
}

// Returns whether a chain is acceptable as a proposal for this instance to vote for.
// This is "EC Compatible" in the pseudocode.
func (i *instance) isCandidate(c ECChain) bool {
	for _, candidate := range i.candidates {
		if c.Eq(candidate) {
			return true
		}
	}
	return false
}

func (i *instance) terminate(decision *Justification) {
	i.log("✅ terminated %s during round %d", &i.value, i.round)
	i.phase = TERMINATED_PHASE
	i.value = decision.Vote.Value
	i.terminationValue = decision
}

func (i *instance) terminated() bool {
	return i.phase == TERMINATED_PHASE
}

func (i *instance) broadcast(round uint64, step Phase, value ECChain, ticket Ticket, justification *Justification) {
	p := Payload{
		Instance: i.instanceID,
		Round:    round,
		Step:     step,
		Value:    value,
	}
	sp := p.MarshalForSigning(i.participant.host.NetworkName())

	sig, err := i.sign(sp)
	if err != nil {
		i.log("error while signing message: %v", err)
		return
	}

	gmsg := &GMessage{
		Sender:        i.participant.id,
		Vote:          p,
		Signature:     sig,
		Ticket:        ticket,
		Justification: justification,
	}
	i.participant.host.Broadcast(gmsg)
	i.enqueueInbox(gmsg)
}

// Sets an alarm to be delivered after a synchrony delay.
// The delay duration increases with each round.
// Returns the absolute time at which the alarm will fire.
func (i *instance) alarmAfterSynchrony() time.Time {
	delta := time.Duration(float64(i.participant.delta) *
		math.Pow(i.participant.deltaBackOffExponent, float64(i.round)))
	timeout := i.participant.host.Time().Add(2 * delta)
	i.participant.host.SetAlarm(timeout)
	return timeout
}

// Builds a justification for a value from a quorum result.
func (i *instance) buildJustification(quorum QuorumResult, round uint64, phase Phase, value ECChain) *Justification {
	aggSignature, err := quorum.Aggregate(i.participant.host)
	if err != nil {
		panic(xerrors.Errorf("aggregating for phase %v: %v", phase, err))
	}
	return &Justification{
		Vote: Payload{
			Instance: i.instanceID,
			Round:    round,
			Step:     phase,
			Value:    value,
		},
		Signers:   quorum.SignersBitfield(),
		Signature: aggSignature,
	}
}

func (i *instance) log(format string, args ...interface{}) {
	if i.tracer != nil {
		msg := fmt.Sprintf(format, args...)
		i.tracer.Log("P%d{%d}: %s (round %d, step %s, proposal %s, value %s)", i.participant.id, i.instanceID, msg,
			i.round, i.phase, &i.proposal, &i.value)
	}
}

func (i *instance) sign(msg []byte) ([]byte, error) {
	_, pubKey := i.powerTable.Get(i.participant.id)
	return i.participant.host.Sign(pubKey, msg)
}

///// Incremental quorum-calculation helper /////

// Accumulates values from a collection of senders and incrementally calculates
// which values have reached a strong quorum of support.
// Supports receiving multiple values from a sender at once, and hence multiple strong quorum values.
// Subsequent messages from a single sender are dropped.
type quorumState struct {
	// Set of senders from which a message has been received.
	senders map[ActorID]struct{}
	// Total power of all distinct senders from which some chain has been received so far.
	sendersTotalPower *StoragePower
	// The power supporting each chain so far.
	chainSupport map[ChainKey]chainSupport
	// Table of senders' power.
	powerTable PowerTable
	// Stores justifications received for some value.
	receivedJustification map[ChainKey]*Justification
}

// A chain value and the total power supporting it
type chainSupport struct {
	chain           ECChain
	power           *StoragePower
	signatures      map[ActorID][]byte
	hasStrongQuorum bool
	hasWeakQuorum   bool
}

// Creates a new, empty quorum state.
func newQuorumState(powerTable PowerTable) *quorumState {
	return &quorumState{
		senders:               map[ActorID]struct{}{},
		sendersTotalPower:     NewStoragePower(0),
		chainSupport:          map[ChainKey]chainSupport{},
		powerTable:            powerTable,
		receivedJustification: map[ChainKey]*Justification{},
	}
}

// Receives a chain from a sender.
// Ignores any subsequent value from a sender from which a value has already been received.
func (q *quorumState) Receive(sender ActorID, value ECChain, signature []byte) {
	senderPower, ok := q.receiveSender(sender)
	if !ok {
		return
	}
	q.receiveInner(sender, value, senderPower, signature)
}

// Receives each prefix of a chain as a distinct value from a sender.
// Note that this method does not store signatures, so it is not possible later to
// create an aggregate for these prefixes.
// This is intended for use in the QUALITY phase.
// Ignores any subsequent values from a sender from which a value has already been received.
func (q *quorumState) ReceiveEachPrefix(sender ActorID, values ECChain) {
	senderPower, ok := q.receiveSender(sender)
	if !ok {
		return
	}
	for j := range values.Suffix() {
		prefix := values.Prefix(j + 1)
		q.receiveInner(sender, prefix, senderPower, nil)
	}
}

// Adds sender's power to total the first time a value is received from them.
// Returns the sender's power, and whether this was the first invocation for this sender.
func (q *quorumState) receiveSender(sender ActorID) (*StoragePower, bool) {
	if _, found := q.senders[sender]; found {
		return nil, false
	}
	q.senders[sender] = struct{}{}
	senderPower, _ := q.powerTable.Get(sender)
	q.sendersTotalPower.Add(q.sendersTotalPower, senderPower)
	return senderPower, true
}

// Receives a chain from a sender.
func (q *quorumState) receiveInner(sender ActorID, value ECChain, power *StoragePower, signature []byte) {
	key := value.Key()
	candidate, ok := q.chainSupport[key]
	if !ok {
		candidate = chainSupport{
			chain:           value,
			power:           NewStoragePower(0),
			signatures:      map[ActorID][]byte{},
			hasStrongQuorum: false,
			hasWeakQuorum:   false,
		}
	}

	candidate.power.Add(candidate.power, power)
	if candidate.signatures[sender] != nil {
		panic("duplicate message should have been dropped")
	}
	candidate.signatures[sender] = signature
	candidate.hasStrongQuorum = hasStrongQuorum(candidate.power, q.powerTable.Total)
	candidate.hasWeakQuorum = hasWeakQuorum(candidate.power, q.powerTable.Total)
	q.chainSupport[key] = candidate
}

// Receives and stores justification for a value from another participant.
func (q *quorumState) ReceiveJustification(value ECChain, justification *Justification) {
	if justification == nil {
		panic("nil justification")
	}
	// Keep only the first one received.
	key := value.Key()
	if _, ok := q.receivedJustification[key]; !ok {
		q.receivedJustification[key] = justification
	}
}

// Lists all values that have been senders from any sender.
// The order of returned values is not defined.
func (q *quorumState) ListAllValues() []ECChain {
	var chains []ECChain
	for _, cp := range q.chainSupport {
		chains = append(chains, cp.chain)
	}
	return chains
}

// Checks whether at least one message has been senders from a strong quorum of senders.
func (q *quorumState) ReceivedFromStrongQuorum() bool {
	return hasStrongQuorum(q.sendersTotalPower, q.powerTable.Total)
}

// Checks whether a chain has reached a strong quorum.
func (q *quorumState) HasStrongQuorumFor(key ChainKey) bool {
	supportForChain, ok := q.chainSupport[key]
	return ok && supportForChain.hasStrongQuorum
}

type QuorumResult struct {
	// Signers is an array of indexes into the powertable, sorted in increasing order
	Signers    []int
	PubKeys    []PubKey
	Signatures [][]byte
}

func (q QuorumResult) Aggregate(v Verifier) ([]byte, error) {
	return v.Aggregate(q.PubKeys, q.Signatures)
}

func (q QuorumResult) SignersBitfield() bitfield.BitField {
	signers := make([]uint64, 0, len(q.Signers))
	for _, s := range q.Signers {
		signers = append(signers, uint64(s))
	}
	ri, _ := rlepluslazy.RunsFromSlice(signers)
	bf, _ := bitfield.NewFromIter(ri)
	return bf
}

// Checks whether a chain has reached a strong quorum.
// If so returns a set of signers and signatures for the value that form a strong quorum.
func (q *quorumState) FindStrongQuorumFor(key ChainKey) (QuorumResult, bool) {
	chainSupport, ok := q.chainSupport[key]
	if !ok || !chainSupport.hasStrongQuorum {
		return QuorumResult{}, false
	}

	// Build an array of indices of signers in the power table.
	signers := make([]int, 0, len(chainSupport.signatures))
	for id := range chainSupport.signatures {
		signers = append(signers, q.powerTable.Lookup[id])
	}
	// Sort power table indices.
	// If the power table entries are ordered by decreasing power,
	// then the first strong quorum found will be the smallest.
	sort.Ints(signers)

	// Accumulate signers and signatures until they reach a strong quorum.
	signatures := make([][]byte, 0, len(chainSupport.signatures))
	pubkeys := make([]PubKey, 0, len(signatures))
	justificationPower := NewStoragePower(0)
	for i, idx := range signers {
		if idx >= len(q.powerTable.Entries) {
			panic(fmt.Sprintf("invalid signer index: %d for %d entries", idx, len(q.powerTable.Entries)))
		}
		entry := q.powerTable.Entries[idx]
		justificationPower.Add(justificationPower, entry.Power)
		signatures = append(signatures, chainSupport.signatures[entry.ID])
		pubkeys = append(pubkeys, entry.PubKey)
		if hasStrongQuorum(justificationPower, q.powerTable.Total) {
			return QuorumResult{
				Signers:    signers[:i+1],
				PubKeys:    pubkeys,
				Signatures: signatures,
			}, true
		}
	}

	return QuorumResult{}, false
}

// Checks whether a chain has reached weak quorum.
func (q *quorumState) HasWeakQuorumFor(key ChainKey) bool {
	cp, ok := q.chainSupport[key]
	return ok && cp.hasWeakQuorum
}

// Returns a list of the chains which have reached an agreeing strong quorum.
// Chains are returned in descending length order.
// This is appropriate for use in the QUALITY phase, where each participant
// votes for every prefix of their preferred chain.
// Panics if there are multiple chains of the same length with strong quorum
// (signalling a violation of assumptions about the adversary).
func (q *quorumState) ListStrongQuorumValues() []ECChain {
	var withQuorum []ECChain
	for key, cp := range q.chainSupport {
		if cp.hasStrongQuorum {
			withQuorum = append(withQuorum, q.chainSupport[key].chain)
		}
	}
	sort.Slice(withQuorum, func(i, j int) bool {
		return len(withQuorum[i]) > len(withQuorum[j])
	})
	prevLength := 0
	for _, v := range withQuorum {
		if len(v) == prevLength {
			panic(fmt.Sprintf("multiple chains of length %d with strong quorum", prevLength))
		}
		prevLength = len(v)
	}
	return withQuorum
}

// Returns the chain with a strong quorum of support, if there is one.
// This is appropriate for use in PREPARE/COMMIT/DECIDE phases, where each participant
// casts a single vote.
// Panics if there are multiple chains with strong quorum
// (signalling a violation of assumptions about the adversary).
func (q *quorumState) FindStrongQuorumValue() (quorumValue ECChain, foundQuorum bool) {
	for key, cp := range q.chainSupport {
		if cp.hasStrongQuorum {
			if foundQuorum {
				panic("multiple chains with strong quorum")
			}
			foundQuorum = true
			quorumValue = q.chainSupport[key].chain
		}
	}
	return
}

//// CONVERGE phase helper /////

type convergeState struct {
	// Participants from which a message has been received.
	senders map[ActorID]struct{}
	// Chains indexed by key.
	values map[ChainKey]ConvergeValue
	// Tickets provided by proposers of each chain.
	tickets map[ChainKey][]ConvergeTicket
}

type ConvergeValue struct {
	Chain         ECChain
	Justification *Justification
}

type ConvergeTicket struct {
	Sender ActorID
	Ticket Ticket
}

func newConvergeState() *convergeState {
	return &convergeState{
		senders: map[ActorID]struct{}{},
		values:  map[ChainKey]ConvergeValue{},
		tickets: map[ChainKey][]ConvergeTicket{},
	}
}

// Receives a new CONVERGE value from a sender.
// Ignores any subsequent value from a sender from which a value has already been received.
func (c *convergeState) Receive(sender ActorID, value ECChain, ticket Ticket, justification *Justification) error {
	if _, ok := c.senders[sender]; ok {
		return nil
	}
	c.senders[sender] = struct{}{}
	if value.IsZero() {
		return fmt.Errorf("bottom cannot be justified for CONVERGE")
	}
	key := value.Key()

	// Keep only the first justification and ticket received for a value.
	if _, found := c.values[key]; !found {
		c.values[key] = ConvergeValue{Chain: value, Justification: justification}
		c.tickets[key] = append(c.tickets[key], ConvergeTicket{Sender: sender, Ticket: ticket})
	}
	return nil
}

// Returns the value with the highest ticket, weighted by sender power.
// Non-determinism here (in case of matching tickets from equivocation) is ok.
// If the same ticket is used for two different values then either we get a decision on one of them
// only or we go to a new round. Eventually there is a round where the max ticket is held by a
// correct participant, who will not double vote.
func (c *convergeState) FindMaxTicketProposal(table PowerTable) ConvergeValue {
	var maxTicket *big.Int
	var maxValue ConvergeValue

	for key, value := range c.values {
		for _, ticket := range c.tickets[key] {
			senderPower, _ := table.Get(ticket.Sender)
			ticketAsInt := new(big.Int).SetBytes(ticket.Ticket)
			weightedTicket := new(big.Int).Mul(ticketAsInt, senderPower)
			if maxTicket == nil || weightedTicket.Cmp(maxTicket) > 0 {
				maxTicket = weightedTicket
				maxValue = value
			}
		}
	}
	return maxValue
}

// Finds some proposal which matches a specific value.
func (c *convergeState) FindProposalFor(chain ECChain) (ConvergeValue, bool) {
	for _, value := range c.values {
		if value.Chain.Eq(chain) {
			return value, true
		}
	}
	return ConvergeValue{}, false
}

///// General helpers /////

// Returns the first candidate value that is a prefix of the preferred value, or the base of preferred.
func findFirstPrefixOf(preferred ECChain, candidates []ECChain) ECChain {
	for _, v := range candidates {
		if preferred.HasPrefix(v) {
			return v
		}
	}

	// No candidates are a prefix of preferred.
	return preferred.BaseChain()
}

// Check whether a portion of storage power is a strong quorum of the total
func hasStrongQuorum(part, total *StoragePower) bool {
	two := NewStoragePower(2)
	three := NewStoragePower(3)

	strongThreshold := new(StoragePower).Mul(total, two)
	strongThreshold.Div(strongThreshold, three)
	return part.Cmp(strongThreshold) > 0
}

// Check whether a portion of storage power is a weak quorum of the total
func hasWeakQuorum(part, total *StoragePower) bool {
	three := NewStoragePower(3)

	weakThreshold := new(StoragePower).Div(total, three)
	return part.Cmp(weakThreshold) > 0
}

// Tests whether lhs is equal to or greater than rhs.
func atOrAfter(lhs time.Time, rhs time.Time) bool {
	return lhs.After(rhs) || lhs.Equal(rhs)
}
