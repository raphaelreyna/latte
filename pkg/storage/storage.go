package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
)

type Storage struct {
	providers map[string]Provider
}

func (s *Storage) RegisterStorageProvider(p Provider, scheme ...string) error {
	if len(scheme) == 0 {
		return errors.New("no schemes provided")
	}

	if s.providers == nil {
		s.providers = make(map[string]Provider)
	}

	dups := make(map[string]struct{})
	for _, schemeName := range scheme {
		if schemeName == "" {
			return errors.New("empty scheme provided")
		}
		if _, found := s.providers[schemeName]; found {
			return fmt.Errorf("scheme %s already registered", schemeName)
		}
		if _, found := dups[schemeName]; found {
			return fmt.Errorf("duplicate scheme %s provided", schemeName)
		}
		dups[schemeName] = struct{}{}
	}

	for _, schemeName := range scheme {
		s.providers[schemeName] = p
	}

	return nil
}

func (s *Storage) provider(u *url.URL) Provider {
	return s.providers[u.Scheme]
}

func (s *Storage) Delete(ctx context.Context, u *url.URL) error {
	var provider = s.provider(u)
	if provider == nil {
		return errors.New("invalid target url schema")
	}

	return provider.Delete(ctx, u)
}

func (s *Storage) WriteCloser(ctx context.Context, u *url.URL) (io.WriteCloser, error) {
	var provider = s.provider(u)
	if provider == nil {
		return nil, errors.New("invalid target url schema")
	}

	return provider.WriteCloser(ctx, u)
}

func (s *Storage) ReadCloser(ctx context.Context, u *url.URL) (io.ReadCloser, error) {
	var provider = s.provider(u)
	if provider == nil {
		return nil, errors.New("invalid target url schema")
	}

	return provider.ReadCloser(ctx, u)
}

func (s *Storage) ExtractPitchArchive(ctx context.Context, src *url.URL, dstPath string) (err error) {
	var provider = s.provider(src)
	if provider == nil {
		return errors.New("invalid target url schema")
	}

	return ExtractPitchArchive(ctx, provider, src, dstPath)
}

func (s *Storage) StoreFile(ctx context.Context, srcPath string, dst *url.URL) (err error) {
	var provider = s.provider(dst)
	if provider == nil {
		return errors.New("invalid target url schema")
	}

	return StoreFile(ctx, provider, srcPath, dst)
}

func (s *Storage) StoreBytes(ctx context.Context, src []byte, dst *url.URL) (err error) {
	var provider = s.provider(dst)
	if provider == nil {
		return errors.New("invalid target url schema")
	}

	return StoreBytes(ctx, provider, src, dst)
}
