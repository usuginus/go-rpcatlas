package transport

import "context"

type FooKind string

const (
	KindAlpha FooKind = "alpha"
	KindBeta  FooKind = "beta"
)

type ProcessFooCommand struct {
	Kind FooKind
	Body string
}

type FooApplication interface {
	ProcessFoo(context.Context, ProcessFooCommand) (string, error)
}

type FooProcessor interface {
	Process(context.Context, ProcessFooCommand) (string, error)
}

type fooApplication struct {
	policy     *fooPolicy
	processors map[FooKind]FooProcessor
}

var _ FooApplication = (*fooApplication)(nil)

func NewFooApplication(store *fooStore, preview *previewClient) FooApplication {
	return &fooApplication{
		policy: &fooPolicy{},
		processors: map[FooKind]FooProcessor{
			KindAlpha: newAlphaProcessor(store),
			KindBeta:  newBetaProcessor(preview),
		},
	}
}

func (a *fooApplication) ProcessFoo(ctx context.Context, cmd ProcessFooCommand) (string, error) {
	processor, ok := a.processors[cmd.Kind]
	if !ok {
		return "", a.policy.RejectUnsupportedKind(cmd.Kind)
	}
	return processor.Process(ctx, cmd)
}

type alphaProcessor struct {
	policy *fooPolicy
	store  *fooStore
}

func newAlphaProcessor(store *fooStore) FooProcessor {
	return &alphaProcessor{
		policy: &fooPolicy{},
		store:  store,
	}
}

func (p *alphaProcessor) Process(ctx context.Context, cmd ProcessFooCommand) (string, error) {
	if err := p.policy.ValidateAlpha(cmd); err != nil {
		return "", err
	}
	return p.store.SaveAlpha(ctx, cmd)
}

type betaProcessor struct {
	policy  *fooPolicy
	preview *previewClient
}

func newBetaProcessor(preview *previewClient) FooProcessor {
	return &betaProcessor{
		policy:  &fooPolicy{},
		preview: preview,
	}
}

func (p *betaProcessor) Process(ctx context.Context, cmd ProcessFooCommand) (string, error) {
	if err := p.policy.ValidateBeta(cmd); err != nil {
		return "", err
	}
	return p.preview.RenderBeta(ctx, cmd)
}
