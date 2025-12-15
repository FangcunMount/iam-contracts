package main

// ==================== 辅助函数 ====================

// genderStringToUint8 将字符串性别转换为 uint8
func genderStringToUint8(gender string) uint8 {
	switch gender {
	case "male":
		return 1
	case "female":
		return 2
	default:
		return 0
	}
}
