package rest

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/go-kit/kit/log"
	level "github.com/go-kit/kit/log/level"
	"github.com/hhio618/go-golem/pkg/executer"
	activity "github.com/hhio618/ya-go-client/ya-activity"
	sse "github.com/r3labs/sse/v2"
)

// ActivityService
type ActivityService struct {
	ctx    context.Context
	client *activity.APIClient
	api    *activity.RequestorControlApiService
	state  *activity.RequestorStateApiService
	logger log.Logger
}

func NewActivityService(ctx context.Context, client *activity.APIClient, logger log.Logger) *ActivityService {
	return &ActivityService{
		ctx:    ctx,
		client: client,
		api:    client.RequestorControlApi,
		state:  client.RequestorStateApi,
		logger: logger,
	}
}

func (as *ActivityService) NewActivity(agreementId string) (*Activity, error) {
	res, _, err := as.api.CreateActivity(as.ctx).AgreementId(agreementId).Execute()
	if err != nil {
		level.Error(as.logger).Log("msg", "creating activity", "err", err)
	}
	activityId, err := strconv.Unquote(res.(string))
	if err != nil {
		return nil, err
	}
	return &Activity{ActivityService: as, id: activityId}, nil

}

type Activity struct {
	*ActivityService
	id string
}

func (a *Activity) Id() string {
	return a.id
}

func (a *Activity) State() (*activity.ActivityState, error) {
	res, _, err := a.state.GetActivityState(a.ctx, a.id).Execute()
	if err != nil {
		level.Error(a.logger).Log("msg", "getting activity state", "err", err)
	}
	return &res, nil
}

func (a *Activity) Send(script []map[string]interface{}, stream bool, deadline time.Time) (Poller, error) {
	scriptText, err := json.Marshal(script)
	if err != nil {
		return nil, err
	}
	req := activity.NewExeScriptRequest(string(scriptText))
	res, _, err := a.api.Exec(a.ctx, a.id).Script(*req).Execute()
	if err != nil {
		level.Error(a.logger).Log("msg", "calling exe", "err", err)
		return nil, err
	}
	batchId, err := strconv.Unquote(res)
	if err != nil {
		return nil, err
	}
	if stream {
		return NewStreamingBatch(a.logger, a.client, a.api, a.id, batchId, len(script), deadline), nil
	}
	return NewPollingBatch(a.logger, a.api, a.id, batchId, len(script), deadline), nil
}

func (a *Activity) DestroyActivity(excType, excVal, excTb interface{}) {
	if excType != nil {
		level.Debug(a.logger).Log("msg", "destroying activity", "id", a.id, "execType", excType, "execVal", excVal, "excTb", excTb)
	} else {
		level.Debug(a.logger).Log("msg", "destroying activity", "id", a.id)
	}
	_, err := a.api.DestroyActivity(a.ctx, a.id).Execute()
	if err != nil {
		level.Debug(a.logger).Log("msg", "got API Exception when destroying activity", "id", a.id, "err", err)
	}
	level.Debug(a.logger).Log("msg", "activity destroyed successfully", "id", a.id)
}

type Result struct {
	Idx     int
	Message string
}

type CommandExecutionError struct {
	Command string
	Message string
}

func (c CommandExecutionError) Error() string {
	msg := fmt.Sprintf("Command %s failed on provider", c.Command)
	if c.Message != "" {
		msg = fmt.Sprintf("%v with message '%v'", msg, c.Message)
	}
	return msg
}

type BatchTimeoutError struct {
}

func (b *BatchTimeoutError) Error() string {
	return "batch timeout error"
}

type Batch struct {
	logger     log.Logger
	api        *activity.RequestorControlApiService
	activityId string
	batchId    string
	size       int
	deadline   time.Time
}

type Poller interface {
	Poll(ctx context.Context) (eventCh chan *executer.CommandEventContext, errCh chan error)
}

func (b *Batch) SecondsLeft() float32 {
	return float32(b.deadline.Sub(time.Now()).Seconds())
}
func (b *Batch) Id() string {
	return b.batchId
}

type PollingBatch struct {
	Batch
}

func NewPollingBatch(logger log.Logger, api *activity.RequestorControlApiService, activityId, batchId string, size int, deadline time.Time) *PollingBatch {
	return &PollingBatch{
		Batch: Batch{
			logger:     logger,
			api:        api,
			activityId: activityId,
			batchId:    batchId,
			size:       size,
			deadline:   deadline,
		},
	}
}

func (pb *PollingBatch) Poll(ctx context.Context) (eventCh chan *executer.CommandEventContext, errCh chan error) {
	errCh = make(chan error)
	eventCh = make(chan *executer.CommandEventContext)
	lastIdx := 0
	go func() {
		for lastIdx < pb.size {
			select {
			case <-ctx.Done():
				break
			default:
			}
			timeout := pb.SecondsLeft()
			if timeout <= 0 {
				errCh <- &BatchTimeoutError{}
				break
			}
			getBatchCtx, cncl := context.WithTimeout(ctx, time.Second*time.Duration(math.Min(float64(timeout), 5)))
			defer cncl()
			results, resp, err := pb.api.GetExecBatchResults(getBatchCtx, pb.activityId, pb.batchId).Execute()
			select {
			case <-getBatchCtx.Done():
				continue
			default:
			}
			if err != nil {
				if resp.StatusCode == 408 {
					continue
				}
				errCh <- err
				break
			}
			anyNew := false
			results = results[lastIdx:]
			for _, result := range results {
				anyNew = true
				if lastIdx != int(result.Index) {
					errCh <- fmt.Errorf("expected %v, got %v", lastIdx, result.Index)
					break
				}
				message := ""
				if result.Message != nil {
					message = *result.Message
				} else if result.Stdout != nil || result.Stderr != nil {
					_message, err := json.Marshal(map[string]string{"stdout": *result.Stdout, "stderr": *result.Stderr})
					if err != nil {
						errCh <- err
					}
					message = string(_message)
				}
				kwargs := map[string]interface{}{
					"cmd_idx": result.Index,
					"message": message,
					"success": (strings.ToLower(result.Result) == "ok"),
				}
				eventCh <- &executer.CommandEventContext{
					EvtCls: executer.CommandExecuted,
					Kwargs: kwargs,
				}
				lastIdx = int(result.Index + 1)
				if *result.IsBatchFinished {
					break
				}

			}
			if !anyNew {
				delay := int(math.Min(3, math.Max(0, float64(pb.SecondsLeft()))))
				time.Sleep(time.Duration(delay) * time.Second)
			}
		}
	}()
	return eventCh, errCh
}

