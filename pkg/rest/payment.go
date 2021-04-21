package rest

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	yap "github.com/hhio618/ya-go-client/ya-payment"
	"github.com/shopspring/decimal"
)

type Resourcer interface {
	Source() error
	UnSource() error
}

type Invoice struct {
	ctx     context.Context
	api     *yap.RequestorApiService
	invoice *yap.Invoice
}

func NewInvoice(ctx context.Context, api *yap.RequestorApiService, invoice *yap.Invoice) *Invoice {
	return &Invoice{
		ctx:     ctx,
		api:     api,
		invoice: invoice,
	}
}

func (i *Invoice) Accept(amount string, allocation Allocation) (*http.Response, error) {
	acceptance := yap.NewAcceptance(amount, allocation.Id)
	res, err := i.api.AcceptInvoice(i.ctx, i.invoice.InvoiceId).Acceptance(*acceptance).Execute()
	return res, err
}

type DebitNote struct {
	ctx       context.Context
	api       *yap.RequestorApiService
	debitNote *yap.DebitNote
}

func NewDebitNote(ctx context.Context, api *yap.RequestorApiService, debitNote *yap.DebitNote) *DebitNote {
	return &DebitNote{
		ctx:       ctx,
		api:       api,
		debitNote: debitNote,
	}
}

func (d *DebitNote) Accept(amount string, allocation Allocation) (*http.Response, error) {
	acceptance := yap.NewAcceptance(amount, allocation.Id)
	res, err := d.api.AcceptDebitNote(d.ctx, d.debitNote.DebitNoteId).Acceptance(*acceptance).Execute()
	return res, err
}

type link struct {
	ctx context.Context
	api *yap.RequestorApiService
}

type AllocationDetails struct {
	spentAmount     decimal.Decimal
	remainingAmount decimal.Decimal
}

type Allocation struct {
	link
	Id     string
	Amount decimal.Decimal

	PaymentPlatform string
	PaymentAddress  string
	Expires         time.Time
}

func (a *Allocation) Details() (*AllocationDetails, error) {
	details, _, err := a.api.GetAllocation(a.ctx, a.Id).Execute()
	if err != nil {
		return nil, err
	}
	spentAmount, err := decimal.NewFromString(details.SpentAmount)
	if err != nil {
		return nil, err
	}
	remainingAmount, err := decimal.NewFromString(details.RemainingAmount)
	if err != nil {
		return nil, err
	}
	return &AllocationDetails{
		spentAmount:     spentAmount,
		remainingAmount: remainingAmount,
	}, nil
}

func (a *Allocation) Delete() error {
	_, err := a.api.ReleaseAllocation(a.ctx, a.Id).Execute()
	if err != nil {
		return err
	}
	return nil
}

type allocationTask struct {
	allocation *Allocation
	api        *yap.RequestorApiService
	Model      *yap.Allocation
	id         string
}

func (a *allocationTask) Alocate() (*Allocation, error) {
	newAllocation, _, err := a.api.CreateAllocation(a.allocation.ctx).Allocation(*a.Model).Execute()
	if err != nil {
		return nil, err
	}
	a.id = newAllocation.AllocationId
	if a.Model.TotalAmount == "" {
		return nil, fmt.Errorf("total amount is empty")
	}
	if a.Model.Timeout == nil {
		return nil, fmt.Errorf("total amount is empty")
	}
	if a.id == "" {
		return nil, fmt.Errorf("id is blank")
	}
	amount, err := decimal.NewFromString(newAllocation.TotalAmount)
	if err != nil {
		return nil, err
	}
	return &Allocation{
		Id:              newAllocation.AllocationId,
		Amount:          amount,
		PaymentPlatform: *newAllocation.PaymentPlatform,
		PaymentAddress:  *newAllocation.Address,
		Expires:         *newAllocation.Timeout,
	}, nil

}

func (a *allocationTask) DeAllocate() error {
	if a.id != "" {
		_, err := a.api.ReleaseAllocation(a.allocation.ctx, a.id).Execute()
		if err != nil {
			return err
		}
	}
	return nil
}

const Slots = "_api"

type Payment struct {
	api *yap.RequestorApiService
}

func NewPayment(client *yap.APIClient) *Payment {
	return &Payment{
		api: client.RequestorApi,
	}
}

func (p *Payment) NewAllocation(amount decimal.Decimal,
	paymentPlatform string,
	paymentAddress string,
	expires *time.Time,
	makeDeposit bool) *allocationTask {
	var allocationTimeout time.Time
	if expires == nil {
		allocationTimeout = time.Now().UTC().Add(time.Minute * 30)
	}
	return &allocationTask{
		api: p.api,
		Model: &yap.Allocation{
			AllocationId:    "",
			PaymentPlatform: &paymentPlatform,
			Address:         &paymentAddress,
			TotalAmount:     amount.String(),
			Timeout:         &allocationTimeout,
			MakeDeposit:     makeDeposit,
			SpentAmount:     "",
			RemainingAmount: "",
		},
	}

}

