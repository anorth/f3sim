package f3

import (
	"bytes"
	"context"
	"errors"
	"slices"
	"time"

	"github.com/filecoin-project/go-f3/certs"
	"github.com/filecoin-project/go-f3/certstore"
	"github.com/filecoin-project/go-f3/ec"
	"github.com/filecoin-project/go-f3/gpbft"
	"github.com/filecoin-project/go-f3/manifest"
	logging "github.com/ipfs/go-log/v2"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	peer "github.com/libp2p/go-libp2p/core/peer"
	"go.uber.org/multierr"
	"golang.org/x/sync/errgroup"
	"golang.org/x/xerrors"
)

type BroadcastMessage func(*gpbft.MessageBuilder)

// gpbftRunner is responsible for running gpbft.Participant, taking in all concurrent events and
// passing them to gpbft in a single thread.
type gpbftRunner struct {
	certStore         *certstore.Store
	manifest          *manifest.Manifest
	ec                ec.Backend
	pubsub            *pubsub.PubSub
	signingMarshaller gpbft.SigningMarshaler
	verifier          gpbft.Verifier
	broadcastCb       BroadcastMessage
	log, logWithSkip  Logger

	participant *gpbft.Participant
	topic       *pubsub.Topic

	alertTimer *time.Timer

	runningCtx context.Context
	errgrp     *errgroup.Group
	ctxCancel  context.CancelFunc
}

func newRunner(
	_ context.Context,
	cs *certstore.Store,
	ec ec.Backend,
	log Logger,
	ps *pubsub.PubSub,
	signer gpbft.SigningMarshaler,
	verifier gpbft.Verifier,
	broadcastCb BroadcastMessage,
	m *manifest.Manifest,
) (*gpbftRunner, error) {
	runningCtx, ctxCancel := context.WithCancel(context.Background())
	errgrp, runningCtx := errgroup.WithContext(runningCtx)

	runner := &gpbftRunner{
		certStore:         cs,
		manifest:          m,
		ec:                ec,
		pubsub:            ps,
		signingMarshaller: signer,
		verifier:          verifier,
		broadcastCb:       broadcastCb,
		log:               log,
		logWithSkip:       log,
		runningCtx:        runningCtx,
		errgrp:            errgrp,
		ctxCancel:         ctxCancel,
	}

	if zapLogger, ok := runner.log.(*logging.ZapEventLogger); ok {
		runner.logWithSkip = logging.WithSkip(zapLogger, 1)
	}

	// create a stopped timer to facilitate alerts requested from gpbft
	runner.alertTimer = time.NewTimer(100 * time.Hour)
	if !runner.alertTimer.Stop() {
		<-runner.alertTimer.C
	}

	runner.log.Infof("Starting gpbft runner")
	opts := append(m.GpbftOptions(), gpbft.WithTracer((*gpbftTracer)(runner)))
	p, err := gpbft.NewParticipant((*gpbftHost)(runner), opts...)
	if err != nil {
		return nil, xerrors.Errorf("creating participant: %w", err)
	}
	runner.participant = p
	return runner, nil
}

