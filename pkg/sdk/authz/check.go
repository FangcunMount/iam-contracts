package authz

import (
	"context"

	authzv4 "github.com/FangcunMount/iam/v5/api/grpc/iam/authz/v4"
	"github.com/FangcunMount/iam/v5/pkg/sdk/errors"
)

func (c *Client) Check(ctx context.Context, req *authzv4.CheckRequest) (*authzv4.CheckResponse, error) {
	resp, err := c.authorizationService.Check(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return resp, nil
}

// Allow performs an unconditional resource/action check. Conditional grants
// fail closed because no ObjectContext attributes are supplied.
func (c *Client) Allow(ctx context.Context, subject, resource, action string) (bool, error) {
	resp, err := c.Check(ctx, &authzv4.CheckRequest{
		Subject: subject, Resource: resource, Action: action,
	})
	if err != nil {
		return false, err
	}
	return resp.Allowed, nil
}

func (c *Client) CheckObject(
	ctx context.Context,
	subject, resource, action, objectID string,
	attributes []*authzv4.ObjectAttribute,
) (*authzv4.CheckResponse, error) {
	return c.Check(ctx, &authzv4.CheckRequest{
		Subject: subject, Resource: resource, Action: action,
		ObjectContext: &authzv4.ObjectContext{ObjectId: objectID, Attributes: attributes},
	})
}

func (c *Client) GetAuthorizationSnapshot(ctx context.Context, req *authzv4.GetAuthorizationSnapshotRequest) (*authzv4.GetAuthorizationSnapshotResponse, error) {
	resp, err := c.authorizationService.GetAuthorizationSnapshot(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return resp, nil
}
