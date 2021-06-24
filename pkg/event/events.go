package event

import (
	"time"

	"github.com/hhio618/go-golem/pkg/props"
)

type ExcInfo struct {
	Err error
}

type Event interface {
	ExtractExcInfo() (*ExcInfo, Event)
}

type HasExcInfo struct {
	ExcInfo *ExcInfo
}

func (e *HasExcInfo) ExtractExcInfo() (*ExcInfo, Event) {
	return e.ExcInfo, &HasExcInfo{}
}

type ComputationStarted struct {
}

func (e *ComputationStarted) ExtractExcInfo() (*ExcInfo, Event) {
	return nil, e
}

type ComputationFinished struct {
}

func (e *ComputationFinished) ExtractExcInfo() (*ExcInfo, Event) {
	return nil, e
}

type SubscriptionCreated struct {
	SubId string
}

func (e *SubscriptionCreated) ExtractExcInfo() (*ExcInfo, Event) {
	return nil, e
}

type SubscriptionFailed struct {
	Reason string
}

func (e *SubscriptionFailed) ExtractExcInfo() (*ExcInfo, Event) {
	return nil, e
}

type CollectFailed struct {
	SubId  string
	Reason string
}

func (e *CollectFailed) ExtractExcInfo() (*ExcInfo, Event) {
	return nil, e
}

type ProposalEvent struct {
	PropId string
}

func (e *ProposalEvent) ExtractExcInfo() (*ExcInfo, Event) {
	return nil, e
}

type ProposalReceived struct {
	ProposalEvent
	ProverId string
}

func (e *ProposalReceived) ExtractExcInfo() (*ExcInfo, Event) {
	return nil, e
}

type ProposalRejected struct {
	ProposalEvent
	Reason string
}

func (e *ProposalRejected) ExtractExcInfo() (*ExcInfo, Event) {
	return nil, e
}

type ProposalResponded struct {
	ProposalEvent
}

func (e *ProposalResponded) ExtractExcInfo() (*ExcInfo, Event) {
	return nil, e
}

type ProposalConfirmed struct {
	ProposalEvent
}

func (e *ProposalConfirmed) ExtractExcInfo() (*ExcInfo, Event) {
	return nil, e
}

type ProposalFailed struct {
	ProposalEvent
	HasExcInfo
}

func (e *ProposalFailed) ExtractExcInfo() (*ExcInfo, Event) {
	return nil, e
}

type NoProposalsConfirmed struct {
	NumOffers int
	Timeout   time.Duration
}

func (e *NoProposalsConfirmed) ExtractExcInfo() (*ExcInfo, Event) {
	return nil, e
}

type AgreementEvent struct {
	AgrId  string
	Reason map[string]string
}

func (e *AgreementEvent) ExtractExcInfo() (*ExcInfo, Event) {
	return nil, e
}

type AgreementCreated struct {
	AgreementEvent
	ProviderId   string
	ProviderInfo props.NodeInfo
}

func (e *AgreementCreated) ExtractExcInfo() (*ExcInfo, Event) {
	return nil, e
}

type AgreementConfirmed struct {
	AgreementEvent
}

func (e *AgreementConfirmed) ExtractExcInfo() (*ExcInfo, Event) {
	return nil, e
}

type AgreementRejected struct {
	AgreementEvent
}

func (e *AgreementRejected) ExtractExcInfo() (*ExcInfo, Event) {
	return nil, e
}

type AgreementTerminated struct {
	AgreementEvent
}

func (e *AgreementTerminated) ExtractExcInfo() (*ExcInfo, Event) {
	return nil, e
}

type DebitNoteReceived struct {
	AgreementEvent
	NoteId string
	Amount string
}

func (e *DebitNoteReceived) ExtractExcInfo() (*ExcInfo, Event) {
	return nil, e
}

type PaymentPrepared struct {
	AgreementEvent
}

func (e *PaymentPrepared) ExtractExcInfo() (*ExcInfo, Event) {
	return nil, e
}

type PaymentQueued struct {
	AgreementEvent
}

func (e *PaymentQueued) ExtractExcInfo() (*ExcInfo, Event) {
	return nil, e
}

type PaymentFailed struct {
	AgreementEvent
	HasExcInfo
}

func (e *PaymentFailed) ExtractExcInfo() (*ExcInfo, Event) {
	return nil, e
}