type StreamingBatch struct {
	Batch
	client *activity.APIClient
}

func NewStreamingBatch(logger log.Logger, client *activity.APIClient, api *activity.RequestorControlApiService, activityId, batchId string, size int, deadline time.Time) *StreamingBatch {
	return &StreamingBatch{
		Batch: Batch{
			logger:     logger,
			api:        api,
			activityId: activityId,
			batchId:    batchId,
			size:       size,
			deadline:   deadline,
		},
		client: client,
	}
}

func (sb *StreamingBatch) Poll(ctx context.Context) (eventCh chan *executer.CommandEventContext, errCh chan error) {
	errCh = make(chan error)
	eventCh = make(chan *executer.CommandEventContext)
	lastIdx := 0
	host := sb.client.GetConfig().Host
	client := sse.NewClient(fmt.Sprintf("%v/activity/%v/exec/%v", host, sb.activityId, sb.batchId))
	//TODO: emulate         api_client.update_params_for_auth(headers, None, ["app_key"]).
	client.Headers = sb.client.GetConfig().DefaultHeader

	var getBatchCtx context.Context
	var cncl context.CancelFunc
	sseCh := make(chan *sse.Event)
	for {
		select {
		case <-ctx.Done():
			break
		default:
		}
		getBatchCtx, cncl = context.WithTimeout(ctx, time.Duration(sb.SecondsLeft())*time.Second)
		defer cncl()
		err := client.SubscribeChanWithContext(getBatchCtx, "", sseCh)
		if err != nil {
			// Retrying in 5 seconds.
			time.Sleep(5 * time.Second)
			continue
		}
		break
	}
	go func() {
		for {
			select {
			case <-ctx.Done():
			case <-getBatchCtx.Done():
				return
			case evt := <-sseCh:
				{
					evtCtx, err := commandEventCtx(evt)
					if err != nil {
						level.Debug(sb.logger).Log("msg", "event stream exception", "batchId", sb.batchId, err, "err")
					}
					if evtCtx.ComputationFinished(lastIdx) {
						return
					}

				}
			default:
			}
		}

	}()

	return eventCh, errCh
}

func commandEventCtx(evt *sse.Event) (*executer.CommandEventContext, error) {
	if string(evt.Event) != "runtime" {
		return nil, fmt.Errorf("unsupported event: %v", string(evt.Event))
	}
	evtMap := make(map[string]interface{})
	err := json.Unmarshal(evt.Data, evtMap)
	if err != nil {
		return nil, err
	}
	evtKinds, ok := evtMap["kind"]
	if !ok {
		return nil, fmt.Errorf("missing kind")
	}
	var evtKind string
	var evtCls executer.CommandClass
	Kwargs := map[string]interface{}{
		"cmd_id": evtMap["index"],
	}
	switch concreteVal := evtKinds.(type) {
	case map[string]interface{}:
		for k := range concreteVal {
			evtKind = k
			break
		}
		if evtKind == "" {
			return nil, fmt.Errorf("empty kind")
		}
		evtData := concreteVal[evtKind]

		switch evtKind {
		case "started":
			switch x := evtData.(type) {
			case map[string]interface{}:
				command, ok := x["command"]
				if !ok {
					return nil, fmt.Errorf("invalid CommandStarted event: missing 'command'")
				}
				evtCls = executer.CommandStarted
				Kwargs["command"] = command
			default:
				return nil, fmt.Errorf("invalid CommandStarted event: missing 'command'")
			}
		case "finished":
			switch x := evtData.(type) {
			case map[string]interface{}:
				_return_code, ok := x["return_code"]
				if !ok {
					return nil, fmt.Errorf("invalid CommandStarted event: missing 'return_code'")
				}
				return_code, err := strconv.Atoi(fmt.Sprintf("%v", _return_code))
				if err != nil {
					return nil, err
				}
				evtCls = executer.CommandExecuted
				Kwargs["success"] = return_code == 0
				Kwargs["message"], _ = x["message"]
			default:
				return nil, fmt.Errorf("invalid CommandStarted event: missing 'return_code'")
			}
		case "stdout":
			evtCls = executer.CommandStdOut
			Kwargs["output"] = fmt.Sprintf("%v", evtData)

		case "stderr":
			evtCls = executer.CommandStdErr
			Kwargs["output"] = fmt.Sprintf("%v", evtData)
		default:
			return nil, fmt.Errorf("unsupported runtime event: %v", evtKind)
		}
	default:
		return nil, fmt.Errorf("unknown kind")
	}
	return &executer.CommandEventContext{EvtCls: evtCls, Kwargs: Kwargs}, nil

}
