package util

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/hhio618/go-golem/pkg/props"
	"github.com/hhio618/go-golem/pkg/storage"
)

type CommnadContainer struct {
	Commands []map[string]interface{}
}

func KwArgs(args ...interface{}) map[string]interface{} {
	kwargs := make(map[string]interface{})
	if len(args)%2 != 0 {
		panic(errors.New("invalid kwargs"))
	}
	for i := 0; i < len(args)/2; i++ {

		kwargs[args[2*i].(string)] = args[2*i+1]
	}
	return kwargs

}

func (c *CommnadContainer) AddCommand(item string, kwargs map[string]interface{}) int {
	_kwargs := make(map[string]interface{})
	for k, v := range kwargs {
		_kwargs[strings.TrimLeft(k, "_")] = v
	}
	idx := len(c.Commands)
	c.Commands = append(c.Commands, _kwargs)
	return idx
}

type Worker interface {
	Prepare() error
	Register(commands CommnadContainer) error
	Post(ctx context.Context) error
	Timeout() *time.Duration
}

type initStep struct {
	Worker
}

func Register(commands CommnadContainer) error {
	commands.AddCommand("deploy", KwArgs())
	commands.AddCommand("start", KwArgs())
	return nil
}

type Uploader interface {
	DoUpload(provider storage.StorageProvider) error
}

type SendWorker interface {
	Worker
	Uploader
}

type baseSendWork struct {
	uploader Uploader
	storage  storage.StorageProvider
}

func (i *baseSendWork) Prepare() error {
	return i.uploader.DoUpload(i.storage)
}

func newBaseSendWork(uploader Uploader,
	storage storage.StorageProvider) *baseSendWork {
	return &baseSendWork{
		uploader: uploader,
		storage:  storage,
	}

}

type sendWork struct {
	SendWorker
	*baseSendWork
	destPath string
	src      storage.Source
	idx      int
}

func (i *sendWork) Register(commands CommnadContainer) error {
	if i.src == nil {
		return errors.New("cmd prepared")
	}
	i.idx = commands.AddCommand("transfer",
		KwArgs(
			"_from", i.src.DownloadUrl(),
			"_to", fmt.Sprintf("container:%v", i.destPath),
		))
	return nil
}

func NewSendWork(storage storage.StorageProvider,
	destPath string) *sendWork {
	s := &sendWork{
		destPath: destPath,
		idx:      -1,
	}
	s.baseSendWork = newBaseSendWork(s, storage)
	return s
}

type sendBytes struct {
	SendWorker
	*sendWork
	data []byte
}

func (i *sendBytes) DoUpload(storage storage.StorageProvider) (err error) {
	if i.data == nil {
		return errors.New("buffer unintialized")
	}
	i.src, err = storage.UploadBytes(i.data)
	return err
}

func NewSendBytes(storage storage.StorageProvider,
	destPath string, data []byte) *sendBytes {
	s := &sendBytes{
		sendWork: &sendWork{
			destPath: destPath,
			idx:      -1,
		},
		data: data,
	}
	s.baseSendWork = newBaseSendWork(s, storage)
	return s
}

type sendJson struct {
	SendWorker
	*sendBytes
}

func NewSendJson(storage storage.StorageProvider,
	destPath string, _data map[string]interface{}) *sendJson {
	data, _ := json.Marshal(_data)
	s := NewSendBytes(storage, destPath, data)
	return &sendJson{sendBytes: s}
}

type sendFile struct {
	SendWorker
	*sendWork
	srcPath string
}

func (i *sendFile) DoUpload(storage storage.StorageProvider) (err error) {
	i.src, err = storage.UploadFile(i.srcPath)
	return err
}

func NewSendFile(sendWork *sendWork, srcPath, destPath string) *sendFile {
	sendWork.destPath = destPath
	return &sendFile{
		sendWork: sendWork,
		srcPath:  srcPath,
	}
}

type run struct {
	Worker
	cmd    string
	args   []string
	env    map[string]string
	stdOut *CaptureContext
	stdIn  *CaptureContext
	idx    int
}

func NewRun(cmd string,
	args []string,
	env map[string]string,
	stdOut *CaptureContext,
	stdIn *CaptureContext) *run {
	return &run{
		cmd:    cmd,
		args:   args,
		env:    env,
		stdOut: stdOut,
		stdIn:  stdIn,
		idx:    -1,
	}
}

func (self *run) Register(commands CommnadContainer) error {
	capture := make(map[string]interface{})
	if self.stdOut != nil {
		capture["stdout"] = self.stdOut.ToMap()
	}
	if self.stdIn != nil {
		capture["stdint"] = self.stdOut.ToMap()
	}
	i.idx = commands.AddCommand("run",
		KwArgs(
			"entry_point", self.cmd,
			"args", self.args,
			"capture", capture,
		))
	return nil
}

