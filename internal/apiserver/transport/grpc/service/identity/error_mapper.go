package identity

import (
	iamgrpc "github.com/FangcunMount/iam/v5/internal/pkg/grpc"
)

func toGRPCError(err error) error {
	return iamgrpc.ToStatusError(err)
}
