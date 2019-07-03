package types

// Process is a interface of the chain Process
type Process interface {
	Version() string
	Name() string
	Init(reg *Register, pm ProcessManager, cn Provider) error
	OnLoadChain(loader LoaderWrapper) error
	BeforeExecuteTransactions(ctw *ContextWrapper) error
	AfterExecuteTransactions(b *Block, ctw *ContextWrapper) error
	OnSaveData(b *Block, ctw *ContextWrapper) error
}

// ProcessBase is a base handler of the chain Process
type ProcessBase struct{}

// InitGenesis initializes genesis data
func (p *ProcessBase) InitGenesis(ctw *ContextWrapper) error {
	return nil
}

// OnLoadChain called when the chain loaded
func (p *ProcessBase) OnLoadChain(loader LoaderWrapper) error {
	return nil
}

// BeforeExecuteTransactions called before processes transactions of the block
func (p *ProcessBase) BeforeExecuteTransactions(ctw *ContextWrapper) error {
	return nil
}

// AfterExecuteTransactions called after processes transactions of the block
func (p *ProcessBase) AfterExecuteTransactions(b *Block, ctw *ContextWrapper) error {
	return nil
}

// OnSaveData called when the context of the block saved
func (p *ProcessBase) OnSaveData(b *Block, ctw *ContextWrapper) error {
	return nil
}
