package executer

type CommandEventContext struct {
	EvtCls interface{}
	Kwargs map[string]interface{}
}

func (cec *CommandEventContext) ComputationFinished(lastIndex int) bool {
	return false
}

type CommandClass int

const (
	CommandExecuted CommandClass = iota + 1
	CommandStarted
	CommandStdOut
	CommandStdErr
)
