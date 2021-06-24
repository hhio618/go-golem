package util

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/hhio618/go-golem/pkg/event"
	"github.com/hhio618/go-golem/pkg/props"
	"github.com/hhio618/go-golem/pkg/rest"
)

type bufferedProposal struct {
	ts       time.Time
	score    float32
	proposal *rest.OfferProposal
}

type Task interface {
	Done() bool
	Cancel() error
	Error() error
}
type bufferedAgreement struct {
	agreement        *rest.Agreement
	nodeInfo         *props.NodeInfo
	workerTask       Task
	hasMultiActivity bool
}

type AgreementPool struct {
	emitter           func(interface{})
	offerBuffer       map[string]bufferedProposal
	agreements        map[string]bufferedAgreement
	log               *sync.Mutex
	rejectedProviders map[string]bool
	confirmed         int
}

func NewAgreementPool(emitter func(interface{})) *AgreementPool {
	return &AgreementPool{
		emitter:           emitter,
		offerBuffer:       make(map[string]bufferedProposal),
		agreements:        make(map[string]bufferedAgreement),
		log:               &sync.Mutex{},
		rejectedProviders: make(map[string]bool),
		confirmed:         1,
	}
}

func (self *AgreementPool) Cycle() {
	agreementsFrozenSet := make(map[string]bool)
	for agreementId := range self.agreements {
		agreementsFrozenSet[agreementId] = true
	}
	for agreementId := range agreementsFrozenSet {
		bufferedAgreement, ok := self.agreements[agreementId]
		if !ok {
			continue
		}
		task := bufferedAgreement.workerTask
		if task != nil && !task.Done() {
			//TODO: double check this behaviour.
			self.ReleaseAgreement(bufferedAgreement.agreement.Id(), task.Error() != nil)
		}
	}
}

func (self *AgreementPool) AddProposal(score float32, proposal *rest.OfferProposal) {
	self.log.Lock()
	defer self.log.Unlock()
	self.offerBuffer[proposal.Issuer()] = bufferedProposal{
		ts:       time.Now(),
		score:    score,
		proposal: proposal,
	}
}

func (self *AgreementPool) UseAgreement(cbk func(*rest.Agreement, *props.NodeInfo) Task) (Task, error) {
	self.log.Lock()
	defer self.log.Unlock()
	agreement, nodeInfo, err := self.getAgreement()
	if err != nil {
		return nil, err
	}
	task := cbk(agreement, nodeInfo)
	self.setWorker(agreement.Id(), task)
	return task, nil
}

func (self *AgreementPool) setWorker(agreementId string, task Task) error {
	bufferedAgreement, ok := self.agreements[agreementId]
	if !ok {
		return nil
	}
	if bufferedAgreement.workerTask != nil {
		return fmt.Errorf("buffered agreement worker task is not nil")
	}
	bufferedAgreement.workerTask = task
	return nil
}

func (self *AgreementPool) getAgreement() (*rest.Agreement, *props.NodeInfo, error) {
	emit := self.emitter

	rand.Seed(time.Now().Unix())
	agreements := make([]bufferedAgreement, 0)
	for _, a := range self.agreements {
		agreements = append(agreements, a)
	}
	if len(agreements) > 0 {
		ba := agreements[rand.Intn(len(agreements))]
		fmt.Printf("Reusing agreement. id: %s", ba.agreement.Id())
		return ba.agreement, ba.nodeInfo, nil
	}

	offers := make([]bufferedProposal, 0)
	for _, a := range self.offerBuffer {
		offers = append(offers, a)
	}

	maxScoreOffers := make(map[float32][]bufferedProposal, 0)
	maxScore := float32(math.MinInt64)
	for _, bp := range offers {
		if bp.score > maxScore {
			maxScore = bp.score
		}
		if len(maxScoreOffers[bp.score]) == 0 {
			maxScoreOffers[bp.score] = []bufferedProposal{}
		}
		maxScoreOffers[bp.score] = append(maxScoreOffers[bp.score], bp)
	}
	var bp bufferedProposal
	if _, ok := maxScoreOffers[maxScore]; ok {
		bp = maxScoreOffers[maxScore][rand.Intn(len(maxScoreOffers[maxScore]))]
	}
	delete(self.offerBuffer, bp.proposal.Issuer())
	ctx := context.TODO()
	agreement, err := bp.proposal.CreateAgreement(ctx, 0)
	select {
	case <-ctx.Done():
		return nil, nil, err
	}
	if err != nil {
		emit(&event.ProposalFailed{
			ProposalEvent: event.ProposalEvent{
				PropId: bp.proposal.Id(),
			},
			HasExcInfo: event.HasExcInfo{
				ExcInfo: &event.ExcInfo{
					Err: err,
				},
			},
		})
		return nil, nil, err
	}
	agreementDetails, err := agreement.Details()
	if err != nil {
		return nil, nil, err
	}
	providerActivty := &props.Activity{}
	err = agreementDetails.ProviderView().Extract(providerActivty)
	if err != nil {
		return nil, nil, err
	}

	requesterActivity := &props.Activity{}
	err = agreementDetails.RequesterView().Extract(requesterActivity)
	if err != nil {
		return nil, nil, err
	}
	nodeInfo := &props.NodeInfo{}
	err = agreementDetails.ProviderView().Extract(nodeInfo)
	if err != nil {
		return nil, nil, err
	}
	fmt.Printf("New agreement. id: %s, provider: %s", agreement.Id(), nodeInfo)
	emit(&event.AgreementCreated{
		AgreementEvent: event.AgreementEvent{
			AgrId: agreement.Id(),
		},
		ProviderId:   bp.proposal.Issuer(),
		ProviderInfo: *nodeInfo,
	})
	if err = agreement.Confirm(); err != nil {
		emit(&event.AgreementRejected{
			AgreementEvent: event.AgreementEvent{
				AgrId: agreement.Id(),
			},
		})
		return nil, nil, err
	}
	delete(self.rejectedProviders, bp.proposal.Issuer())
	self.agreements[agreement.Id()] = bufferedAgreement{
		agreement:        agreement,
		nodeInfo:         nodeInfo,
		workerTask:       nil,
		hasMultiActivity: providerActivty.MultiActivity && requesterActivity.MultiActivity,
	}
	emit(&event.AgreementConfirmed{
		AgreementEvent: event.AgreementEvent{
			AgrId: agreement.Id(),
		},
	})
	self.confirmed += 1
	return agreement, nodeInfo, nil

}

