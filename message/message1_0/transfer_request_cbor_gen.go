// Code generated by github.com/whyrusleeping/cbor-gen. DO NOT EDIT.

package message1_0

import (
	"fmt"
	"io"

	datatransfer "github.com/chenjianmei111/go-data-transfer"
	cbg "github.com/whyrusleeping/cbor-gen"
	xerrors "golang.org/x/xerrors"
)

var _ = xerrors.Errorf

var lengthBuftransferRequest = []byte{137}

func (t *transferRequest) MarshalCBOR(w io.Writer) error {
	if t == nil {
		_, err := w.Write(cbg.CborNull)
		return err
	}
	if _, err := w.Write(lengthBuftransferRequest); err != nil {
		return err
	}

	scratch := make([]byte, 9)

	// t.BCid (cid.Cid) (struct)

	if t.BCid == nil {
		if _, err := w.Write(cbg.CborNull); err != nil {
			return err
		}
	} else {
		if err := cbg.WriteCidBuf(scratch, w, *t.BCid); err != nil {
			return xerrors.Errorf("failed to write cid field t.BCid: %w", err)
		}
	}

	// t.Type (uint64) (uint64)

	if err := cbg.WriteMajorTypeHeaderBuf(scratch, w, cbg.MajUnsignedInt, uint64(t.Type)); err != nil {
		return err
	}

	// t.Paus (bool) (bool)
	if err := cbg.WriteBool(w, t.Paus); err != nil {
		return err
	}

	// t.Part (bool) (bool)
	if err := cbg.WriteBool(w, t.Part); err != nil {
		return err
	}

	// t.Pull (bool) (bool)
	if err := cbg.WriteBool(w, t.Pull); err != nil {
		return err
	}

	// t.Stor (typegen.Deferred) (struct)
	if err := t.Stor.MarshalCBOR(w); err != nil {
		return err
	}

	// t.Vouch (typegen.Deferred) (struct)
	if err := t.Vouch.MarshalCBOR(w); err != nil {
		return err
	}

	// t.VTyp (datatransfer.TypeIdentifier) (string)
	if len(t.VTyp) > cbg.MaxLength {
		return xerrors.Errorf("Value in field t.VTyp was too long")
	}

	if err := cbg.WriteMajorTypeHeaderBuf(scratch, w, cbg.MajTextString, uint64(len(t.VTyp))); err != nil {
		return err
	}
	if _, err := io.WriteString(w, string(t.VTyp)); err != nil {
		return err
	}

	// t.XferID (uint64) (uint64)

	if err := cbg.WriteMajorTypeHeaderBuf(scratch, w, cbg.MajUnsignedInt, uint64(t.XferID)); err != nil {
		return err
	}

	return nil
}

func (t *transferRequest) UnmarshalCBOR(r io.Reader) error {
	*t = transferRequest{}

	br := cbg.GetPeeker(r)
	scratch := make([]byte, 8)

	maj, extra, err := cbg.CborReadHeaderBuf(br, scratch)
	if err != nil {
		return err
	}
	if maj != cbg.MajArray {
		return fmt.Errorf("cbor input should be of type array")
	}

	if extra != 9 {
		return fmt.Errorf("cbor input had wrong number of fields")
	}

	// t.BCid (cid.Cid) (struct)

	{

		b, err := br.ReadByte()
		if err != nil {
			return err
		}
		if b != cbg.CborNull[0] {
			if err := br.UnreadByte(); err != nil {
				return err
			}

			c, err := cbg.ReadCid(br)
			if err != nil {
				return xerrors.Errorf("failed to read cid field t.BCid: %w", err)
			}

			t.BCid = &c
		}

	}
	// t.Type (uint64) (uint64)

	{

		maj, extra, err = cbg.CborReadHeaderBuf(br, scratch)
		if err != nil {
			return err
		}
		if maj != cbg.MajUnsignedInt {
			return fmt.Errorf("wrong type for uint64 field")
		}
		t.Type = uint64(extra)

	}
	// t.Paus (bool) (bool)

	maj, extra, err = cbg.CborReadHeaderBuf(br, scratch)
	if err != nil {
		return err
	}
	if maj != cbg.MajOther {
		return fmt.Errorf("booleans must be major type 7")
	}
	switch extra {
	case 20:
		t.Paus = false
	case 21:
		t.Paus = true
	default:
		return fmt.Errorf("booleans are either major type 7, value 20 or 21 (got %d)", extra)
	}
	// t.Part (bool) (bool)

	maj, extra, err = cbg.CborReadHeaderBuf(br, scratch)
	if err != nil {
		return err
	}
	if maj != cbg.MajOther {
		return fmt.Errorf("booleans must be major type 7")
	}
	switch extra {
	case 20:
		t.Part = false
	case 21:
		t.Part = true
	default:
		return fmt.Errorf("booleans are either major type 7, value 20 or 21 (got %d)", extra)
	}
	// t.Pull (bool) (bool)

	maj, extra, err = cbg.CborReadHeaderBuf(br, scratch)
	if err != nil {
		return err
	}
	if maj != cbg.MajOther {
		return fmt.Errorf("booleans must be major type 7")
	}
	switch extra {
	case 20:
		t.Pull = false
	case 21:
		t.Pull = true
	default:
		return fmt.Errorf("booleans are either major type 7, value 20 or 21 (got %d)", extra)
	}
	// t.Stor (typegen.Deferred) (struct)

	{

		t.Stor = new(cbg.Deferred)

		if err := t.Stor.UnmarshalCBOR(br); err != nil {
			return xerrors.Errorf("failed to read deferred field: %w", err)
		}
	}
	// t.Vouch (typegen.Deferred) (struct)

	{

		t.Vouch = new(cbg.Deferred)

		if err := t.Vouch.UnmarshalCBOR(br); err != nil {
			return xerrors.Errorf("failed to read deferred field: %w", err)
		}
	}
	// t.VTyp (datatransfer.TypeIdentifier) (string)

	{
		sval, err := cbg.ReadStringBuf(br, scratch)
		if err != nil {
			return err
		}

		t.VTyp = datatransfer.TypeIdentifier(sval)
	}
	// t.XferID (uint64) (uint64)

	{

		maj, extra, err = cbg.CborReadHeaderBuf(br, scratch)
		if err != nil {
			return err
		}
		if maj != cbg.MajUnsignedInt {
			return fmt.Errorf("wrong type for uint64 field")
		}
		t.XferID = uint64(extra)

	}
	return nil
}
