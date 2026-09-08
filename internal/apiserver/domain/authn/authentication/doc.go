// Package authentication 负责登录流程中的身份核验：根据凭据选择策略，
// 核验请求者是否控制某个登录身份，并输出 AuthDecision 与 Principal。
// 登录准入、会话建立和令牌颁发由后续环节完成。
package authentication
