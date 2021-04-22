package rest

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/hhio618/go-golem/pkg/props"
	yam "github.com/hhio618/ya-go-client/ya-market"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
)

type view struct {
	Properties props.Props
}

func (v *view) Extract(model props.Model) error {
	return props.FromProperties(v.Properties, model)
}

type AgreementDetails struct {
	RawDetails *yam.Agreement
}

func (ad *AgreementDetails) ProviderView() *view {
	offer := ad.RawDetails.Offer
	return &view{Properties: offer.Properties.(map[string]interface{})}
}

func (ad *AgreementDetails) RequesterView() *view {
	demand := ad.RawDetails.Demand
	return &view{Properties: demand.Properties.(map[string]interface{})}
}

type Agreement struct {
	ctx          context.Context
	logger       log.Logger
	api          *yam.RequestorApiService
	subscription Subscription
	id           string
}

func (a *Agreement) Id() string {
	return a.id
}

func (a *Agreement) Details() (*AgreementDetails, error) {
	detail, _, err := a.api.GetAgreement(a.ctx, a.id).Execute()
	if err != nil {
		return nil, err
	}
	return &AgreementDetails{RawDetails: &detail}, nil
}

func (a *Agreement) Confirm() error {
	_, err := a.api.ConfirmAgreement(a.ctx, a.id).Execute()
	if err != nil {
		level.Debug(a.logger).Log("msg", "wait for approval", "err", err)
		return err
	}
	waitCtx, cncl := context.WithTimeout(a.ctx, 16*time.Second)
	defer cncl()
	_, err = a.api.WaitForApproval(waitCtx, a.id).Timeout(15).Execute()
	select {
	case <-waitCtx.Done():
		level.Debug(a.logger).Log("msg", "client-side timeout")
		return fmt.Errorf("client-side timeout")
	default:
	}
	if err != nil {
		level.Debug(a.logger).Log("msg", "wait for approval", "err", err)
		return err
	}
	return nil
}

func (a *Agreement) Terminate(reason map[string]interface{}) error {
	resp, err := a.api.TerminateAgreement(a.ctx, a.id).
		RequestBody(reason).
		Execute()
	if err != nil {
		level.Debug(a.logger).Log("msg", "terminate agreement", "err", err)

		if resp.StatusCode == 410 {
			var jsonObj map[string]interface{}
			bodyBytes, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				level.Debug(a.logger).Log("msg",
					"read terminate agreement response", "err", err)
				return err
			}
			err = json.Unmarshal(bodyBytes, &jsonObj)
			if err != nil {
				level.Debug(a.logger).Log("msg",
					"unmarshal terminate agreement", "err", err)
				return err
			}
			level.Debug(a.logger).Log("msg", "terminateAgreement error",
				"status", 410, "id", a.id,
				"message", jsonObj["message"])
			return err
		}
	}
	level.Info(a.logger).Log("msg",
		"terminateAgreement returned successfully", "id", a.id)
	return nil
}

var (
	OfferProposalSlots = []string{"_proposal", "_subscription"}
)

type OfferProposal struct {
	ctx          context.Context
	logger       log.Logger
	proposal     *yam.ProposalEvent
	subscription Subscription
}

func (o *OfferProposal) Issuer() string {
	return o.proposal.Proposal.IssuerId
}

func (o *OfferProposal) Id() string {
	return o.proposal.Proposal.ProposalId
}

func (o *OfferProposal) Props() props.Props {
	return o.proposal.Proposal.Properties.(props.Props)
}

func (o *OfferProposal) IsDraft() bool {
	return o.proposal.Proposal.State == "draft"
}

func (o *OfferProposal) Reject(reason string) error {
	if reason == "" {
		reason = "Rejected"
	}
	_, err := o.subscription.api.
		RejectProposalOffer(o.ctx, o.subscription.id, o.Id()).
		RequestBody(map[string]interface{}{"message": reason}).
		Execute()
	return err
}

func (o *OfferProposal) Respond(props props.Props, constraints string) (string, error) {
	proposal := yam.DemandOfferBase{
		Properties:  props,
		Constraints: constraints,
	}
	demandCtx, cncl := context.WithTimeout(o.ctx, time.Second*5)
	defer cncl()
	newProposal, _, err := o.subscription.api.CounterProposalDemand(demandCtx, o.subscription.id, o.Id()).
		DemandOfferBase(proposal).Execute()
	select {
	case <-demandCtx.Done():
		return "", errors.Wrap(err, "timeout exceeded")
	default:
	}
	return newProposal, err
}

