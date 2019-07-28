package formulator

import (
	"github.com/fletaio/fleta/common"
	"github.com/fletaio/fleta/common/amount"
	"github.com/fletaio/fleta/core/types"
	"github.com/fletaio/fleta/encoding"
	"github.com/fletaio/fleta/process/admin"
	"github.com/fletaio/fleta/process/vault"
)

// Formulator manages balance of accounts of the chain
type Formulator struct {
	*types.ProcessBase
	pid   uint8
	pm    types.ProcessManager
	cn    types.Provider
	vault *vault.Vault
	admin *admin.Admin
}

// NewFormulator returns a Formulator
func NewFormulator(pid uint8) *Formulator {
	p := &Formulator{
		pid: pid,
	}
	return p
}

// ID returns the id of the process
func (p *Formulator) ID() uint8 {
	return p.pid
}

// Name returns the name of the process
func (p *Formulator) Name() string {
	return "fleta.formulator"
}

// Version returns the version of the process
func (p *Formulator) Version() string {
	return "0.0.1"
}

// Init initializes the process
func (p *Formulator) Init(reg *types.Register, pm types.ProcessManager, cn types.Provider) error {
	p.pm = pm
	p.cn = cn

	if vp, err := pm.ProcessByName("fleta.vault"); err != nil {
		return err
	} else if v, is := vp.(*vault.Vault); !is {
		return types.ErrInvalidProcess
	} else {
		p.vault = v
	}
	if vp, err := pm.ProcessByName("fleta.admin"); err != nil {
		return err
	} else if v, is := vp.(*admin.Admin); !is {
		return types.ErrInvalidProcess
	} else {
		p.admin = v
	}

	reg.RegisterAccount(1, &FormulatorAccount{})
	reg.RegisterTransaction(1, &CreateAlpha{})
	reg.RegisterTransaction(2, &CreateSigma{})
	reg.RegisterTransaction(3, &CreateOmega{})
	reg.RegisterTransaction(4, &CreateHyper{})
	reg.RegisterTransaction(5, &Revoke{})
	reg.RegisterTransaction(6, &Staking{})
	reg.RegisterTransaction(7, &Unstaking{})
	reg.RegisterTransaction(8, &UpdateValidatorPolicy{})
	reg.RegisterTransaction(9, &UpdateUserAutoStaking{})
	reg.RegisterTransaction(10, &ChangeOwner{})
	return nil
}

// InitPolicy called at OnInitGenesis of an application
func (p *Formulator) InitPolicy(ctw *types.ContextWrapper, rp *RewardPolicy, ap *AlphaPolicy, sp *SigmaPolicy, op *OmegaPolicy, hp *HyperPolicy) error {
	ctw = types.SwitchContextWrapper(p.pid, ctw)

	if bs, err := encoding.Marshal(rp); err != nil {
		return err
	} else {
		ctw.SetProcessData(tagRewardPolicy, bs)
	}
	if bs, err := encoding.Marshal(ap); err != nil {
		return err
	} else {
		ctw.SetProcessData(tagAlphaPolicy, bs)
	}
	if bs, err := encoding.Marshal(sp); err != nil {
		return err
	} else {
		ctw.SetProcessData(tagSigmaPolicy, bs)
	}
	if bs, err := encoding.Marshal(op); err != nil {
		return err
	} else {
		ctw.SetProcessData(tagOmegaPolicy, bs)
	}
	if bs, err := encoding.Marshal(hp); err != nil {
		return err
	} else {
		ctw.SetProcessData(tagHyperPolicy, bs)
	}
	return nil
}