func (p *Payment) Allocations(ctx context.Context) ([]Allocation, error) {
	allocations := make([]Allocation, 0)
	_allocations, _, err := p.api.GetAllocations(ctx).Execute()
	if err != nil {
		return nil, err
	}
	for _, a := range _allocations {
		amount, err := decimal.NewFromString(a.TotalAmount)
		if err != nil {
			return nil, err
		}
		allocations = append(allocations, Allocation{
			Id:              a.AllocationId,
			Amount:          amount,
			PaymentPlatform: *a.PaymentPlatform,
			PaymentAddress:  *a.Address,
			Expires:         *a.Timeout,
		})
	}
	return allocations, nil
}

func (p *Payment) Allocation(ctx context.Context, allocationId string) (*Allocation, error) {
	a, _, err := p.api.GetAllocation(ctx, allocationId).Execute()
	if err != nil {
		return nil, err
	}
	amount, err := decimal.NewFromString(a.TotalAmount)
	if err != nil {
		return nil, err
	}
	return &Allocation{
		Id:              a.AllocationId,
		Amount:          amount,
		PaymentPlatform: *a.PaymentPlatform,
		PaymentAddress:  *a.Address,
		Expires:         *a.Timeout,
	}, nil
}

func (p *Payment) Accounts(ctx context.Context, allocationId string) ([]yap.Account, error) {
	accounts, _, err := p.api.GetRequestorAccounts(ctx).Execute()
	if err != nil {
		return nil, err
	}
	return accounts, nil
}

func (p *Payment) DecorateDemand(ctx context.Context, ids []string) (*yap.MarketDecoration, error) {
	res, _, err := p.api.GetDemandDecorations(ctx).AllocationIds(ids).Execute()
	return &res, err
}

func (p *Payment) DebitNote(ctx context.Context, debitNoteId string) (*DebitNote, error) {
	res, _, err := p.api.GetDebitNote(ctx, debitNoteId).Execute()
	return &DebitNote{
		debitNote: &res,
		api:       p.api,
	}, err
}

func (p *Payment) Invoices(ctx context.Context) ([]Invoice, error) {
	invoices := make([]Invoice, 0)
	res, _, err := p.api.GetInvoices(ctx).Execute()
	if err != nil {
		return nil, err
	}
	for _, inv := range res {
		invoices = append(invoices, Invoice{
			invoice: &inv,
			api:     p.api,
		})
	}
	return invoices, nil
}

func (p *Payment) Invoice(ctx context.Context, invoiceId string) (*Invoice, error) {
	res, _, err := p.api.GetInvoice(ctx, invoiceId).Execute()
	if err != nil {
		return nil, err
	}

	return &Invoice{
		api:     p.api,
		invoice: &res,
	}, nil
}

func (p *Payment) IncomingInvoice(ctx context.Context) (chan *Invoice, error) {
	ts := time.Now().UTC()
	invCh := make(chan *Invoice)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			_, resp, err := p.api.GetInvoiceEvents(ctx).AfterTimestamp(ts).Execute()
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
			err = json.Unmarshal(bodyBytes, events)
			if err != nil {
				//TODO: log this.
				time.Sleep(1 * time.Second)
				continue
			}
			for _, ev := range events {
				if eType, ok := ev["eventType"]; ok {
					switch eType {
					case "InvoiceReceivedEvent":
						ts, _ = ev["eventDate"].(time.Time)
						invId, ok := ev["invoiceId"]
						if !ok {
							//TODO: log this.
							// Empty invoice id in event.
							time.Sleep(1 * time.Second)
							continue
						}
						invoice, _, err := p.api.GetInvoice(ctx, invId.(string)).Execute()
						if err != nil {
							//TODO: log this.
							time.Sleep(1 * time.Second)
							continue
						}
						invCh <- &Invoice{invoice: &invoice, api: p.api}
					default:
						time.Sleep(1 * time.Second)
						continue
					}
				}
			}
		}

	}()
	return invCh, nil
}

func (p *Payment) IncomingDebitNotes(ctx context.Context) (chan *DebitNote, error) {
	ts := time.Now().UTC()
	debitCh := make(chan *DebitNote)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			_, resp, err := p.api.GetDebitNoteEvents(ctx).AfterTimestamp(ts).Execute()
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
			err = json.Unmarshal(bodyBytes, events)
			if err != nil {
				//TODO: log this.
				time.Sleep(1 * time.Second)
				continue
			}
			for _, ev := range events {
				if eType, ok := ev["eventType"]; ok {
					switch eType {
					case "DebitNoteReceivedEvent":
						ts, _ = ev["eventDate"].(time.Time)
						debitNoteId, ok := ev["debitNoteId"]
						if !ok {
							//TODO: log this.
							// Empty debit note id in event.
							time.Sleep(1 * time.Second)
							continue
						}
						debitNote, _, err := p.api.GetDebitNote(ctx, debitNoteId.(string)).Execute()
						if err != nil {
							//TODO: log this.
							time.Sleep(1 * time.Second)
							continue
						}
						debitCh <- &DebitNote{debitNote: &debitNote, api: p.api}
					default:
						time.Sleep(1 * time.Second)
						continue
					}
				}
			}
		}

	}()
	return debitCh, nil
}
