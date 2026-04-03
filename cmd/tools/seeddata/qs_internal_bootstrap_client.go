package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"time"

	qsbootstrappb "github.com/FangcunMount/iam-contracts/cmd/tools/seeddata/internal/qsbootstrappb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

type qsBootstrapOperatorRequest struct {
	OrgID    int64
	UserID   int64
	Name     string
	Email    string
	Phone    string
	IsActive bool
}

type qsBootstrapOperatorResponse struct {
	OperatorID uint64
	Created    bool
	Message    string
	Roles      []string
}

func callQSBootstrapOperator(
	ctx context.Context,
	cfg QSInternalGRPCConfig,
	req qsBootstrapOperatorRequest,
) (*qsBootstrapOperatorResponse, error) {
	dialCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	transportCreds, err := buildQSInternalGRPCCredentials(cfg)
	if err != nil {
		return nil, err
	}

	conn, err := grpc.DialContext(
		dialCtx,
		cfg.Address,
		grpc.WithTransportCredentials(transportCreds),
	)
	if err != nil {
		return nil, fmt.Errorf("dial qs internal grpc %s: %w", cfg.Address, err)
	}
	defer conn.Close()

	client := qsbootstrappb.NewInternalServiceClient(conn)
	resp, err := client.BootstrapOperator(dialCtx, &qsbootstrappb.BootstrapOperatorRequest{
		OrgId:    req.OrgID,
		UserId:   req.UserID,
		Name:     req.Name,
		Email:    req.Email,
		Phone:    req.Phone,
		IsActive: req.IsActive,
	})
	if err != nil {
		return nil, fmt.Errorf("bootstrap operator grpc call failed: %w", err)
	}

	return &qsBootstrapOperatorResponse{
		OperatorID: resp.OperatorId,
		Created:    resp.Created,
		Message:    resp.Message,
		Roles:      append([]string(nil), resp.Roles...),
	}, nil
}

func buildQSInternalGRPCCredentials(cfg QSInternalGRPCConfig) (credentials.TransportCredentials, error) {
	if cfg.Insecure {
		return insecure.NewCredentials(), nil
	}

	tlsCfg := &tls.Config{
		MinVersion: tls.VersionTLS12,
		ServerName: cfg.ServerName,
	}

	if cfg.CAFile != "" {
		caPEM, err := os.ReadFile(cfg.CAFile)
		if err != nil {
			return nil, fmt.Errorf("read qs_internal_grpc.ca_file: %w", err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(caPEM) {
			return nil, fmt.Errorf("append qs internal grpc ca failed")
		}
		tlsCfg.RootCAs = pool
	}

	if cfg.CertFile != "" || cfg.KeyFile != "" {
		if cfg.CertFile == "" || cfg.KeyFile == "" {
			return nil, fmt.Errorf("qs_internal_grpc.cert_file and key_file must both be set for mTLS")
		}
		cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("load qs internal grpc client certificate: %w", err)
		}
		tlsCfg.Certificates = []tls.Certificate{cert}
	}

	return credentials.NewTLS(tlsCfg), nil
}