func (o *OfferProposal) String() string {
	return fmt.Sprintf("OfferProposal(%v, %v, %v)",
		o.proposal.Proposal.ProposalId,
		o.proposal.Proposal.State,
		o.proposal.Proposal.IssuerId)
}
func (o *OfferProposal) CreateAgreement(ctx context.Context, timeout time.Duration) (*Agreement, error) {
	if timeout == 0 {
		timeout = time.Hour
	}
	proposal := yam.AgreementProposal{
		ProposalId: o.Id(),
		ValidTo:    time.Now().Add(timeout),
	}

	newProposal, _, err := o.subscription.api.CreateAgreement(ctx).
		AgreementProposal(proposal).Execute()
	if err != nil {
		return nil, err
	}
	return &Agreement{
		ctx:          ctx,
		api:          o.subscription.api,
		subscription: o.subscription,
		id:           newProposal,
	}, nil
}

type Subscription struct {
	ctx     context.Context
	logger  log.Logger
	api     *yam.RequestorApiService
	id      string
	open    bool
	deleted bool
	details *yam.Demand
}

func NewSubscription(logger log.Logger, ctx context.Context, api *yam.RequestorApiService,
	id string,
	open bool,
	deleted bool,
	details *yam.Demand) *Subscription {
	return &Subscription{
		logger:  logger,
		ctx:     ctx,
		api:     api,
		id:      id,
		open:    true,
		deleted: false,
		details: details,
	}
}

func (s *Subscription) Id() string {
	return s.id
}

func (s *Subscription) Close() {
	s.open = false
}

func (s *Subscription) Start() error {
	return nil
}

func (s *Subscription) Stop(excType, excValue, traceback interface{}) error {
	return s.Delete()
}

func (s *Subscription) ValidateDetails() error {
	if s.details == nil {
		return fmt.Errorf("expected details on list object")
	}
	return nil
}

func (s *Subscription) Delete() error {
	s.open = false
	if !s.deleted {
		_, err := s.api.UnsubscribeDemand(s.ctx, s.id).Execute()
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Subscription) Events(ctx context.Context) chan *OfferProposal {
	proposalCh := make(chan *OfferProposal)
	go func() {
		for s.open {
			select {
			case <-ctx.Done():
				return
			default:
			}
			_, resp, err := s.api.CollectOffers(ctx, s.id).Timeout(10).
				MaxEvents(10).Execute()
			if err != nil {
				//TODO: log this.
				time.Sleep(1 * time.Second)
				continue
			}
			var events []map[string]interface{}
			bodyBytes, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				//TODO: log this.
				time.Sleep(1 * time.Second)
				continue
			}
			err = json.Unmarshal(bodyBytes, &events)
			if err != nil {
				//TODO: log this.
				time.Sleep(1 * time.Second)
				continue
			}

			for _, ev := range events {
				if eType, ok := ev["eventType"]; ok {
					switch eType {
					case "ProposalEvent":
						_, ok := ev["proposal"]
						if !ok {
							//TODO: log this.
							time.Sleep(1 * time.Second)
							continue
						}
						proposalEvent := &yam.ProposalEvent{}
						err := mapstructure.Decode(ev, proposalEvent)
						if err != nil {
							//TODO: log this.
							time.Sleep(1 * time.Second)
							continue
						}
						proposalCh <- &OfferProposal{proposal: proposalEvent}
					default:
						time.Sleep(1 * time.Second)
						continue
					}
				}
			}
		}

	}()
	return proposalCh
}

type Market struct {
	ctx    context.Context
	logger log.Logger
	api    *yam.RequestorApiService
}

func (m *Market) Subscribe(props props.Props, constraints string) (*Subscription, error) {
	proposal := yam.DemandOfferBase{
		Properties:  props,
		Constraints: constraints,
	}
	id, _, err := m.api.SubscribeDemand(m.ctx).DemandOfferBase(proposal).Execute()
	if err != nil {
		return nil, err
	}
	return &Subscription{
		api: m.api,
		id:  id,
	}, nil
}

func (m *Market) Subscriptions() ([]Subscription, error) {
	subscriptions := make([]Subscription, 0)
	demands, _, err := m.api.GetDemands(m.ctx).Execute()
	if err != nil {
		return nil, err
	}
	for _, d := range demands {
		subscriptions = append(subscriptions, Subscription{
			api:     m.api,
			details: &d,
			id:      d.DemandId,
		})
	}
	return subscriptions, nil
}
