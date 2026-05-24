package transport

import "fmt"

type fooPolicy struct{}

func (p *fooPolicy) RejectUnsupportedKind(kind FooKind) error {
	return fmt.Errorf("unsupported foo kind: %s", kind)
}

func (p *fooPolicy) ValidateAlpha(cmd ProcessFooCommand) error {
	if cmd.Body == "" {
		return fmt.Errorf("alpha body is required")
	}
	return nil
}

func (p *fooPolicy) ValidateBeta(cmd ProcessFooCommand) error {
	if cmd.Body == "" {
		return fmt.Errorf("beta body is required")
	}
	return nil
}