// OnLoadChain called when the chain loaded
func (p *Formulator) OnLoadChain(loader types.LoaderWrapper) error {
	p.admin.AdminAddress(loader, p.Name())
	if bs := loader.ProcessData(tagRewardPolicy); len(bs) == 0 {
		return ErrRewardPolicyShouldBeSetupInApplication
	}
	if bs := loader.ProcessData(tagAlphaPolicy); len(bs) == 0 {
		return ErrAlphaPolicyShouldBeSetupInApplication
	}
	if bs := loader.ProcessData(tagSigmaPolicy); len(bs) == 0 {
		return ErrSigmaPolicyShouldBeSetupInApplication
	}
	if bs := loader.ProcessData(tagOmegaPolicy); len(bs) == 0 {
		return ErrOmegaPolicyShouldBeSetupInApplication
	}
	if bs := loader.ProcessData(tagHyperPolicy); len(bs) == 0 {
		return ErrHyperPolicyShouldBeSetupInApplication
	}
	return nil
}

// BeforeExecuteTransactions called before processes transactions of the block
func (p *Formulator) BeforeExecuteTransactions(ctw *types.ContextWrapper) error {
	return nil
}

// AfterExecuteTransactions called after processes transactions of the block
func (p *Formulator) AfterExecuteTransactions(b *types.Block, ctw *types.ContextWrapper) error {
	p.addGenCount(ctw, b.Header.Generator)

	policy := &RewardPolicy{}
	if err := encoding.Unmarshal(ctw.ProcessData(tagRewardPolicy), &policy); err != nil {
		return err
	}

	lastPaidHeight := p.getLastPaidHeight(ctw)
	if ctw.TargetHeight() >= lastPaidHeight+policy.PayRewardEveryBlocks {
		CountMap, err := p.flushGenCountMap(ctw)
		if err != nil {
			return err
		}

		StackRewardMap := types.NewAddressAmountMap()
		if bs := ctw.ProcessData(tagStackRewardMap); len(bs) > 0 {
			if err := encoding.Unmarshal(bs, &StackRewardMap); err != nil {
				return err
			}
		}

		RewardPowerSum := amount.NewCoinAmount(0, 0)
		RewardPowerMap := map[common.Address]*amount.Amount{}
		StakingRewardPowerMap := map[common.Address]*amount.Amount{}
		Hypers := []*FormulatorAccount{}
		for GenAddress, GenCount := range CountMap {
			acc, err := ctw.Account(GenAddress)
			if err != nil {
				return err
			}
			frAcc, is := acc.(*FormulatorAccount)
			if !is {
				return types.ErrInvalidAccountType
			}
			switch frAcc.FormulatorType {
			case AlphaFormulatorType:
				am := frAcc.Amount.MulC(int64(GenCount)).MulC(int64(policy.AlphaEfficiency1000)).DivC(1000)
				RewardPowerSum = RewardPowerSum.Add(am)
				RewardPowerMap[GenAddress] = am
			case SigmaFormulatorType:
				am := frAcc.Amount.MulC(int64(GenCount)).MulC(int64(policy.SigmaEfficiency1000)).DivC(1000)
				RewardPowerSum = RewardPowerSum.Add(am)
				RewardPowerMap[GenAddress] = am
			case OmegaFormulatorType:
				am := frAcc.Amount.MulC(int64(GenCount)).MulC(int64(policy.OmegaEfficiency1000)).DivC(1000)
				RewardPowerSum = RewardPowerSum.Add(am)
				RewardPowerMap[GenAddress] = am
			case HyperFormulatorType:
				Hypers = append(Hypers, frAcc)

				am := frAcc.Amount.MulC(int64(GenCount)).MulC(int64(policy.HyperEfficiency1000)).DivC(1000)
				RewardPowerSum = RewardPowerSum.Add(am)
				RewardPowerMap[GenAddress] = am

				PrevAmountMap := types.NewAddressAmountMap()
				if bs := ctw.AccountData(frAcc.Address(), tagStakingAmountMap); len(bs) > 0 {
					if err := encoding.Unmarshal(bs, &PrevAmountMap); err != nil {
						return err
					}
				}
				AmountMap, err := p.GetStakingAmountMap(ctw, frAcc.Address())
				if err != nil {
					return err
				}
				CurrentAmountMap := types.NewAddressAmountMap()
				CrossAmountMap := map[common.Address]*amount.Amount{}
				for StakingAddress, StakingAmount := range AmountMap {
					CurrentAmountMap.Put(StakingAddress, StakingAmount)
					if PrevStakingAmount, has := PrevAmountMap.Get(StakingAddress); has {
						if !PrevStakingAmount.IsZero() && !StakingAmount.IsZero() {
							if StakingAmount.Less(PrevStakingAmount) {
								CrossAmountMap[StakingAddress] = StakingAmount
							} else {
								CrossAmountMap[StakingAddress] = PrevStakingAmount
							}
						}
					}
				}
				if bs, err := encoding.Marshal(CurrentAmountMap); err != nil {
					return err
				} else {
					ctw.SetAccountData(frAcc.Address(), tagStakingAmountMap, bs)
				}

				StakingRewardPower := amount.NewCoinAmount(0, 0)
				StakingPowerMap := types.NewAddressAmountMap()
				if bs := ctw.AccountData(frAcc.Address(), tagStakingPowerMap); len(bs) > 0 {
					if err := encoding.Unmarshal(bs, &StakingPowerMap); err != nil {
						return err
					}
				}
				for StakingAddress, StakingAmount := range CrossAmountMap {
					if sm, has := StakingPowerMap.Get(StakingAddress); has {
						StakingPowerMap.Put(StakingAddress, sm.Add(StakingAmount))
					} else {
						StakingPowerMap.Put(StakingAddress, StakingAmount)
					}
					StakingRewardPower = StakingRewardPower.Add(StakingAmount.MulC(int64(GenCount)).MulC(int64(policy.StakingEfficiency1000)).DivC(1000))
				}

				StackReward, has := StackRewardMap.Get(frAcc.Address())
				if has {
					StakingPowerSum := amount.NewCoinAmount(0, 0)
					StakingPowerMap.EachAll(func(StakingAddress common.Address, StakingPower *amount.Amount) bool {
						StakingPowerSum = StakingPowerSum.Add(StakingPower)
						return true
					})
					if !StakingPowerSum.IsZero() {
						var inErr error
						Ratio := StackReward.Mul(amount.COIN).Div(StakingPowerSum)
						StakingPowerMap.EachAll(func(StakingAddress common.Address, StakingPower *amount.Amount) bool {
							StackStakingAmount := StakingPower.Mul(Ratio).Div(amount.COIN)
							StakingPowerMap.Put(StakingAddress, StakingPower.Add(StackStakingAmount))
							StakingRewardPower = StakingRewardPower.Add(StackStakingAmount.MulC(int64(GenCount)).MulC(int64(policy.StakingEfficiency1000)).DivC(1000))
							return true
						})
						if inErr != nil {
							return inErr
						}
					}
				}

				if bs, err := encoding.Marshal(StakingPowerMap); err != nil {
					return err
				} else {
					ctw.SetAccountData(frAcc.Address(), tagStakingPowerMap, bs)
				}
				StakingRewardPowerMap[GenAddress] = StakingRewardPower
				RewardPowerSum = RewardPowerSum.Add(StakingRewardPower)
			default:
				return types.ErrInvalidAccountType
			}
		}

		if !RewardPowerSum.IsZero() {
			TotalReward := policy.RewardPerBlock.MulC(int64(ctw.TargetHeight() - lastPaidHeight))
			TotalFee := p.vault.CollectedFee(ctw)
			if err := p.vault.SubCollectedFee(ctw, TotalFee); err != nil {
				return err
			}
			TotalReward = TotalReward.Add(TotalFee)

			Ratio := TotalReward.Mul(amount.COIN).Div(RewardPowerSum)
			for RewardAddress, RewardPower := range RewardPowerMap {
				RewardAmount := RewardPower.Mul(Ratio).Div(amount.COIN)
				if !RewardAmount.IsZero() {
					if err := p.vault.AddBalance(ctw, RewardAddress, RewardAmount); err != nil {
						return err
					}
				}
			}
			for GenAddress, StakingRewardPower := range StakingRewardPowerMap {
				if has, err := ctw.HasAccount(GenAddress); err != nil {
					return err
				} else if has {
					RewardAmount := StakingRewardPower.Mul(Ratio).Div(amount.COIN)
					if sm, has := StackRewardMap.Get(GenAddress); has {
						StackRewardMap.Put(GenAddress, sm.Add(RewardAmount))
					} else {
						StackRewardMap.Put(GenAddress, RewardAmount)
					}
				}
			}
		}
		for _, frAcc := range Hypers {
			if StackReward, has := StackRewardMap.Get(frAcc.Address()); has {
				lastStakingPaidHeight := p.getLastStakingPaidHeight(ctw, frAcc.Address())
				if ctw.TargetHeight() >= lastStakingPaidHeight+policy.PayRewardEveryBlocks*frAcc.Policy.PayOutInterval {
					StakingPowerMap := types.NewAddressAmountMap()
					if bs := ctw.AccountData(frAcc.Address(), tagStakingPowerMap); len(bs) > 0 {
						if err := encoding.Unmarshal(bs, &StakingPowerMap); err != nil {
							return err
						}
					}

					StakingPowerSum := amount.NewCoinAmount(0, 0)
					StakingPowerMap.EachAll(func(StakingAddress common.Address, StakingPower *amount.Amount) bool {
						StakingPowerSum = StakingPowerSum.Add(StakingPower)
						return true
					})
					if !StakingPowerSum.IsZero() {
						CommissionSum := amount.NewCoinAmount(0, 0)
						var inErr error
						Ratio := StackReward.Mul(amount.COIN).Div(StakingPowerSum)
						StakingPowerMap.EachAll(func(StakingAddress common.Address, StakingPower *amount.Amount) bool {
							RewardAmount := StakingPower.Mul(Ratio).Div(amount.COIN)
							if frAcc.Policy.CommissionRatio1000 > 0 {
								Commission := RewardAmount.MulC(int64(frAcc.Policy.CommissionRatio1000)).DivC(1000)
								CommissionSum = CommissionSum.Add(Commission)
								RewardAmount = RewardAmount.Sub(Commission)
							}
							if !RewardAmount.IsZero() {
								if p.getUserAutoStaking(ctw, frAcc.Address(), StakingAddress) {
									p.AddStakingAmount(ctw, frAcc.Address(), StakingAddress, RewardAmount)
								} else {
									if err := p.vault.AddBalance(ctw, StakingAddress, RewardAmount); err != nil {
										inErr = err
										return false
									}
								}
							}
							return true
						})
						if inErr != nil {
							return inErr
						}

						if !CommissionSum.IsZero() {
							if err := p.vault.AddBalance(ctw, frAcc.Address(), CommissionSum); err != nil {
								return err
							}
						}
					}
					ctw.SetAccountData(frAcc.Address(), tagStakingPowerMap, nil)

					StackRewardMap.Delete(frAcc.Address())
					p.setLastStakingPaidHeight(ctw, frAcc.Address(), ctw.TargetHeight())
				}
			}
		}
		if bs, err := encoding.Marshal(StackRewardMap); err != nil {
			return err
		} else {
			ctw.SetProcessData(tagStackRewardMap, bs)
		}

		//ctw.EmitEvent()
		//Addr, Earn, Commision, Staked, Adds

		//log.Println("Paid at", ctw.TargetHeight())
		p.setLastPaidHeight(ctw, ctw.TargetHeight())
	}
	return nil
}

// OnSaveData called when the context of the block saved
func (p *Formulator) OnSaveData(b *types.Block, ctw *types.ContextWrapper) error {
	return nil
}
