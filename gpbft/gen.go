// Code generated by github.com/whyrusleeping/cbor-gen. DO NOT EDIT.

package gpbft

import (
	"fmt"
	"io"
	"math"
	"sort"

	cid "github.com/ipfs/go-cid"
	cbg "github.com/whyrusleeping/cbor-gen"
	xerrors "golang.org/x/xerrors"
)

var _ = xerrors.Errorf
var _ = cid.Undef
var _ = math.E
var _ = sort.Sort

var lengthBufTipSet = []byte{132}

func (t *TipSet) MarshalCBOR(w io.Writer) error {
	if t == nil {
		_, err := w.Write(cbg.CborNull)
		return err
	}

	cw := cbg.NewCborWriter(w)

	if _, err := cw.Write(lengthBufTipSet); err != nil {
		return err
	}

	// t.Epoch (int64) (int64)
	if t.Epoch >= 0 {
		if err := cw.WriteMajorTypeHeader(cbg.MajUnsignedInt, uint64(t.Epoch)); err != nil {
			return err
		}
	} else {
		if err := cw.WriteMajorTypeHeader(cbg.MajNegativeInt, uint64(-t.Epoch-1)); err != nil {
			return err
		}
	}

	// t.TipSet ([]uint8) (slice)
	if len(t.TipSet) > 2097152 {
		return xerrors.Errorf("Byte array in field t.TipSet was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajByteString, uint64(len(t.TipSet))); err != nil {
		return err
	}

	if _, err := cw.Write(t.TipSet); err != nil {
		return err
	}

	// t.PowerTable ([]uint8) (slice)
	if len(t.PowerTable) > 2097152 {
		return xerrors.Errorf("Byte array in field t.PowerTable was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajByteString, uint64(len(t.PowerTable))); err != nil {
		return err
	}

	if _, err := cw.Write(t.PowerTable); err != nil {
		return err
	}

	// t.Commitments ([32]uint8) (array)
	if len(t.Commitments) > 2097152 {
		return xerrors.Errorf("Byte array in field t.Commitments was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajByteString, uint64(len(t.Commitments))); err != nil {
		return err
	}

	if _, err := cw.Write(t.Commitments[:]); err != nil {
		return err
	}
	return nil
}

func (t *TipSet) UnmarshalCBOR(r io.Reader) (err error) {
	*t = TipSet{}

	cr := cbg.NewCborReader(r)

	maj, extra, err := cr.ReadHeader()
	if err != nil {
		return err
	}
	defer func() {
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}
	}()

	if maj != cbg.MajArray {
		return fmt.Errorf("cbor input should be of type array")
	}

	if extra != 4 {
		return fmt.Errorf("cbor input had wrong number of fields")
	}

	// t.Epoch (int64) (int64)
	{
		maj, extra, err := cr.ReadHeader()
		if err != nil {
			return err
		}
		var extraI int64
		switch maj {
		case cbg.MajUnsignedInt:
			extraI = int64(extra)
			if extraI < 0 {
				return fmt.Errorf("int64 positive overflow")
			}
		case cbg.MajNegativeInt:
			extraI = int64(extra)
			if extraI < 0 {
				return fmt.Errorf("int64 negative overflow")
			}
			extraI = -1 - extraI
		default:
			return fmt.Errorf("wrong type for int64 field: %d", maj)
		}

		t.Epoch = int64(extraI)
	}
	// t.TipSet ([]uint8) (slice)

	maj, extra, err = cr.ReadHeader()
	if err != nil {
		return err
	}

	if extra > 2097152 {
		return fmt.Errorf("t.TipSet: byte array too large (%d)", extra)
	}
	if maj != cbg.MajByteString {
		return fmt.Errorf("expected byte array")
	}

	if extra > 0 {
		t.TipSet = make([]uint8, extra)
	}

	if _, err := io.ReadFull(cr, t.TipSet); err != nil {
		return err
	}

	// t.PowerTable ([]uint8) (slice)

	maj, extra, err = cr.ReadHeader()
	if err != nil {
		return err
	}

	if extra > 2097152 {
		return fmt.Errorf("t.PowerTable: byte array too large (%d)", extra)
	}
	if maj != cbg.MajByteString {
		return fmt.Errorf("expected byte array")
	}

	if extra > 0 {
		t.PowerTable = make([]uint8, extra)
	}

	if _, err := io.ReadFull(cr, t.PowerTable); err != nil {
		return err
	}

	// t.Commitments ([32]uint8) (array)

	maj, extra, err = cr.ReadHeader()
	if err != nil {
		return err
	}

	if extra > 2097152 {
		return fmt.Errorf("t.Commitments: byte array too large (%d)", extra)
	}
	if maj != cbg.MajByteString {
		return fmt.Errorf("expected byte array")
	}
	if extra != 32 {
		return fmt.Errorf("expected array to have 32 elements")
	}

	t.Commitments = [32]uint8{}
	if _, err := io.ReadFull(cr, t.Commitments[:]); err != nil {
		return err
	}
	return nil
}

var lengthBufGMessage = []byte{133}

func (t *GMessage) MarshalCBOR(w io.Writer) error {
	if t == nil {
		_, err := w.Write(cbg.CborNull)
		return err
	}

	cw := cbg.NewCborWriter(w)

	if _, err := cw.Write(lengthBufGMessage); err != nil {
		return err
	}

	// t.Sender (gpbft.ActorID) (uint64)

	if err := cw.WriteMajorTypeHeader(cbg.MajUnsignedInt, uint64(t.Sender)); err != nil {
		return err
	}

	// t.Vote (gpbft.Payload) (struct)
	if err := t.Vote.MarshalCBOR(cw); err != nil {
		return err
	}

	// t.Signature ([]uint8) (slice)
	if len(t.Signature) > 2097152 {
		return xerrors.Errorf("Byte array in field t.Signature was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajByteString, uint64(len(t.Signature))); err != nil {
		return err
	}

	if _, err := cw.Write(t.Signature); err != nil {
		return err
	}

	// t.Ticket (gpbft.Ticket) (slice)
	if len(t.Ticket) > 2097152 {
		return xerrors.Errorf("Byte array in field t.Ticket was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajByteString, uint64(len(t.Ticket))); err != nil {
		return err
	}

	if _, err := cw.Write(t.Ticket); err != nil {
		return err
	}

	// t.Justification (gpbft.Justification) (struct)
	if err := t.Justification.MarshalCBOR(cw); err != nil {
		return err
	}
	return nil
}

func (t *GMessage) UnmarshalCBOR(r io.Reader) (err error) {
	*t = GMessage{}

	cr := cbg.NewCborReader(r)

	maj, extra, err := cr.ReadHeader()
	if err != nil {
		return err
	}
	defer func() {
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}
	}()

	if maj != cbg.MajArray {
		return fmt.Errorf("cbor input should be of type array")
	}

	if extra != 5 {
		return fmt.Errorf("cbor input had wrong number of fields")
	}

	// t.Sender (gpbft.ActorID) (uint64)

	{

		maj, extra, err = cr.ReadHeader()
		if err != nil {
			return err
		}
		if maj != cbg.MajUnsignedInt {
			return fmt.Errorf("wrong type for uint64 field")
		}
		t.Sender = ActorID(extra)

	}
	// t.Vote (gpbft.Payload) (struct)

	{

		if err := t.Vote.UnmarshalCBOR(cr); err != nil {
			return xerrors.Errorf("unmarshaling t.Vote: %w", err)
		}

	}
	// t.Signature ([]uint8) (slice)

	maj, extra, err = cr.ReadHeader()
	if err != nil {
		return err
	}

	if extra > 2097152 {
		return fmt.Errorf("t.Signature: byte array too large (%d)", extra)
	}
	if maj != cbg.MajByteString {
		return fmt.Errorf("expected byte array")
	}

	if extra > 0 {
		t.Signature = make([]uint8, extra)
	}

	if _, err := io.ReadFull(cr, t.Signature); err != nil {
		return err
	}

	// t.Ticket (gpbft.Ticket) (slice)

	maj, extra, err = cr.ReadHeader()
	if err != nil {
		return err
	}

	if extra > 2097152 {
		return fmt.Errorf("t.Ticket: byte array too large (%d)", extra)
	}
	if maj != cbg.MajByteString {
		return fmt.Errorf("expected byte array")
	}

	if extra > 0 {
		t.Ticket = make([]uint8, extra)
	}

	if _, err := io.ReadFull(cr, t.Ticket); err != nil {
		return err
	}

	// t.Justification (gpbft.Justification) (struct)

	{

		b, err := cr.ReadByte()
		if err != nil {
			return err
		}
		if b != cbg.CborNull[0] {
			if err := cr.UnreadByte(); err != nil {
				return err
			}
			t.Justification = new(Justification)
			if err := t.Justification.UnmarshalCBOR(cr); err != nil {
				return xerrors.Errorf("unmarshaling t.Justification pointer: %w", err)
			}
		}

	}
	return nil
}

var lengthBufPayload = []byte{132}

func (t *Payload) MarshalCBOR(w io.Writer) error {
	if t == nil {
		_, err := w.Write(cbg.CborNull)
		return err
	}

	cw := cbg.NewCborWriter(w)

	if _, err := cw.Write(lengthBufPayload); err != nil {
		return err
	}

	// t.Instance (uint64) (uint64)

	if err := cw.WriteMajorTypeHeader(cbg.MajUnsignedInt, uint64(t.Instance)); err != nil {
		return err
	}

	// t.Round (uint64) (uint64)

	if err := cw.WriteMajorTypeHeader(cbg.MajUnsignedInt, uint64(t.Round)); err != nil {
		return err
	}

	// t.Step (gpbft.Phase) (uint8)
	if err := cw.WriteMajorTypeHeader(cbg.MajUnsignedInt, uint64(t.Step)); err != nil {
		return err
	}

	// t.Value (gpbft.ECChain) (slice)
	if len(t.Value) > 8192 {
		return xerrors.Errorf("Slice value in field t.Value was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajArray, uint64(len(t.Value))); err != nil {
		return err
	}
	for _, v := range t.Value {
		if err := v.MarshalCBOR(cw); err != nil {
			return err
		}

	}
	return nil
}

func (t *Payload) UnmarshalCBOR(r io.Reader) (err error) {
	*t = Payload{}

	cr := cbg.NewCborReader(r)

	maj, extra, err := cr.ReadHeader()
	if err != nil {
		return err
	}
	defer func() {
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}
	}()

	if maj != cbg.MajArray {
		return fmt.Errorf("cbor input should be of type array")
	}

	if extra != 4 {
		return fmt.Errorf("cbor input had wrong number of fields")
	}

	// t.Instance (uint64) (uint64)

	{

		maj, extra, err = cr.ReadHeader()
		if err != nil {
			return err
		}
		if maj != cbg.MajUnsignedInt {
			return fmt.Errorf("wrong type for uint64 field")
		}
		t.Instance = uint64(extra)

	}
	// t.Round (uint64) (uint64)

	{

		maj, extra, err = cr.ReadHeader()
		if err != nil {
			return err
		}
		if maj != cbg.MajUnsignedInt {
			return fmt.Errorf("wrong type for uint64 field")
		}
		t.Round = uint64(extra)

	}
	// t.Step (gpbft.Phase) (uint8)

	maj, extra, err = cr.ReadHeader()
	if err != nil {
		return err
	}
	if maj != cbg.MajUnsignedInt {
		return fmt.Errorf("wrong type for uint8 field")
	}
	if extra > math.MaxUint8 {
		return fmt.Errorf("integer in input was too large for uint8 field")
	}
	t.Step = Phase(extra)
	// t.Value (gpbft.ECChain) (slice)

	maj, extra, err = cr.ReadHeader()
	if err != nil {
		return err
	}

	if extra > 8192 {
		return fmt.Errorf("t.Value: array too large (%d)", extra)
	}

	if maj != cbg.MajArray {
		return fmt.Errorf("expected cbor array")
	}

	if extra > 0 {
		t.Value = make([]TipSet, extra)
	}

	for i := 0; i < int(extra); i++ {
		{
			var maj byte
			var extra uint64
			var err error
			_ = maj
			_ = extra
			_ = err

			{

				if err := t.Value[i].UnmarshalCBOR(cr); err != nil {
					return xerrors.Errorf("unmarshaling t.Value[i]: %w", err)
				}

			}

		}
	}
	return nil
}

var lengthBufJustification = []byte{131}

func (t *Justification) MarshalCBOR(w io.Writer) error {
	if t == nil {
		_, err := w.Write(cbg.CborNull)
		return err
	}

	cw := cbg.NewCborWriter(w)

	if _, err := cw.Write(lengthBufJustification); err != nil {
		return err
	}

	// t.Vote (gpbft.Payload) (struct)
	if err := t.Vote.MarshalCBOR(cw); err != nil {
		return err
	}

	// t.Signers (bitfield.BitField) (struct)
	if err := t.Signers.MarshalCBOR(cw); err != nil {
		return err
	}

	// t.Signature ([]uint8) (slice)
	if len(t.Signature) > 2097152 {
		return xerrors.Errorf("Byte array in field t.Signature was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajByteString, uint64(len(t.Signature))); err != nil {
		return err
	}

	if _, err := cw.Write(t.Signature); err != nil {
		return err
	}

	return nil
}

func (t *Justification) UnmarshalCBOR(r io.Reader) (err error) {
	*t = Justification{}

	cr := cbg.NewCborReader(r)

	maj, extra, err := cr.ReadHeader()
	if err != nil {
		return err
	}
	defer func() {
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}
	}()

	if maj != cbg.MajArray {
		return fmt.Errorf("cbor input should be of type array")
	}

	if extra != 3 {
		return fmt.Errorf("cbor input had wrong number of fields")
	}

	// t.Vote (gpbft.Payload) (struct)

	{

		if err := t.Vote.UnmarshalCBOR(cr); err != nil {
			return xerrors.Errorf("unmarshaling t.Vote: %w", err)
		}

	}
	// t.Signers (bitfield.BitField) (struct)

	{

		if err := t.Signers.UnmarshalCBOR(cr); err != nil {
			return xerrors.Errorf("unmarshaling t.Signers: %w", err)
		}

	}
	// t.Signature ([]uint8) (slice)

	maj, extra, err = cr.ReadHeader()
	if err != nil {
		return err
	}

	if extra > 2097152 {
		return fmt.Errorf("t.Signature: byte array too large (%d)", extra)
	}
	if maj != cbg.MajByteString {
		return fmt.Errorf("expected byte array")
	}

	if extra > 0 {
		t.Signature = make([]uint8, extra)
	}

	if _, err := io.ReadFull(cr, t.Signature); err != nil {
		return err
	}

	return nil
}
