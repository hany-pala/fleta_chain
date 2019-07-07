package vault

import (
	"bytes"
	"encoding/json"

	"github.com/fletaio/fleta/common"
	"github.com/fletaio/fleta/common/amount"
	"github.com/fletaio/fleta/core/types"
)

// Withdraw makes utxos from the account
type Withdraw struct {
	Timestamp_ uint64
	Seq_       uint64
	From       common.Address
	Vout       []*types.TxOut
}

// Timestamp returns the timestamp of the transaction
func (tx *Withdraw) Timestamp() uint64 {
	return tx.Timestamp_
}

// Seq returns the sequence of the transaction
func (tx *Withdraw) Seq() uint64 {
	return tx.Seq_
}

// Fee returns the fee of the transaction
func (tx *Withdraw) Fee(loader types.LoaderWrapper) *amount.Amount {
	return amount.COIN.DivC(10)
}

// Validate validates signatures of the transaction
func (tx *Withdraw) Validate(p types.Process, loader types.LoaderWrapper, signers []common.PublicHash) error {
	if tx.Seq() <= loader.Seq(tx.From) {
		return types.ErrInvalidSequence
	}

	fromAcc, err := loader.Account(tx.From)
	if err != nil {
		return err
	}
	if err := fromAcc.Validate(loader, signers); err != nil {
		return err
	}
	return nil
}

// Execute updates the context by the transaction
func (tx *Withdraw) Execute(p types.Process, ctw *types.ContextWrapper, index uint16) error {
	sp := p.(*Vault)

	sn := ctw.Snapshot()
	defer ctw.Revert(sn)

	if tx.Seq() != ctw.Seq(tx.From)+1 {
		return types.ErrInvalidSequence
	}
	ctw.AddSeq(tx.From)

	if has, err := ctw.HasAccount(tx.From); err != nil {
		return err
	} else if !has {
		return types.ErrNotExistAccount
	}
	outsum := tx.Fee(ctw)
	for n, vout := range tx.Vout {
		if vout.Amount.Less(amount.COIN.DivC(10)) {
			return types.ErrDustAmount
		}
		outsum = outsum.Add(vout.Amount)
		if err := ctw.CreateUTXO(types.MarshalID(ctw.TargetHeight(), index, uint16(n)), vout); err != nil {
			return err
		}
	}
	if err := sp.SubBalance(ctw, tx.From, outsum); err != nil {
		return err
	}

	ctw.Commit(sn)
	return nil
}

// MarshalJSON is a marshaler function
func (tx *Withdraw) MarshalJSON() ([]byte, error) {
	var buffer bytes.Buffer
	buffer.WriteString(`{`)
	buffer.WriteString(`"timestamp":`)
	if bs, err := json.Marshal(tx.Timestamp_); err != nil {
		return nil, err
	} else {
		buffer.Write(bs)
	}
	buffer.WriteString(`,`)
	buffer.WriteString(`"seq":`)
	if bs, err := json.Marshal(tx.Seq_); err != nil {
		return nil, err
	} else {
		buffer.Write(bs)
	}
	buffer.WriteString(`,`)
	buffer.WriteString(`"from":`)
	if bs, err := tx.From.MarshalJSON(); err != nil {
		return nil, err
	} else {
		buffer.Write(bs)
	}
	buffer.WriteString(`,`)
	buffer.WriteString(`"vout":`)
	buffer.WriteString(`[`)
	for i, vout := range tx.Vout {
		if i > 0 {
			buffer.WriteString(`,`)
		}
		if bs, err := vout.MarshalJSON(); err != nil {
			return nil, err
		} else {
			buffer.Write(bs)
		}
	}
	buffer.WriteString(`]`)
	buffer.WriteString(`}`)
	return buffer.Bytes(), nil
}