func (h *gpbftRunner) Start(ctx context.Context) (_err error) {
	defer func() {
		if _err != nil {
			_err = multierr.Append(_err, h.Stop(ctx))
		}
	}()

	startInstance := h.manifest.InitialInstance
	if latest := h.certStore.Latest(); latest != nil {
		startInstance = latest.GPBFTInstance + 1
	}

	messageQueue, err := h.startPubsub()
	if err != nil {
		return err
	}

	if err := h.participant.StartInstanceAt(startInstance, time.Now()); err != nil {
		return xerrors.Errorf("starting a participant: %w", err)
	}

	// Subscribe to new certificates. We don't bother canceling the subscription as that'll
	// happen automatically when the channel fills.
	finalityCertificates := make(chan *certs.FinalityCertificate, 4)
	_, _ = h.certStore.SubscribeForNewCerts(finalityCertificates)

	h.errgrp.Go(func() (_err error) {
		defer func() {
			if _err != nil && h.runningCtx.Err() == nil {
				h.log.Errorf("exited GPBFT runner early: %+v", _err)
			}
		}()
		for h.runningCtx.Err() == nil {
			// prioritise finality certificates and alarm delivery
			select {
			case c, ok := <-finalityCertificates:
				if !ok {
					finalityCertificates = make(chan *certs.FinalityCertificate, 4)
					c, _ = h.certStore.SubscribeForNewCerts(finalityCertificates)
				}
				if err := h.receiveCertificate(c); err != nil {
					return err
				}
				continue
			case <-h.alertTimer.C:
				if err := h.participant.ReceiveAlarm(); err != nil {
					return err
				}
				continue
			default:
			}

			// Handle messages, finality certificates, and alarms
			select {
			case c, ok := <-finalityCertificates:
				if !ok {
					finalityCertificates = make(chan *certs.FinalityCertificate, 4)
					c, _ = h.certStore.SubscribeForNewCerts(finalityCertificates)
				}
				if err := h.receiveCertificate(c); err != nil {
					return err
				}
			case <-h.alertTimer.C:
				if err := h.participant.ReceiveAlarm(); err != nil {
					return err
				}
			case msg, ok := <-messageQueue:
				if !ok {
					return xerrors.Errorf("incoming message queue closed")
				}
				if err := h.participant.ReceiveMessage(msg); err != nil {
					// We silently drop failed messages because GPBFT will
					// return errors for, e.g., messages from old instances.
					// Given the async nature of our pubsub message handling, we
					// could easily receive these.
					h.log.Debugf("error when processing message: %+v", err)
				}
			case <-h.runningCtx.Done():
				return nil
			}

		}
		return nil
	})
	return nil
}

func (h *gpbftRunner) receiveCertificate(c *certs.FinalityCertificate) error {
	nextInstance := c.GPBFTInstance + 1
	if h.participant.CurrentInstance() <= nextInstance {
		return nil
	}

	nextInstanceStart := h.computeNextInstanceStart(c)
	return h.participant.StartInstanceAt(nextInstance, nextInstanceStart)
}

func (h *gpbftRunner) computeNextInstanceStart(cert *certs.FinalityCertificate) time.Time {
	ecDelay := time.Duration(h.manifest.ECDelayMultiplier * float64(h.manifest.ECPeriod))

	ts, err := h.ec.GetTipset(h.runningCtx, cert.ECChain.Head().Key)
	if err != nil {
		// this should not happen
		h.log.Errorf("could not get timestamp of just finalized tipset: %+v", err)
		return time.Now().Add(ecDelay)
	}

	if cert.ECChain.HasSuffix() {
		// we decided on something new, the tipset that got finalized can at minimum be 30-60s old.
		return ts.Timestamp().Add(ecDelay)
	}
	if cert.GPBFTInstance == h.manifest.InitialInstance {
		// if we are at initial instance, there is no history to look at
		return ts.Timestamp().Add(ecDelay)
	}
	backoffTable := h.manifest.BaseDecisionBackoffTable

	attempts := 0
	backoffMultipler := 1.0 // to account for the one ECDelay after which we got the base decistion
	for instance := cert.GPBFTInstance - 1; instance > h.manifest.InitialInstance; instance-- {
		cert, err := h.certStore.Get(h.runningCtx, instance)
		if err != nil {
			h.log.Errorf("error while getting instance %d from certstore: %+v", instance, err)
			break
		}
		if !cert.ECChain.HasSuffix() {
			attempts += 1
		}
		if attempts < len(backoffTable) {
			backoffMultipler += backoffTable[attempts]
		} else {
			// if we are beyond backoffTable, reuse the last element
			backoffMultipler += backoffTable[len(backoffTable)-1]
		}
	}

	backoff := time.Duration(float64(ecDelay) * backoffMultipler)
	h.log.Infof("backing off for: %v", backoff)

	return ts.Timestamp().Add(backoff)
}

