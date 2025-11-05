package register

import "encoding/json"

// serializeMetaToJSON 将元数据 map 序列化为 JSON bytes
func serializeMetaToJSON(metaMap map[string]string) ([]byte, error) {
	if len(metaMap) == 0 {
		return nil, nil
	}

	// 直接使用标准库 JSON 序列化
	return json.Marshal(metaMap)
}