type StorageEvent struct {
	*DownloadStarted
	*DownloadFinished
}
type baseReceiveContent struct {
	*sendWork
	srcPath string
	dstPath string
	emitter func(*StorageEvent)
	dstSlot storage.IDestination
	idx     int
}

func newBaseReceiveContent(sendWork *sendWork, srcPath string, emitter func(*StorageEvent)) *baseReceiveContent {
	return &baseReceiveContent{
		sendWork: sendWork,
		srcPath:  srcPath,
		emitter:  emitter,
		idx:      -1,
	}
}

func (self *baseReceiveContent) Prepair() error {
	self.dstSlot = self.storage.NewDestination(self.destPath)
	return nil
}

func (self *baseReceiveContent) Register(commands CommnadContainer) error {
	if self.dstSlot == nil {
		return fmt.Errorf("command creation without prepare")
	}
	self.idx = commands.AddCommand("transfer",
		KwArgs(
			"_from", fmt.Sprintf("container:%v", self.srcPath),
			"to", self.dstSlot.UploadUrl(),
		))
	return nil
}

func (self *baseReceiveContent) emitDownloadStart() {
	if self.emitter != nil {
		self.emitter(
			&StorageEvent{
				DownloadStarted: &DownloadStarted{
					Path: self.srcPath,
				},
			},
		)
	}
}

func (self *baseReceiveContent) emitDownloadEnd() {
	if self.emitter != nil {
		self.emitter(
			&StorageEvent{
				DownloadFinished: &DownloadFinished{
					Path: self.destPath,
				},
			},
		)
	}
}

type recieveFile struct {
	*baseReceiveContent
}

func NewRecieveFile(b *baseReceiveContent, dstPath string) *recieveFile {
	b.dstPath = dstPath
	return &recieveFile{
		baseReceiveContent: b,
	}
}

func (self *recieveFile) Post(ctx context.Context) error {
	self.emitDownloadStart()
	if self.destPath == "" || self.dstSlot == nil {
		return fmt.Errorf("empty destination")
	}
	self.dstSlot.DownloadFile(ctx, self.dstPath)
	self.emitDownloadEnd()
	return nil
}

type recieveBytes struct {
	*baseReceiveContent
	onDownload func(interface{})
	limit      int
}

func NewRecieveByte(b *baseReceiveContent, onDownload func(interface{})) *recieveBytes {
	return &recieveBytes{
		baseReceiveContent: b,
		onDownload:         onDownload,
		limit:              storage.DownloadBytesLimitDefault,
	}
}

func (self *recieveBytes) Post(ctx context.Context) error {
	self.emitDownloadStart()
	if self.destPath == "" || self.dstSlot == nil {
		return fmt.Errorf("empty destination")
	}
	self.dstSlot.DownloadBytes(ctx, self.limit, self.onDownload, func(e error) {})
	self.emitDownloadEnd()
	return nil
}

type recieveJson struct {
	*baseReceiveContent
	onDownload func(interface{})
	limit      int
}

func NewRecieveJson(b *baseReceiveContent, onDownload func(interface{})) *recieveJson {
	return &recieveJson{
		baseReceiveContent: b,
		onDownload:         onDownload,
		limit:              storage.DownloadBytesLimitDefault,
	}
}

func (self *recieveJson) Post(ctx context.Context) error {
	self.emitDownloadStart()
	if self.destPath == "" || self.dstSlot == nil {
		return fmt.Errorf("empty destination")
	}
	self.dstSlot.DownloadBytes(ctx, self.limit, func(b interface{}) {
		bytes := b.([]byte)
		var out interface{}
		err := json.Unmarshal(bytes, out)
		if err != nil {
			fmt.Errorf("err: %v", err)
			return
		}
		self.onDownload(out)
	}, func(e error) {})
	self.emitDownloadEnd()
	return nil
}

type Steps struct {
	Worker
	steps   []Worker
	timeout time.Duration
}

func (self *Steps) Timeout() time.Duration {
	return self.timeout
}

func (self *Steps) Prepair() error {
	for _, step := range self.steps {
		err := step.Prepare()
		if err != nil {
			return err
		}
	}
	return nil
}

func (self *Steps) Register(commands CommnadContainer) error {
	for _, step := range self.steps {
		err := step.Register(commands)
		if err != nil {
			return err
		}
	}
	return nil
}

func (self *Steps) Post(ctx context.Context) error {
	for _, step := range self.steps {
		err := step.Post(ctx)
		if err != nil {
			return err
		}
	}
	return nil
}

