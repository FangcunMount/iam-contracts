package idutil

import (
	"crypto/rand"
	"time"

	"github.com/sony/sonyflake"
	hashids "github.com/speps/go-hashids"

	"github.com/fangcun-mount/iam-contracts/pkg/util/iputil"
	"github.com/fangcun-mount/iam-contracts/pkg/util/stringutil"
)

// 62进制字母表
const (
	Alphabet62 = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"
	Alphabet36 = "abcdefghijklmnopqrstuvwxyz1234567890"
)

// 雪花算法实例
var sf *sonyflake.Sonyflake

// 初始化雪花算法
func init() {
	var st sonyflake.Settings
	st.MachineID = func() (uint16, error) {
		ip := iputil.GetLocalIP()

		return uint16([]byte(ip)[2])<<8 + uint16([]byte(ip)[3]), nil
	}

	sf = sonyflake.NewSonyflake(st)
	// 如果初始化失败(例如在测试环境中),使用 nil 标记,后续使用降级方案
	if sf == nil {
		// 将在 GetIntID 中使用降级方案
	}
}

// GetIntID 获取雪花算法生成的唯一ID
// 在测试环境或雪花算法不可用时,返回基于时间戳的ID
func GetIntID() uint64 {
	if sf == nil {
		// 降级方案:使用纳秒时间戳作为ID(仅用于测试)
		return uint64(time.Now().UnixNano())
	}

	id, err := sf.NextID()
	if err != nil {
		// 降级方案:使用纳秒时间戳
		return uint64(time.Now().UnixNano())
	}

	return id
}

// GetInstanceID 获取实例ID
func GetInstanceID(uid uint64, prefix string) string {
	hd := hashids.NewData()
	hd.Alphabet = Alphabet36
	hd.MinLength = 6
	hd.Salt = "x20k5x"

	h, err := hashids.NewWithData(hd)
	if err != nil {
		panic(err)
	}

	i, err := h.Encode([]int{int(uid)})
	if err != nil {
		panic(err)
	}

	return prefix + stringutil.Reverse(i)
}

// GetUUID36 获取36进制ID
func GetUUID36(prefix string) string {
	id := GetIntID()
	hd := hashids.NewData()
	hd.Alphabet = Alphabet36

	h, err := hashids.NewWithData(hd)
	if err != nil {
		panic(err)
	}

	i, err := h.Encode([]int{int(id)})
	if err != nil {
		panic(err)
	}

	return prefix + stringutil.Reverse(i)
}

// randString 随机字符串
func randString(letters string, n int) string {
	output := make([]byte, n)

	// We will take n bytes, one byte for each character of output.
	randomness := make([]byte, n)

	// read all random
	_, err := rand.Read(randomness)
	if err != nil {
		panic(err)
	}

	l := len(letters)
	// fill output
	for pos := range output {
		// get random item
		random := randomness[pos]

		// random % 64
		randomPos := random % uint8(l)

		// put into output
		output[pos] = letters[randomPos]
	}

	return string(output)
}

// 生成36位随机字符串
func NewSecretID() string {
	return randString(Alphabet62, 36)
}

// 生成32位随机字符串
func NewSecretKey() string {
	return randString(Alphabet62, 32)
}
