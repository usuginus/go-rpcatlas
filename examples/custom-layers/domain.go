package transport

import "errors"

type fooPolicy struct{}

func (p *fooPolicy) Validate(cmd ProcessFooCommand) error {
	if cmd.Title == "" {
		return errors.New("title is required")
	}
	if cmd.Body == "" {
		return errors.New("body is required")
	}
	return nil
}
