// Package interceptors 提供 mTLS 认证拦截器
package interceptors

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// MTLSInterceptor mTLS 认证拦截器
// 从 TLS 连接中提取客户端证书信息，验证服务身份
func MTLSInterceptor(opts ...MTLSOption) grpc.UnaryServerInterceptor {
	options := defaultMTLSOptions()
	for _, opt := range opts {
		opt(options)
	}

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// 检查是否跳过该方法
		if options.skipMatcher.Match(info.FullMethod) {
			return handler(ctx, req)
		}

		// 提取服务身份
		identity, err := ExtractServiceIdentityFromTLS(ctx)
		if err != nil {
			if options.logger != nil {
				options.logger.LogError("mTLS authentication failed",
					map[string]interface{}{
						"method": info.FullMethod,
						"error":  err.Error(),
					})
			}
			return nil, status.Error(codes.Unauthenticated, "client certificate required")
		}

		// 验证服务身份
		if options.validator != nil {
			if err := options.validator.Validate(identity); err != nil {
				if options.logger != nil {
					options.logger.LogError("mTLS service validation failed",
						map[string]interface{}{
							"method":  info.FullMethod,
							"service": identity.ServiceName,
							"error":   err.Error(),
						})
				}
				return nil, status.Error(codes.Unauthenticated, "service not authorized")
			}
		}

		// 将服务身份注入上下文
		ctx = ContextWithServiceIdentity(ctx, identity)

		if options.logger != nil {
			options.logger.LogInfo("mTLS authentication succeeded",
				map[string]interface{}{
					"method":    info.FullMethod,
					"service":   identity.ServiceName,
					"namespace": identity.ServiceNamespace,
				})
		}

		return handler(ctx, req)
	}
}

// MTLSStreamInterceptor mTLS 流式认证拦截器
func MTLSStreamInterceptor(opts ...MTLSOption) grpc.StreamServerInterceptor {
	options := defaultMTLSOptions()
	for _, opt := range opts {
		opt(options)
	}

	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if options.skipMatcher.Match(info.FullMethod) {
			return handler(srv, ss)
		}

		ctx := ss.Context()
		identity, err := ExtractServiceIdentityFromTLS(ctx)
		if err != nil {
			return status.Error(codes.Unauthenticated, "client certificate required")
		}

		if options.validator != nil {
			if err := options.validator.Validate(identity); err != nil {
				return status.Error(codes.Unauthenticated, "service not authorized")
			}
		}

		ctx = ContextWithServiceIdentity(ctx, identity)
		wrappedStream := &WrappedServerStream{
			ServerStream: ss,
			Ctx:          ctx,
		}

		return handler(srv, wrappedStream)
	}
}

// ExtractServiceIdentityFromTLS 从 TLS 连接提取服务身份
func ExtractServiceIdentityFromTLS(ctx context.Context) (*ServiceIdentity, error) {
	p, ok := peer.FromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("failed to get peer from context")
	}

	tlsInfo, ok := p.AuthInfo.(credentials.TLSInfo)
	if !ok {
		return nil, fmt.Errorf("peer auth info is not TLS")
	}

	if len(tlsInfo.State.PeerCertificates) == 0 {
		return nil, fmt.Errorf("no client certificate found")
	}

	cert := tlsInfo.State.PeerCertificates[0]
	return ParseCertificateIdentity(cert)
}

// ParseCertificateIdentity 从证书解析服务身份
func ParseCertificateIdentity(cert *x509.Certificate) (*ServiceIdentity, error) {
	identity := &ServiceIdentity{
		CommonName:          cert.Subject.CommonName,
		OrganizationalUnits: cert.Subject.OrganizationalUnit,
		CertSerialNumber:    cert.SerialNumber.String(),
		DNSSANs:             cert.DNSNames,
		NotBefore:           cert.NotBefore,
		NotAfter:            cert.NotAfter,
	}

	// 从 URI SANs 提取服务信息（spiffe:// 格式）
	for _, uri := range cert.URIs {
		identity.URISANs = append(identity.URISANs, uri.String())

		// 解析 SPIFFE ID: spiffe://trust-domain/ns/namespace/sa/service
		if uri.Scheme == "spiffe" {
			parts := strings.Split(uri.Path, "/")
			// /ns/{namespace}/sa/{service}
			for i := 0; i < len(parts)-1; i++ {
				if parts[i] == "ns" && i+1 < len(parts) {
					identity.ServiceNamespace = parts[i+1]
				}
				if parts[i] == "sa" && i+1 < len(parts) {
					identity.ServiceName = parts[i+1]
				}
			}
		}
	}

	// 如果没有从 URI 解析出服务名，使用 CN
	if identity.ServiceName == "" {
		identity.ServiceName = extractServiceNameFromCN(cert.Subject.CommonName)
	}

	// 从 OU 解析命名空间
	if identity.ServiceNamespace == "" {
		identity.ServiceNamespace = extractNamespaceFromOU(cert.Subject.OrganizationalUnit)
	}

	return identity, nil
}