// Sends a message to all other participants.
// The message's sender must be one that the network interface can sign on behalf of.
func (h *gpbftRunner) BroadcastMessage(msg *gpbft.GMessage) error {
	if h.topic == nil {
		return pubsub.ErrTopicClosed
	}
	var bw bytes.Buffer
	err := msg.MarshalCBOR(&bw)
	if err != nil {
		return xerrors.Errorf("marshalling GMessage for broadcast: %w", err)
	}

	err = h.topic.Publish(h.runningCtx, bw.Bytes())
	if err != nil {
		return xerrors.Errorf("publishing message: %w", err)
	}
	return nil
}

var _ pubsub.ValidatorEx = (*gpbftRunner)(nil).validatePubsubMessage

func (h *gpbftRunner) validatePubsubMessage(ctx context.Context, pID peer.ID,
	msg *pubsub.Message) pubsub.ValidationResult {
	var gmsg gpbft.GMessage
	err := gmsg.UnmarshalCBOR(bytes.NewReader(msg.Data))
	if err != nil {
		return pubsub.ValidationReject
	}

	validatedMessage, err := h.participant.ValidateMessage(&gmsg)
	if errors.Is(err, gpbft.ErrValidationInvalid) {
		h.log.Debugf("validation error during validation: %+v", err)
		return pubsub.ValidationReject
	}
	if err != nil {
		h.log.Warnf("unknown error during validation: %+v", err)
		return pubsub.ValidationIgnore
	}
	msg.ValidatorData = validatedMessage
	return pubsub.ValidationAccept
}

func (h *gpbftRunner) setupPubsub() error {
	pubsubTopicName := h.manifest.PubSubTopic()
	err := h.pubsub.RegisterTopicValidator(pubsubTopicName, h.validatePubsubMessage)
	if err != nil {
		return xerrors.Errorf("registering topic validator: %w", err)
	}

	topic, err := h.pubsub.Join(pubsubTopicName)
	if err != nil {
		return xerrors.Errorf("could not join on pubsub topic: %s: %w", pubsubTopicName, err)
	}
	h.topic = topic
	return nil
}

func (h *gpbftRunner) teardownPubsub() error {
	var err error
	if h.topic != nil {
		err = multierr.Combine(
			h.topic.Close(),
			h.pubsub.UnregisterTopicValidator(h.topic.String()),
		)

		if errors.Is(err, context.Canceled) {
			err = nil
		}
	}
	return err
}

func (h *gpbftRunner) startPubsub() (<-chan gpbft.ValidatedMessage, error) {
	if err := h.setupPubsub(); err != nil {
		return nil, err
	}

	sub, err := h.topic.Subscribe()
	if err != nil {
		return nil, xerrors.Errorf("could not subscribe to pubsub topic: %s: %w", sub.Topic(), err)
	}

	messageQueue := make(chan gpbft.ValidatedMessage, 20)
	h.errgrp.Go(func() error {
		defer func() {
			sub.Cancel()
			close(messageQueue)
		}()

		for h.runningCtx.Err() == nil {
			var msg *pubsub.Message
			msg, err := sub.Next(h.runningCtx)
			if err != nil {
				if h.runningCtx.Err() != nil {
					return nil
				}
				return xerrors.Errorf("pubsub message subscription returned an error: %w", err)
			}
			gmsg, ok := msg.ValidatorData.(gpbft.ValidatedMessage)
			if !ok {
				h.log.Errorf("invalid msgValidatorData: %+v", msg.ValidatorData)
				continue
			}
			select {
			case messageQueue <- gmsg:
			case <-h.runningCtx.Done():
				return nil
			}
		}
		return nil
	})
	return messageQueue, nil
}

type gpbftTracer gpbftRunner