func (self *AgreementPool) ReleaseAgreement(agreementId string, allowReuse bool) error {
	self.log.Lock()
	defer self.log.Unlock()
	bufferedAgreement, ok := self.agreements[agreementId]
	if !ok {
		return errors.New("not found")
	}
	bufferedAgreement.workerTask = nil
	if !allowReuse || !bufferedAgreement.hasMultiActivity {
		reason := map[string]string{"message": "Work cancelled", "golem.requestor.code": "Cancelled"}
		//TODO: is this a good idea?
		go self.terminateAgreement(agreementId, reason)
	}
	return nil
}

// terminateAgreement will terminate the agreement with given `agreementId`.
func (self *AgreementPool) terminateAgreement(agreementId string, reason map[string]string) {
	bufferedAgreement, ok := self.agreements[agreementId]
	if !ok {
		//TODO:
		//logger.warning("Trying to terminate agreement not in the pool. id: %s", agreement_id)
		return
	}

	provider := "<couldn't get provider name>"
	agreementDetails, err := bufferedAgreement.agreement.Details()
	if err != nil {
		//TODO:
		//logger.debug("Cannot get details for agreement %s", agreement_id, exc_info=True)
	} else {
		node := &props.NodeInfo{}
		err = agreementDetails.ProviderView().Extract(node)
		if err != nil {
			provider = node.Name
		}
	}

	//TODO:
	// logger.debug(
	// 	"Terminating agreement. id: %s, reason: %s, provider: %s",
	// 	agreement_id,
	// 	reason,
	// 	provider,
	// )
	fmt.Printf("provider: %v", provider)

	if bufferedAgreement.workerTask != nil && !bufferedAgreement.workerTask.Done() {
		//TODO:
		// logger.debug(
		// 	"Terminating agreement that still has worker. agreement_id: %s, worker: %s",
		// 	buffered_agreement.agreement.id,
		// 	buffered_agreement.worker_task,
		// )
		bufferedAgreement.workerTask.Done()
	}

	// Converting reason to a map[string]interface{} type.
	r := make(map[string]interface{})
	for k, v := range reason {
		r[k] = v
	}

	if bufferedAgreement.hasMultiActivity {
		if err := bufferedAgreement.agreement.Terminate(r); err != nil {
			//TODO:
			// logger.debug(
			// 	"Couldn't terminate agreement. id: %s, provider: %s",
			// 	buffered_agreement.agreement.id,
			// 	provider,
			// )
		}
	}

	delete(self.agreements, agreementId)
	self.emitter(event.AgreementTerminated{AgreementEvent: event.AgreementEvent{AgrId: agreementId, Reason: reason}})
}
func (self *AgreementPool) terminateAll(reason map[string]string) {
	self.log.Lock()
	defer self.log.Unlock()
	var frozen map[string]bufferedAgreement
	/* Copy Content from self.agreements to frozen*/
	for index, element := range self.agreements {
		frozen[index] = element
	}
	for agreementId := range frozen {
		//TODO:
		go self.terminateAgreement(agreementId, reason)
	}

}

func (self *AgreementPool) onAgreementTerminated(agrId string, reason map[string]string) {
	/*
		Reacts to agreement termination event

		Should be called when AgreementTerminated event is received.
	*/
	self.log.Lock()
	defer self.log.Unlock()
	bufferedAgreement, ok := self.agreements[agrId]
	if !ok {
		return
	}

	if bufferedAgreement.workerTask != nil {
		bufferedAgreement.workerTask.Cancel()
	}
	delete(self.agreements, agrId)
	self.emitter(event.AgreementTerminated{AgreementEvent: event.AgreementEvent{AgrId: agrId, Reason: reason}})

}
