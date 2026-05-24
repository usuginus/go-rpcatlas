package transport

import "errors"

type fooPolicy struct{}

func (p *fooPolicy) ValidateAlpha(payload AlphaPayload) error {
	if payload.Body == "" {
		return errors.New("body is required")
	}
	return nil
}

func (p *fooPolicy) ValidateBeta(payload BetaPayload) error {
	if payload.URL == "" {
		return errors.New("URL is required")
	}
	return nil
}

func (p *fooPolicy) RejectUnsupportedPayload() error {
	return errors.New("unsupported payload")
}

func (p *fooPolicy) RejectUnsupportedMode(mode string) error {
	return errors.New("unsupported mode")
}
