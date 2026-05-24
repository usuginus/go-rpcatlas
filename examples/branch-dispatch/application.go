package transport

import "context"

type FooApplication interface {
	ProcessFoo(context.Context, ProcessFooCommand) (string, error)
}

type ProcessFooCommand struct {
	Mode    string
	Payload FooPayload
}

type FooPayload interface {
	fooPayload()
}

type AlphaPayload struct {
	Body string
}

func (AlphaPayload) fooPayload() {}

func (a AlphaPayload) Normalize() string {
	return a.Body
}

type BetaPayload struct {
	URL string
}

func (BetaPayload) fooPayload() {}

func (a BetaPayload) Normalize() string {
	return a.URL
}

type fooApplication struct {
	policy *fooPolicy
	store  *fooStore
	index  *previewClient
}

var _ FooApplication = (*fooApplication)(nil)

func (a *fooApplication) ProcessFoo(ctx context.Context, cmd ProcessFooCommand) (string, error) {
	switch payload := cmd.Payload.(type) {
	case AlphaPayload:
		payload.Normalize()
		if err := a.policy.ValidateAlpha(payload); err != nil {
			return "", err
		}
	case BetaPayload:
		payload.Normalize()
		if err := a.policy.ValidateBeta(payload); err != nil {
			return "", err
		}
	default:
		return "", a.policy.RejectUnsupportedPayload()
	}

	switch cmd.Mode {
	case "draft":
		return a.store.SaveDraft(ctx, cmd)
	case "publish":
		fooID, err := a.store.Publish(ctx, cmd)
		if err != nil {
			return "", err
		}
		if err := a.index.Index(ctx, fooID); err != nil {
			return "", err
		}
		return fooID, nil
	default:
		return "", a.policy.RejectUnsupportedMode(cmd.Mode)
	}
}
