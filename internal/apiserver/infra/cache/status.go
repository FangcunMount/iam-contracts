package cache

// FamilyStatus 描述某个缓存族当前的只读状态。
type FamilyStatus struct {
	Family          Family
	Configured      bool
	Healthy         bool
	EntryCountKnown bool
	Notes           []string
}

// RuntimeStatus 描述某个缓存后端当前的运行状态。
type RuntimeStatus struct {
	Backend    BackendKind
	Configured bool
	Healthy    bool
	Notes      []string
}
