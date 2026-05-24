package grpc

type fooConverter struct{}

func (fooConverter) ToResponse(foo *Foo) *FooResponse {
	return &FooResponse{
		ID:    foo.ID,
		Title: foo.Title,
	}
}
