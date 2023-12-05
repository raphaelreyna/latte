package test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
)

type MockWriteCloser struct {
	bytes.Buffer
}

func (m *MockWriteCloser) Close() error {
	return nil
}

type MockProvider struct {
	Data map[string]*MockWriteCloser
}

func (m *MockProvider) Delete(ctx context.Context, u *url.URL) error {
	delete(m.Data, u.String())
	return nil
}

func (m *MockProvider) WriteCloser(ctx context.Context, u *url.URL) (w io.WriteCloser, err error) {
	wc := new(MockWriteCloser)
	m.Data[u.String()] = wc
	return wc, nil
}

func (m *MockProvider) ReadCloser(ctx context.Context, u *url.URL) (r io.ReadCloser, err error) {
	wc, found := m.Data[u.String()]
	if !found {
		return nil, fmt.Errorf("unable to find %s", u.String())
	}

	return io.NopCloser(&wc.Buffer), nil
}
