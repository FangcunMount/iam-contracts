// Package signin 编排凭据登录
//
// 登录流程统一为：身份核验 → 登录准入 → 会话建立 → 令牌颁发。
//  1. 身份核验：MethodRegistry 选择登录方式，ProofFactory 构造凭据，
//     Authenticator 返回 AuthDecision，成功时确认 Principal；CredentialRecorder 记录凭据副作用。
//  2. 登录准入：AdmissionPolicy 判断主体是否允许建立登录态。
//  3. 会话建立：SessionCreator 创建并保存 Session。
//  4. 令牌颁发：InitialTokenIssuer 签发令牌并保存初始 RefreshToken。
//
// 四个环节全部完成才表示登录成功；会话建立后的失败由 SignIn 补偿。
//
// 请求上下文规范：
// TenantID、RemoteIP、UserAgent 必须由 transport / compatibility 层写入
// LoginRequest 顶层字段。method.Payload 只保存具体登录方式自己的字段，
// 不承载公共请求上下文。
//
// 新增登录方式时必须同步扩展：
// method.AuthMethod、method.LoginMethod、method.Payload、proof.Builder、
// proof.Factory/assembler 注册、领域 authentication 策略，以及公开协议需要的
// compatibility wire payload 解析。漏掉 proof.Builder 注册会在运行时得到
// ErrProofBuildFailed / unsupported credential kind。
package signin
