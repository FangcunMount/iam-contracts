package resource

// Action 动作值对象（用于运行时校验）
type Action struct {
	Name        string // 动作名称，如 read_all, read_own
	DisplayName string // 显示名称
	Scope       Scope  // all/own，用于区分是否需要 owner 校验
}

// Scope 动作作用域
type Scope string

const (
	ScopeAll Scope = "all" // 全局作用域
	ScopeOwn Scope = "own" // 仅自己
)

// 预定义动作枚举（V1 最小集）
var (
	ActionCreate     = Action{Name: "create", DisplayName: "创建", Scope: ScopeOwn}
	ActionReadAll    = Action{Name: "read_all", DisplayName: "读取(全部)", Scope: ScopeAll}
	ActionReadOwn    = Action{Name: "read_own", DisplayName: "读取(自己)", Scope: ScopeOwn}
	ActionUpdateAll  = Action{Name: "update_all", DisplayName: "更新(全部)", Scope: ScopeAll}
	ActionUpdateOwn  = Action{Name: "update_own", DisplayName: "更新(自己)", Scope: ScopeOwn}
	ActionDeleteAll  = Action{Name: "delete_all", DisplayName: "删除(全部)", Scope: ScopeAll}
	ActionDeleteOwn  = Action{Name: "delete_own", DisplayName: "删除(自己)", Scope: ScopeOwn}
	ActionApprove    = Action{Name: "approve", DisplayName: "审批", Scope: ScopeAll}
	ActionExport     = Action{Name: "export", DisplayName: "导出", Scope: ScopeAll}
	ActionDisableAll = Action{Name: "disable_all", DisplayName: "禁用(全部)", Scope: ScopeAll}
)

// StandardActions 标准动作集合
var StandardActions = []Action{
	ActionCreate,
	ActionReadAll,
	ActionReadOwn,
	ActionUpdateAll,
	ActionUpdateOwn,
	ActionDeleteAll,
	ActionDeleteOwn,
	ActionApprove,
	ActionExport,
	ActionDisableAll,
}

// GetActionByName 根据名称获取动作
func GetActionByName(name string) *Action {
	for _, action := range StandardActions {
		if action.Name == name {
			return &action
		}
	}
	return nil
}
