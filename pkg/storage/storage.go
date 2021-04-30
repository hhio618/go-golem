package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"math"
)

const (
	BufferSize                = 40960
	DownloadBytesLimitDefault = 1 * 1024 * 1024
)

type Content struct {
	Length int
	Stream chan []byte
}

func ContentFrom(length int, r io.ReadCloser) *Content {
	stream := make(chan []byte)
	buf := make([]byte, BufferSize)
	for {
		n, err := r.Read(buf)
		if err == io.EOF {
			break
		}
		stream <- buf[:n]
	}
	return &Content{
		Length: length,
		Stream: stream,
	}
}

type Source interface {
	DownloadUrl() string
	ContentLength() int
}

type IDestination interface {
	UploadUrl() string
	DownloadStream() (*Content, error)
	DownloadFile(ctx context.Context, destPath string)
	DownloadBytes(ctx context.Context, limit int, resultFunc func(interface{}), errFunc func(error))
}

type Destination struct {
	Destination IDestination
}

func (d *Destination) DownloadBytes(ctx context.Context, limit int, resultFunc func([]byte), errFunc func(error)) {
	if limit == 0 {
		limit = DownloadBytesLimitDefault
	}
	output := make([]byte, 0)
	content, err := d.Destination.DownloadStream()
	if err != nil {
		errFunc(errors.New("downloading stream"))
		return
	}
	go func() {
		for {
			select {
			case <-ctx.Done():
				errFunc(errors.New("context canceled"))
				return
			case chunk := <-content.Stream:
				limitRemaining := limit - len(chunk)
				if limitRemaining > len(chunk) {
					output = append(output, chunk...)
				} else {
					output = append(output, chunk[:limitRemaining]...)
					resultFunc(output)
					return
				}

			}
		}
	}()
}

func (d *Destination) DownloadFile(ctx context.Context, destPath string) {
	d.DownloadBytes(ctx, math.MaxInt64, func(b []byte) {
		err := ioutil.WriteFile(destPath, b, fs.ModePerm)
		if err != nil {
			fmt.Printf("err: %v", err)
		}
	}, func(e error) {
		if e != nil {
			fmt.Printf("err: %v", e)
		}
	})
}

type InputStorageProvider interface {
	UploadStream(length int, stream []byte) (Source, error)
	UploadBytes(data []byte) (Source, error)
	UploadFile(filePath string) (Source, error)
}

type InputStorage struct {
	InputStorageProvider
}

func (i *InputStorage) UploadBytes(data []byte) (Source, error) {
	return i.InputStorageProvider.UploadStream(len(data), data)
}

func (i *InputStorage) UploadFile(filePath string) (Source, error) {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	return i.InputStorageProvider.UploadStream(len(data), data)

}

type OutputStorageProvider interface {
	NewDestination(destFile string) IDestination
}

type StorageProvider interface {
	InputStorageProvider
	OutputStorageProvider
}

type ComposedStorageProvider struct {
	StorageProvider StorageProvider
}

func NewComposedStorageProvider(InputStorageProvider InputStorageProvider,
	OutputStorageProvider OutputStorageProvider) *ComposedStorageProvider {
	return &ComposedStorageProvider{
		StorageProvider: StorageProvider{
			InputStorageProvider:  InputStorageProvider,
			OutputStorageProvider: OutputStorageProvider,
		},
	}
}