// Log fulfills the gpbft.Tracer interface
func (h *gpbftTracer) Log(fmt string, args ...any) {
	h.logWithSkip.Debugf(fmt, args...)
}

var _ gpbft.Tracer = (*gpbftTracer)(nil)

// gpbftHost is a newtype of gpbftRunner exposing APIs required by the gpbft.Participant
type gpbftHost gpbftRunner

func (h *gpbftHost) collectChain(base ec.TipSet, head ec.TipSet) ([]ec.TipSet, error) {
	// TODO: optimize when head is way beyond base
	res := make([]ec.TipSet, 0, 2*gpbft.CHAIN_MAX_LEN)
	res = append(res, head)

	for !bytes.Equal(head.Key(), base.Key()) {
		if head.Epoch() < base.Epoch() {
			// we reorged away from base
			// scream and panic??
			// TODO make sure this is correct, re-boostrap/manifest swap code has to be able to
			// catch it
			panic("reorg-ed away from base, dunno what to do, reboostrap is the answer")
		}
		var err error
		head, err = h.ec.GetParent(h.runningCtx, head)
		if err != nil {
			return nil, xerrors.Errorf("walking back the chain: %w", err)
		}
		res = append(res, head)
	}
	slices.Reverse(res)
	return res[1:], nil
}

func (h *gpbftRunner) Stop(_ctx context.Context) error {
	h.ctxCancel()
	return multierr.Combine(
		h.errgrp.Wait(),
		h.teardownPubsub(),
	)
}

// Returns inputs to the next GPBFT instance.
// These are:
// - the supplemental data.
// - the EC chain to propose.
// These will be used as input to a subsequent instance of the protocol.
// The chain should be a suffix of the last chain notified to the host via
// ReceiveDecision (or known to be final via some other channel).
func (h *gpbftHost) GetProposalForInstance(instance uint64) (*gpbft.SupplementalData, gpbft.ECChain, error) {
	var baseTsk gpbft.TipSetKey
	if instance == h.manifest.InitialInstance {
		ts, err := h.ec.GetTipsetByEpoch(h.runningCtx,
			h.manifest.BootstrapEpoch-h.manifest.ECFinality)
		if err != nil {
			return nil, nil, xerrors.Errorf("getting boostrap base: %w", err)
		}
		baseTsk = ts.Key()
	} else {
		cert, err := h.certStore.Get(h.runningCtx, instance-1)
		if err != nil {
			return nil, nil, xerrors.Errorf("getting cert for previous instance(%d): %w", instance-1, err)
		}
		baseTsk = cert.ECChain.Head().Key
	}

	baseTs, err := h.ec.GetTipset(h.runningCtx, baseTsk)
	if err != nil {
		return nil, nil, xerrors.Errorf("getting base TS: %w", err)
	}
	headTs, err := h.ec.GetHead(h.runningCtx)
	if err != nil {
		return nil, nil, xerrors.Errorf("getting head TS: %w", err)
	}
	if time.Since(headTs.Timestamp()) < h.manifest.ECPeriod {
		// less than ECPeriod since production of the head
		// agreement is unlikely
		headTs, err = h.ec.GetParent(h.runningCtx, headTs)
		if err != nil {
			return nil, nil, xerrors.Errorf("getting the parent of head TS: %w", err)
		}
	}

	collectedChain, err := h.collectChain(baseTs, headTs)
	if err != nil {
		return nil, nil, xerrors.Errorf("collecting chain: %w", err)
	}

	base := gpbft.TipSet{
		Epoch: baseTs.Epoch(),
		Key:   baseTs.Key(),
	}
	pte, err := h.ec.GetPowerTable(h.runningCtx, baseTs.Key())
	if err != nil {
		return nil, nil, xerrors.Errorf("getting power table for base: %w", err)
	}
	base.PowerTable, err = certs.MakePowerTableCID(pte)
	if err != nil {
		return nil, nil, xerrors.Errorf("computing powertable CID for base: %w", err)
	}

	suffix := make([]gpbft.TipSet, min(gpbft.CHAIN_MAX_LEN-1, len(collectedChain))) // -1 because of base
	for i := range suffix {
		suffix[i].Key = collectedChain[i].Key()
		suffix[i].Epoch = collectedChain[i].Epoch()

		pte, err = h.ec.GetPowerTable(h.runningCtx, suffix[i].Key)
		if err != nil {
			return nil, nil, xerrors.Errorf("getting power table for suffix %d: %w", i, err)
		}
		suffix[i].PowerTable, err = certs.MakePowerTableCID(pte)
		if err != nil {
			return nil, nil, xerrors.Errorf("computing powertable CID for base: %w", err)
		}
	}
	chain, err := gpbft.NewChain(base, suffix...)
	if err != nil {
		return nil, nil, xerrors.Errorf("making new chain: %w", err)
	}

	var supplData gpbft.SupplementalData
	pt, _, err := h.GetCommitteeForInstance(instance + 1)
	if err != nil {
		return nil, nil, xerrors.Errorf("getting commite for %d: %w", instance+1, err)
	}

	supplData.PowerTable, err = certs.MakePowerTableCID(pt.Entries)
	if err != nil {
		return nil, nil, xerrors.Errorf("making power table cid for supplemental data: %w", err)
	}

	return &supplData, chain, nil
}