// extractServiceNameFromCN 从 CN 提取服务名
// 支持格式：service-name, service-name.namespace.svc.cluster.local
func extractServiceNameFromCN(cn string) string {
	if cn == "" {
		return ""
	}

	// 处理 Kubernetes 风格的 CN
	if strings.Contains(cn, ".svc.") {
		parts := strings.Split(cn, ".")
		if len(parts) > 0 {
			return parts[0]
		}
	}

	return cn
}

// extractNamespaceFromOU 从 OU 提取命名空间
func extractNamespaceFromOU(ous []string) string {
	for _, ou := range ous {
		lower := strings.ToLower(ou)
		if strings.HasPrefix(lower, "ns:") || strings.HasPrefix(lower, "namespace:") {
			parts := strings.SplitN(ou, ":", 2)
			if len(parts) == 2 {
				return parts[1]
			}
		}
		// 常见的环境命名空间
		if lower == "production" || lower == "staging" || lower == "development" ||
			lower == "prod" || lower == "dev" || lower == "test" {
			return lower
		}
	}
	return ""
}

// ===== 选项定义 =====

// mtlsOptions mTLS 拦截器选项
type mtlsOptions struct {
	skipMatcher *SkipMethodMatcher
	validator   IdentityValidator
	logger      InterceptorLogger
}

func defaultMTLSOptions() *mtlsOptions {
	return &mtlsOptions{
		skipMatcher: NewSkipMethodMatcher(DefaultSkipMethods()...),
	}
}

// MTLSOption mTLS 拦截器选项函数
type MTLSOption func(*mtlsOptions)

// WithMTLSSkipMethods 设置跳过认证的方法列表
func WithMTLSSkipMethods(methods ...string) MTLSOption {
	return func(o *mtlsOptions) {
		o.skipMatcher.Add(methods...)
	}
}

// WithMTLSValidator 设置服务身份验证器
func WithMTLSValidator(validator IdentityValidator) MTLSOption {
	return func(o *mtlsOptions) {
		o.validator = validator
	}
}

// WithMTLSLogger 设置日志记录器
func WithMTLSLogger(logger InterceptorLogger) MTLSOption {
	return func(o *mtlsOptions) {
		o.logger = logger
	}
}

// WithAllowedServices 设置允许的服务列表
func WithAllowedServices(services ...string) MTLSOption {
	return func(o *mtlsOptions) {
		o.validator = &allowedServicesValidator{services: services}
	}
}

// allowedServicesValidator 允许服务列表验证器
type allowedServicesValidator struct {
	services []string
}

func (v *allowedServicesValidator) Validate(identity *ServiceIdentity) error {
	for _, s := range v.services {
		if identity.ServiceName == s {
			return nil
		}
	}
	return fmt.Errorf("service %q not in allowed list", identity.ServiceName)
}

// ===== 日志接口 =====

// InterceptorLogger 拦截器日志接口
type InterceptorLogger interface {
	LogInfo(msg string, fields map[string]interface{})
	LogError(msg string, fields map[string]interface{})
}

// ===== TLS 证书验证辅助 =====

// TLSClientConfig 创建客户端 TLS 配置
func TLSClientConfig(caCert, clientCert, clientKey []byte, serverName string) (*tls.Config, error) {
	// 加载 CA 证书
	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to append CA certificate")
	}

	// 加载客户端证书
	cert, err := tls.X509KeyPair(clientCert, clientKey)
	if err != nil {
		return nil, fmt.Errorf("failed to load client certificate: %w", err)
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caCertPool,
		ServerName:   serverName,
		MinVersion:   tls.VersionTLS12,
	}, nil
}

// TLSServerConfig 创建服务端 TLS 配置
func TLSServerConfig(caCert, serverCert, serverKey []byte, clientAuth tls.ClientAuthType) (*tls.Config, error) {
	// 加载 CA 证书（用于验证客户端）
	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to append CA certificate")
	}

	// 加载服务端证书
	cert, err := tls.X509KeyPair(serverCert, serverKey)
	if err != nil {
		return nil, fmt.Errorf("failed to load server certificate: %w", err)
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientCAs:    caCertPool,
		ClientAuth:   clientAuth,
		MinVersion:   tls.VersionTLS12,
	}, nil
}