type WorkContext struct {
	Id           string
	nodeInfo     *props.NodeInfo
	storage      storage.StorageProvider
	emitter      func(*StorageEvent)
	pendingSteps []Worker
	started      bool
}

func NewWorkContext(ctxId string,
	nodeInfo *props.NodeInfo,
	storage storage.StorageProvider,
	emitter func(*StorageEvent),
) *WorkContext {
	return &WorkContext{
		Id:           ctxId,
		nodeInfo:     nodeInfo,
		storage:      storage,
		emitter:      emitter,
		pendingSteps: make([]Worker, 0),
		started:      false,
	}
}

func (self *WorkContext) prepare() {
	if !self.started {
		self.pendingSteps = append(self.pendingSteps, &initStep{})
		self.started = true
	}
}

func (self *WorkContext) ProviderName() string {
	return self.nodeInfo.Name
}

func (self *WorkContext) SendJson(jsonPath string, data map[string]interface{}) {
	self.prepare()
	self.pendingSteps = append(self.pendingSteps,
		NewSendJson(self.storage, jsonPath, data))

}

func (self *WorkContext) SendBytes(destPath string, data []byte) {
	self.prepare()
	self.pendingSteps = append(self.pendingSteps,
		NewSendBytes(self.storage, destPath, data))

}

func (self *WorkContext) SendFile(srcPath, destPath string) {
	self.prepare()
	self.pendingSteps = append(self.pendingSteps,
		NewSendFile(self.storage, srcPath, destPath))
}

func (self *WorkContext) Run(cmd string, args []string, env map[string]string) {
	stdOut := NewCaptureContext("stream")
	stdErr := NewCaptureContext("stream")
	self.prepare()
	self.pendingSteps = append(self.pendingSteps,
		NewRun(cmd, args, env, stdOut, stdErr))
}

func (self *WorkContext) DownloadFile(srcPath, destPath string) {
	self.prepare()
	base := newBaseReceiveContent(NewSendWork(self.storage, destPath), srcPath, self.emitter)
	self.pendingSteps = append(self.pendingSteps,
		NewRecieveFile(base, destPath))
}

func (self *WorkContext) DownloadBytes(srcPath string, onDownload func([]interface{})) {
	self.prepare()
	base := newBaseReceiveContent(NewSendWork(self.storage, ""), srcPath, self.emitter)
	self.pendingSteps = append(self.pendingSteps,
		NewRecieveByte(base, onDownload))
}

func (self *WorkContext) DownloadJson(srcPath string, onDownload func([]interface{})) {
	self.prepare()
	base := newBaseReceiveContent(NewSendWork(self.storage, ""), srcPath, self.emitter)
	self.pendingSteps = append(self.pendingSteps,
		NewRecieveJson(base, onDownload))
}

func (self *WorkContext) commit(timeout time.Duration) *Steps {
	steps := make([]Worker, len(self.pendingSteps))
	copy(steps, self.pendingSteps)
	self.pendingSteps = make([]Worker, 0)
	return &Steps{steps: steps,
		timeout: timeout}
}

type CaptureMode string

const (
	Head     CaptureMode = "head"
	Tail     CaptureMode = "tail"
	HeadTail CaptureMode = "headTail"
	Stream   CaptureMode = "stream"
)

type CaptureFormat string

const (
	Bin CaptureFormat = "bin"
	Str CaptureFormat = "str"
)

type CaptureContext struct {
	mode  CaptureMode
	limit *int
	fmt   *CaptureFormat
}

func NewCaptureContext(mode CaptureMode,
	limit *int,
	_fmt *CaptureFormat) (*CaptureContext, error) {
	switch mode {
	case "", "all":
		return newCaptureContext(Head, nil, _fmt), nil
	case Head, Tail, HeadTail, Stream:
		return newCaptureContext(mode, limit, _fmt), nil
	default:
		return nil, fmt.Errorf("invalid output capture mode: %v", mode)
	}

}

func newCaptureContext(mode CaptureMode,
	limit *int,
	fmt *CaptureFormat) *CaptureContext {
	return &CaptureContext{
		mode:  mode,
		limit: limit,
		fmt:   fmt,
	}
}

func (self *CaptureContext) ToMap() map[string]interface{} {
	inner := make(map[string]interface{})
	if self.limit != nil {
		inner[string(self.mode)] = self.limit
	}
	if self.fmt != nil {
		inner["format"] = string(*self.fmt)
	}
	out := make(map[string]interface{})
	if self.IsStreaming() {
		out["stream"] = inner
	} else {
		out["atEnd"] = inner
	}
	return out

}

func (self *CaptureContext) IsStreaming() bool {
	return self.mode == Stream
}