func (h *gpbftHost) GetCommitteeForInstance(instance uint64) (*gpbft.PowerTable, []byte, error) {
	var powerTsk gpbft.TipSetKey
	var powerEntries gpbft.PowerEntries
	var err error

	if instance < h.manifest.InitialInstance+h.manifest.CommiteeLookback {
		//boostrap phase
		ts, err := h.ec.GetTipsetByEpoch(h.runningCtx, h.manifest.BootstrapEpoch-h.manifest.ECFinality)
		if err != nil {
			return nil, nil, xerrors.Errorf("getting tipset for boostrap epoch with lookback: %w", err)
		}
		powerTsk = ts.Key()
		powerEntries, err = h.ec.GetPowerTable(h.runningCtx, powerTsk)
		if err != nil {
			return nil, nil, xerrors.Errorf("getting power table: %w", err)
		}
	} else {
		cert, err := h.certStore.Get(h.runningCtx, instance-h.manifest.CommiteeLookback)
		if err != nil {
			return nil, nil, xerrors.Errorf("getting finality certificate: %w", err)
		}
		powerTsk = cert.ECChain.Head().Key

		powerEntries, err = h.certStore.GetPowerTable(h.runningCtx, instance)
		if err != nil {
			h.log.Debugf("failed getting power table from certstore: %v, falling back to EC", err)

			powerEntries, err = h.ec.GetPowerTable(h.runningCtx, powerTsk)
			if err != nil {
				return nil, nil, xerrors.Errorf("getting power table: %w", err)
			}
		}
	}

	ts, err := h.ec.GetTipset(h.runningCtx, powerTsk)
	if err != nil {
		return nil, nil, xerrors.Errorf("getting tipset: %w", err)
	}

	table := gpbft.NewPowerTable()
	err = table.Add(powerEntries...)
	if err != nil {
		return nil, nil, xerrors.Errorf("adding entries to power table: %w", err)
	}

	return table, ts.Beacon(), nil
}

// Returns the network's name (for signature separation)
func (h *gpbftHost) NetworkName() gpbft.NetworkName {
	return h.manifest.NetworkName
}

// Sends a message to all other participants.
// The message's sender must be one that the network interface can sign on behalf of.
func (h *gpbftHost) RequestBroadcast(mb *gpbft.MessageBuilder) error {
	mb.SetNetworkName(h.manifest.NetworkName)
	mb.SetSigningMarshaler(h.signingMarshaller)
	(h.broadcastCb)(mb)
	return nil
}

