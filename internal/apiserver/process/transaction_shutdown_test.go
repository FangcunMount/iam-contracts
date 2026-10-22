package process

import (
	"context"
	"errors"
	"github.com/FangcunMount/component-base/pkg/processruntime"
	config "github.com/FangcunMount/iam/v5/internal/apiserver/config"
	options "github.com/FangcunMount/iam/v5/internal/apiserver/options"
	grpcpkg "github.com/FangcunMount/iam/v5/internal/pkg/grpc"
	"github.com/FangcunMount/reliable-messaging/transaction"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/test/bufconn"
	"net"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestTransactionHostDrainsTransportBeforeCheckerAndDatabase(t *testing.T) {
	var order []string
	deps := shutdownSequenceDeps{transactionMode: true,
		beginDrain:     func() { order = append(order, "admission") },
		closeHTTP:      func() error { order = append(order, "HTTP"); return nil },
		closeGRPC:      func() error { order = append(order, "gRPC"); return nil },
		stopPolicySync: func(context.Context) error { order = append(order, "subscriber"); return nil },
		stopReliable:   func(context.Context) error { order = append(order, "Checker"); return nil },
		closeDatabase:  func() error { order = append(order, "MySQL"); return nil },
	}
	if err := runShutdownSequence(deps); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(order, []string{"admission", "HTTP", "gRPC", "subscriber", "Checker", "MySQL"}) {
		t.Fatal(order)
	}
}
func TestTransactionHostTimeoutDoesNotReleaseRealInFlightDependencies(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})
	var gate, transport, checker, database atomic.Int32
	closer := newTransactionShutdown(shutdownSequenceDeps{transactionMode: true,
		beginDrain:    func() { gate.Add(1) },
		closeGRPC:     func() error { transport.Add(1); close(entered); <-release; return nil },
		stopReliable:  func(context.Context) error { checker.Add(1); return nil },
		closeDatabase: func() error { database.Add(1); return nil },
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for i := 0; i < 8; i++ {
		var pending *transaction.ShutdownIncomplete
		err := closer.Close(ctx)
		if !errors.As(err, &pending) || !pending.RuntimePending || !errors.Is(err, context.Canceled) {
			t.Fatal("false completion", err)
		}
	}
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("transport drain absent")
	}
	if gate.Load() != 1 || transport.Load() != 1 || checker.Load() != 0 || database.Load() != 0 {
		t.Fatal("lost real resource ownership")
	}
	close(release)
	wait, cancelWait := context.WithTimeout(context.Background(), time.Second)
	defer cancelWait()
	if err := closer.Wait(wait); err != nil {
		t.Fatal(err)
	}
	if checker.Load() != 1 || database.Load() != 1 {
		t.Fatal("continued drain did not complete exactly once")
	}
}
func TestTransactionHostFailedDrainRetainsCheckerAndMySQL(t *testing.T) {
	want := errors.New("transport could not drain")
	released := false
	err := runShutdownSequence(shutdownSequenceDeps{transactionMode: true,
		closeGRPC:     func() error { return want },
		stopReliable:  func(context.Context) error { released = true; return nil },
		closeDatabase: func() error { released = true; return nil },
	})
	if !errors.Is(err, want) || released {
		t.Fatal("failed drain closed dependent resources", err)
	}
}

type heldHealth struct {
	healthpb.UnimplementedHealthServer
	entered, release chan struct{}
}

func (h *heldHealth) Check(context.Context, *healthpb.HealthCheckRequest) (*healthpb.HealthCheckResponse, error) {
	close(h.entered)
	<-h.release
	return &healthpb.HealthCheckResponse{Status: healthpb.HealthCheckResponse_SERVING}, nil
}
func TestActualTransactionGRPCDrainDoesNotForceStopOriginalHandlerAtFiveSeconds(t *testing.T) {
	listener := bufconn.Listen(1 << 20)
	transport := grpc.NewServer()
	service := &heldHealth{entered: make(chan struct{}), release: make(chan struct{})}
	healthpb.RegisterHealthServer(transport, service)
	var release sync.Once
	t.Cleanup(func() { release.Do(func() { close(service.release) }); transport.Stop(); _ = listener.Close() })
	go func() { _ = transport.Serve(listener) }()
	client, err := grpc.NewClient("passthrough:///owned-bufnet", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) { return listener.DialContext(ctx) }))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = client.Close() }()
	rpc := make(chan error, 1)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	go func() {
		_, err := healthpb.NewHealthClient(client).Check(ctx, &healthpb.HealthCheckRequest{})
		rpc <- err
	}()
	select {
	case <-service.entered:
	case <-ctx.Done():
		t.Fatal("owned RPC absent")
	}
	opts := options.NewOptions()
	opts.Messaging.Transactions = map[string]options.TransactionOptions{"iam.mysql": {}}
	host := &apiServer{cfg: &config.Config{Options: opts}, grpcServer: &grpcpkg.Server{Server: transport}}
	deps := host.buildShutdownSequenceDeps(processruntime.Lifecycle{})
	var closed atomic.Int32
	deps.closeDatabase = func() error { closed.Add(1); return nil }
	closer := newTransactionShutdown(deps)
	wait, cancelWait := context.WithCancel(context.Background())
	cancelWait()
	if err := closer.Close(wait); err == nil {
		t.Fatal("busy host reported complete")
	}
	select {
	case err := <-rpc:
		t.Fatal("accepted RPC was forced to stop", err)
	case <-time.After(5100 * time.Millisecond):
	}
	if closed.Load() != 0 {
		t.Fatal("database released while handler still running")
	}
	release.Do(func() { close(service.release) })
	done, cancelDone := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancelDone()
	if err := closer.Wait(done); err != nil {
		t.Fatal(err)
	}
	if err := <-rpc; err != nil {
		t.Fatal(err)
	}
	if closed.Load() != 1 {
		t.Fatal("real drain did not release database once")
	}
}
