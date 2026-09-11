package authz

import (
	"context"
	"fmt"

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

// Allow performs a resource/action authorization check.
func (c *Client) Allow(ctx context.Context, subject, resource, action string) (bool, error) {
	resp, err := c.Check(ctx, &authzv4.CheckRequest{
		Subject: subject, Resource: resource, Action: action,
	})
	if err != nil {
		return false, err
	}
	return resp.Allowed, nil
}

// Deprecated: conditional authorization has been retired. Use Allow or Check.
func (c *Client) CheckObject(
	ctx context.Context,
	subject, resource, action, objectID string,
	attributes []*authzv4.ObjectAttribute,
) (*authzv4.CheckResponse, error) {
	return nil, fmt.Errorf("conditional authorization has been retired")
}

func (c *Client) GetAuthorizationSnapshot(ctx context.Context, req *authzv4.GetAuthorizationSnapshotRequest) (*authzv4.GetAuthorizationSnapshotResponse, error) {
	resp, err := c.authorizationService.GetAuthorizationSnapshot(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return resp, nil
}