// Returns the current network time.
func (h *gpbftHost) Time() time.Time {
	return time.Now()
}

// Sets an alarm to fire after the given timestamp.
// At most one alarm can be set at a time.
// Setting an alarm replaces any previous alarm that has not yet fired.
// The timestamp may be in the past, in which case the alarm will fire as soon as possible
// (but not synchronously).
func (h *gpbftHost) SetAlarm(at time.Time) {
	h.log.Debugf("set alarm for %v", at)
	// we cannot reuse the timer because we don't know if it was read or not
	h.alertTimer.Stop()
	h.alertTimer = time.NewTimer(time.Until(at))
}

// Receives a finality decision from the instance, with signatures from a strong quorum
// of participants justifying it.
// The decision payload always has round = 0 and step = DECIDE.
// The notification must return the timestamp at which the next instance should begin,
// based on the decision received (which may be in the past).
// E.g. this might be: finalised tipset timestamp + epoch duration + stabilisation delay.
func (h *gpbftHost) ReceiveDecision(decision *gpbft.Justification) time.Time {
	h.log.Infof("got decision at instance %d, finalized head at epoch: %d",
		decision.Vote.Instance, decision.Vote.Value.Head().Epoch)
	cert, err := h.saveDecision(decision)
	if err != nil {
		h.log.Errorf("error while saving decision: %+v", err)
	}
	return (*gpbftRunner)(h).computeNextInstanceStart(cert)
}

func (h *gpbftHost) saveDecision(decision *gpbft.Justification) (*certs.FinalityCertificate, error) {
	instance := decision.Vote.Instance
	current, _, err := h.GetCommitteeForInstance(instance)
	if err != nil {
		return nil, xerrors.Errorf("getting commitee for current instance %d: %w", instance, err)
	}

	next, _, err := h.GetCommitteeForInstance(instance + 1)
	if err != nil {
		return nil, xerrors.Errorf("getting commitee for next instance %d: %w", instance+1, err)
	}
	powerDiff := certs.MakePowerTableDiff(current.Entries, next.Entries)

	cert, err := certs.NewFinalityCertificate(powerDiff, decision)
	if err != nil {
		return nil, xerrors.Errorf("forming certificate out of decision: %w", err)
	}
	_, _, _, err = certs.ValidateFinalityCertificates(h, h.NetworkName(), current.Entries, decision.Vote.Instance, nil, *cert)
	if err != nil {
		return nil, xerrors.Errorf("certificate is invalid: %w", err)
	}

	err = h.certStore.Put(h.runningCtx, cert)
	if err != nil {
		return nil, xerrors.Errorf("saving ceritifcate in a store: %w", err)
	}

	return cert, nil
}

// MarshalPayloadForSigning marshals the given payload into the bytes that should be signed.
// This should usually call `Payload.MarshalForSigning(NetworkName)` except when testing as
// that method is slow (computes a merkle tree that's necessary for testing).
func (h *gpbftHost) MarshalPayloadForSigning(nn gpbft.NetworkName, p *gpbft.Payload) []byte {
	return h.signingMarshaller.MarshalPayloadForSigning(nn, p)
}

// Verifies a signature for the given public key.
// Implementations must be safe for concurrent use.
func (h *gpbftHost) Verify(pubKey gpbft.PubKey, msg []byte, sig []byte) error {
	return h.verifier.Verify(pubKey, msg, sig)
}

// Aggregates signatures from a participants.
func (h *gpbftHost) Aggregate(pubKeys []gpbft.PubKey, sigs [][]byte) ([]byte, error) {
	return h.verifier.Aggregate(pubKeys, sigs)
}

// VerifyAggregate verifies an aggregate signature.
// Implementations must be safe for concurrent use.
func (h *gpbftHost) VerifyAggregate(payload []byte, aggSig []byte, signers []gpbft.PubKey) error {
	return h.verifier.VerifyAggregate(payload, aggSig, signers)
}
