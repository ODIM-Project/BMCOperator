package controllers

import (
	"context"

	// infraiov1 "github.com/ODIM-Project/bmc-operator/api/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type mockClient struct {
	objType string
}

// Client mocks
func (m mockClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object) error {
	return nil
}

func (m mockClient) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	return nil
}

func (m mockClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	return nil
}

func (m mockClient) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	return nil
}

func (m mockClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	return nil
}

func (m mockClient) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	return nil
}

func (m mockClient) DeleteAllOf(ctx context.Context, obj client.Object, opts ...client.DeleteAllOfOption) error {
	return nil
}

func (m mockClient) Status() client.StatusWriter {
	return mockClient{}
}

func (m mockClient) RESTMapper() meta.RESTMapper {
	return nil
}

func (m mockClient) Scheme() *runtime.Scheme {
	return nil
}