// Package mtls 提供 mTLS 客户端身份信息提取和上下文管理
package mtls

import (
	"context"
	"crypto/x509"
	"fmt"
	"strings"

	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
)

// ServiceIdentity 服务身份信息（从 mTLS 证书中提取）
type ServiceIdentity struct {
	// 证书基本信息
	CommonName          string   `json:"common_name"`          // 证书 CN
	OrganizationalUnits []string `json:"organizational_units"` // 证书 OU 列表
	Organization        []string `json:"organization"`         // 证书 O 列表
	DNSNames            []string `json:"dns_names"`            // SAN DNS 名称
	IPAddresses         []string `json:"ip_addresses"`         // SAN IP 地址

	// 解析后的服务标识
	ServiceName      string `json:"service_name"`      // 服务名称（从 CN 或 SAN 提取）
	ServiceNamespace string `json:"service_namespace"` // 服务命名空间
	Environment      string `json:"environment"`       // 环境标识 (dev/staging/prod)

	// 证书元数据
	SerialNumber     string `json:"serial_number"`      // 证书序列号
	NotBefore        string `json:"not_before"`         // 证书生效时间
	NotAfter         string `json:"not_after"`          // 证书过期时间
	IssuerCommonName string `json:"issuer_common_name"` // 颁发者 CN
}

// 上下文键类型
type contextKey string

const (
	// ServiceIdentityKey 服务身份上下文键
	ServiceIdentityKey contextKey = "grpc_service_identity"
	// PeerCertificateKey 对端证书上下文键
	PeerCertificateKey contextKey = "grpc_peer_certificate"
)

// ExtractServiceIdentity 从 gRPC 上下文中提取服务身份
func ExtractServiceIdentity(ctx context.Context) (*ServiceIdentity, error) {
	// 先检查是否已经在上下文中
	if identity, ok := ctx.Value(ServiceIdentityKey).(*ServiceIdentity); ok {
		return identity, nil
	}

	// 从 peer 信息中提取
	p, ok := peer.FromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("failed to get peer from context")
	}

	tlsInfo, ok := p.AuthInfo.(credentials.TLSInfo)
	if !ok {
		return nil, fmt.Errorf("peer auth info is not TLS")
	}

	if len(tlsInfo.State.VerifiedChains) == 0 || len(tlsInfo.State.VerifiedChains[0]) == 0 {
		return nil, fmt.Errorf("no verified certificate chains")
	}

	// 获取客户端证书（链的第一个证书）
	clientCert := tlsInfo.State.VerifiedChains[0][0]
	return ParseCertificateIdentity(clientCert), nil
}

// ParseCertificateIdentity 从 X.509 证书中解析服务身份
func ParseCertificateIdentity(cert *x509.Certificate) *ServiceIdentity {
	identity := &ServiceIdentity{
		CommonName:          cert.Subject.CommonName,
		OrganizationalUnits: cert.Subject.OrganizationalUnit,
		Organization:        cert.Subject.Organization,
		DNSNames:            cert.DNSNames,
		SerialNumber:        cert.SerialNumber.String(),
		NotBefore:           cert.NotBefore.Format("2006-01-02T15:04:05Z"),
		NotAfter:            cert.NotAfter.Format("2006-01-02T15:04:05Z"),
		IssuerCommonName:    cert.Issuer.CommonName,
	}

	// 提取 IP 地址
	for _, ip := range cert.IPAddresses {
		identity.IPAddresses = append(identity.IPAddresses, ip.String())
	}

	// 解析服务名称和命名空间
	// 约定格式: <service>.<namespace>.svc 或直接 <service>.svc
	identity.ServiceName, identity.ServiceNamespace = ParseServiceFromCN(cert.Subject.CommonName)

	// 如果 CN 没有解析出来，尝试从 SAN 解析
	if identity.ServiceName == "" && len(cert.DNSNames) > 0 {
		identity.ServiceName, identity.ServiceNamespace = ParseServiceFromDNS(cert.DNSNames[0])
	}

	// 从 OU 提取环境信息
	for _, ou := range cert.Subject.OrganizationalUnit {
		if IsEnvironment(ou) {
			identity.Environment = ou
			break
		}
	}

	return identity
}

// ParseServiceFromCN 从 CN 解析服务名称
func ParseServiceFromCN(cn string) (serviceName, namespace string) {
	// 支持格式:
	// - qs.svc
	// - qs.internal.svc
	// - qs-service
	parts := strings.Split(cn, ".")
	if len(parts) >= 2 {
		if parts[len(parts)-1] == "svc" {
			if len(parts) >= 3 {
				return parts[0], parts[1]
			}
			return parts[0], "default"
		}
	}

	// 如果没有 .svc 后缀，直接使用 CN 作为服务名
	return cn, "default"
}

// ParseServiceFromDNS 从 DNS SAN 解析服务名称
func ParseServiceFromDNS(dns string) (serviceName, namespace string) {
	// 支持格式:
	// - qs.internal.example.com
	// - qs-grpc.svc.cluster.local
	parts := strings.Split(dns, ".")
	if len(parts) >= 1 {
		return parts[0], "default"
	}
	return dns, "default"
}

// IsEnvironment 判断是否为环境标识
func IsEnvironment(s string) bool {
	environments := []string{"dev", "development", "staging", "stage", "prod", "production", "test", "qa"}
	lower := strings.ToLower(s)
	for _, env := range environments {
		if lower == env {
			return true
		}
	}
	return false
}

// ContextWithServiceIdentity 将服务身份注入上下文
func ContextWithServiceIdentity(ctx context.Context, identity *ServiceIdentity) context.Context {
	return context.WithValue(ctx, ServiceIdentityKey, identity)
}

// ServiceIdentityFromContext 从上下文获取服务身份
func ServiceIdentityFromContext(ctx context.Context) (*ServiceIdentity, bool) {
	identity, ok := ctx.Value(ServiceIdentityKey).(*ServiceIdentity)
	return identity, ok
}

// GetServiceName 快速获取服务名称
func GetServiceName(ctx context.Context) string {
	if identity, ok := ServiceIdentityFromContext(ctx); ok {
		return identity.ServiceName
	}
	return ""
}

// GetServiceNamespace 快速获取服务命名空间
func GetServiceNamespace(ctx context.Context) string {
	if identity, ok := ServiceIdentityFromContext(ctx); ok {
		return identity.ServiceNamespace
	}
	return ""
}

// String 返回服务身份的字符串表示
func (s *ServiceIdentity) String() string {
	if s.ServiceNamespace != "" && s.ServiceNamespace != "default" {
		return fmt.Sprintf("%s.%s", s.ServiceName, s.ServiceNamespace)
	}
	return s.ServiceName
}

// FullIdentifier 返回完整的服务标识符
func (s *ServiceIdentity) FullIdentifier() string {
	parts := []string{s.ServiceName}
	if s.ServiceNamespace != "" {
		parts = append(parts, s.ServiceNamespace)
	}
	if s.Environment != "" {
		parts = append(parts, s.Environment)
	}
	return strings.Join(parts, "/")
}
