package storage

import (
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/ybbus/jsonrpc/v2"
	"golang.org/x/crypto/sha3"
)

type PubLink struct {
	File string
	Url  string
}

var (
	CommandStatus = []string{"ok", "error"}
)

type GftpDriver interface {
}

func Service(debug bool) {

}

type process struct {
	client jsonrpc.RPCClient
	debug  bool
	done   chan bool
	lock   *sync.Mutex
	p      *exec.Cmd
}

func NewProcess(debug bool, client jsonrpc.RPCClient) GftpDriver {
	return &process{
		client: client,
		debug:  debug,
		lock:   new(sync.Mutex),
		done:   make(chan bool),
	}
}

func (p *process) Start() error {
	p.p = exec.Command("gftp server")
	if p.debug {
		p.p.Env = os.Environ()
		p.p.Env = append(p.p.Env, "RUST_LOG=debug")
	}
	err := p.p.Start()
	if err != nil {
		fmt.Printf("gftp exited with error: %v", err)
		return err
	}

	err = p.p.Wait()
	if err != nil {
		fmt.Printf("gftp returned with error: %v", err)
	}
	p.done <- true
	return err
}
func (p *process) Stop() error {
	return p.close()
}

func (p *process) SendMessage(message string) (string, error) {
	p.lock.Lock()
	defer p.lock.Unlock()

	cmdReader, err := p.p.StdoutPipe()
	if err != nil {
		fmt.Printf("gftp stdout: %v", err)
		return "", err
	}
	cmdWriter, err := p.p.StdinPipe()
	if err != nil {
		fmt.Printf("gftp stdin: %v", err)
		return "", err
	}
	//TODO: serialize message?
	msg := []byte(message + "\n")
	_, err = cmdWriter.Write(msg)
	if err != nil {
		fmt.Printf("gftp write: %v", err)
		return "", err
	}
	fmt.Printf("\n => out: %v", string(msg))
	msg, err = ioutil.ReadAll(cmdReader)
	fmt.Printf("\n <= in: %v", string(msg))
	if err != nil {
		p.p.Stderr.Write([]byte("Please check if gftp is installed and is in your $PATH.\n"))
		return "", err
	}
	//TODO: json serialize output?
	return string(msg), nil

}

func (p *process) close() error {
	p.lock.Lock()
	defer p.lock.Unlock()

	ctx, cncl := context.WithTimeout(context.TODO(), time.Second*10)
	defer cncl()
	p.p.Process.Signal(os.Interrupt)
	for {
		select {
		case <-ctx.Done():
			err := p.p.Process.Kill()

			return errors.Wrap(err, "gftp process was killed after a timeout")
		case <-p.done:
			return nil
		}
	}
}

type GftpSource struct {
	link PubLink
	len  int
}

func (g *GftpSource) DownloadUrl() string {
	return g.link.Url
}

func (g *GftpSource) ContentLength() int {
	return g.len
}

type GftpDestination struct {
	Destination
	process process
	link    PubLink
}

func (g *GftpDestination) UploadUrl() string {
	return g.link.Url
}

func (g *GftpDestination) DownloadStream() (*Content, error) {
	filePath := g.link.File
	fileBytes, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	// Stream using a buffered channel.
	stream := make(chan []byte, 1)
	stream <- fileBytes
	return &Content{
		Length: len(fileBytes),
		Stream: stream,
	}, nil

}

func (g *GftpDestination) DownloadFile(ctx context.Context, dest string) {
	if dest == g.link.File {
		return
	}
	g.Destination.DownloadFile(ctx, dest)

}

func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

type GftProvider struct {
	StorageProvider
	tmpDir           string
	process          process
	registeredSource map[string]Source
}

func NewGftProvider(_tmpDir string, process process) *GftProvider {
	var tmpDir string
	if ok, _ := exists(_tmpDir); ok {
		tmpDir = _tmpDir
	}
	return &GftProvider{
		registeredSource: make(map[string]Source),
		tmpDir:           tmpDir,
		process:          process,
	}
}

func (g *GftProvider) Start() error {
	return nil
}

func (g *GftProvider) Stop() {

}

func (g *GftProvider) newTmpFile() (*os.File, error) {
	return ioutil.TempFile(g.tmpDir, "tmpfile")
}

func (g *GftProvider) UploadStream(length int, stream []byte) Source {
	file, err := g.newTmpFile()
	if err != nil {
		return nil
	}
	// write the whole body at once
	err = ioutil.WriteFile(file.Name(), stream, os.ModePerm)
	if err != nil {
		panic(err)
	}
	return &GftpSource{}

}

func (g *GftProvider) UploadFile(filePath string) (Source, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	h := sha3.New256()
	if _, err := io.Copy(h, f); err != nil {
		log.Fatal(err)
	}
	digest := hex.EncodeToString(h.Sum(nil))
	if s, ok := g.registeredSource[digest]; ok {
		fmt.Printf("File %s already published, digest: %s", filePath, digest)
		return s, nil
	}
	fmt.Printf("Publishing file %s, digest: %s", filePath, digest)
	process
}