type InvoiceReceived struct {
	AgreementEvent
	InvId  string
	Amount string
}

func (e *InvoiceReceived) ExtractExcInfo() (*ExcInfo, Event) {
	return nil, e
}

type WorkerStarted struct {
	AgreementEvent
}

func (e *WorkerStarted) ExtractExcInfo() (*ExcInfo, Event) {
	return nil, e
}

type ActivityCreated struct {
	AgreementEvent
	ActId string
}

func (e *ActivityCreated) ExtractExcInfo() (*ExcInfo, Event) {
	return nil, e
}

type ActivityCreateFailed struct {
	AgreementEvent
	HasExcInfo
}

func (e *ActivityCreateFailed) ExtractExcInfo() (*ExcInfo, Event) {
	return nil, e
}

type TaskEvent struct {
	TaskData interface{}
}

func (e *TaskEvent) ExtractExcInfo() (*ExcInfo, Event) {
	return nil, e
}

type TaskStarted struct {
	TaskEvent
	AgreementEvent
}

func (e *TaskStarted) ExtractExcInfo() (*ExcInfo, Event) {
	return nil, e
}

type WorkerFinished struct {
	HasExcInfo
	AgreementEvent
}

func (e *WorkerFinished) ExtractExcInfo() (*ExcInfo, Event) {
	return nil, e
}

type ScriptEvent struct {
	AgreementEvent
	TaskId string
}

func (e *ScriptEvent) ExtractExcInfo() (*ExcInfo, Event) {
	return nil, e
}

type ScriptSent struct {
	ScriptEvent
	Cmds interface{}
}

func (e *ScriptSent) ExtractExcInfo() (*ExcInfo, Event) {
	return nil, e
}

type GettingResults struct {
	ScriptEvent
}

func (e *GettingResults) ExtractExcInfo() (*ExcInfo, Event) {
	return nil, e
}

type ScriptFinished struct {
	ScriptEvent
}

func (e *ScriptFinished) ExtractExcInfo() (*ExcInfo, Event) {
	return nil, e
}

type CommandEvent struct {
	ScriptEvent
	CmdIdx int
}

func (e *CommandEvent) ExtractExcInfo() (*ExcInfo, Event) {
	return nil, e
}

type CommandEventContext struct {
	EvtCls interface{}
	Kwargs map[string]interface{}
}

func (cec *CommandEventContext) ComputationFinished(lastIndex int) bool {
	switch cec.EvtCls.(type) {
	case CommandExecuted:
		if v, ok := cec.Kwargs["cmd_idx"]; ok {
			if v.(int) < lastIndex {
				return false
			}
			if _, ok := cec.Kwargs["success"]; !ok {
				return true
			}
		}
	default:
		return false
	}
	return false
}

type CommandExecuted struct {
	CommandEvent
	Command interface{}
	Failed  bool
	Message string
}

func (e *CommandExecuted) ExtractExcInfo() (*ExcInfo, Event) {
	return nil, e
}

type CommandStarted struct {
	CommandEvent
	Command string
}

func (e *CommandStarted) ExtractExcInfo() (*ExcInfo, Event) {
	return nil, e
}

type CommandStdOut struct {
	CommandEvent
	Output string
}

func (e *CommandStdOut) ExtractExcInfo() (*ExcInfo, Event) {
	return nil, e
}

type CommandStdErr struct {
	CommandEvent
	Output string
}

func (e *CommandStdErr) ExtractExcInfo() (*ExcInfo, Event) {
	return nil, e
}

type TaskAccepted struct {
	TaskEvent
	Result interface{}
}

func (e *TaskAccepted) ExtractExcInfo() (*ExcInfo, Event) {
	return nil, e
}

type TaskRejected struct {
	TaskEvent
	Reason string
}

func (e *TaskRejected) ExtractExcInfo() (*ExcInfo, Event) {
	return nil, e
}

type DownloadStarted struct {
	Event
	Path string
}

func (e *DownloadStarted) ExtractExcInfo() (*ExcInfo, Event) {
	return nil, e
}

type DownloadFinished struct {
	Event
	Path string
}

func (e *DownloadFinished) ExtractExcInfo() (*ExcInfo, Event) {
	return nil, e
}

type ShutdownFinished struct {
	HasExcInfo
}

func (e *ShutdownFinished) ExtractExcInfo() (*ExcInfo, Event) {
	return nil, e
}
