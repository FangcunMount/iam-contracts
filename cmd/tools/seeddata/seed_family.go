package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mozillazg/go-pinyin"

	childApp "github.com/FangcunMount/iam-contracts/internal/apiserver/application/uc/child"
	guardApp "github.com/FangcunMount/iam-contracts/internal/apiserver/application/uc/guardianship"
	ucUOW "github.com/FangcunMount/iam-contracts/internal/apiserver/application/uc/uow"
	userApp "github.com/FangcunMount/iam-contracts/internal/apiserver/application/uc/user"
)

// ==================== 配置常量 ====================

const (
	// defaultFamilyCount 默认生成的家庭数量
	defaultFamilyCount = 1000
	// defaultWorkerCount 默认并发 worker 数量
	defaultWorkerCount = 20
	// maxPhoneRetry 生成唯一手机号最大重试次数
	maxPhoneRetry = 10
	// maxDBRetry 数据库操作最大重试次数（遇到 duplicate key 时）
	maxDBRetry = 3
)

// ==================== 用户中心相关类型定义 ====================

// parentSeed 父/母种子数据
type parentSeed struct {
	Alias    string // 别名，用于后续引用（姓名全拼 + 手机号后4位）
	Name     string // 真实姓名
	Nickname string // 昵称
	Phone    string
	Gender   string // male/female
}

// childrenSeed 儿童种子数据
type childrenSeed struct {
	Alias    string // 别名，用于后续引用
	Name     string
	IDCard   string
	Gender   string
	Birthday string
	Height   uint32 // 厘米
	Weight   uint32 // 克
}

// familySeed 家庭种子数据
type familySeed struct {
	Index    int
	Father   *parentSeed
	Mother   *parentSeed
	Children []childrenSeed
}

// familyDevMode 控制是否输出详细运行日志（开发模式）。
// 在 main 启动时由 --dev 标志设置。
var familyDevMode bool

// famPrintf 仅在开发模式下打印详细日志。
func famPrintf(format string, args ...interface{}) {
	if !familyDevMode {
		return
	}
	fmt.Printf(format, args...)
}

// ==================== PhoneSet 线程安全的手机号集合 ====================

// PhoneSet 线程安全的手机号去重集合
type PhoneSet struct {
	mu     sync.Mutex
	phones map[string]struct{}
}

// newPhoneSet 创建新的 PhoneSet
func newPhoneSet() *PhoneSet {
	return &PhoneSet{
		phones: make(map[string]struct{}, 100000),
	}
}

// Add 添加手机号，返回是否添加成功（false 表示已存在）
func (s *PhoneSet) Add(phone string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.phones[phone]; exists {
		return false
	}
	s.phones[phone] = struct{}{}
	return true
}

// GenerateUniquePhone 生成唯一的手机号
func (s *PhoneSet) GenerateUniquePhone() (string, error) {
	for i := 0; i < maxPhoneRetry; i++ {
		phone := generateFakePhone()
		if s.Add(phone) {
			return phone, nil
		}
	}
	return "", fmt.Errorf("GenerateUniquePhone: too many conflicts after %d attempts", maxPhoneRetry)
}

// ==================== Faker 辅助函数 ====================

// 常见中国姓氏（按人口比例排序的前100个）
var chineseSurnames = []string{
	"王", "李", "张", "刘", "陈", "杨", "黄", "赵", "周", "吴",
	"徐", "孙", "马", "胡", "朱", "郭", "何", "林", "罗", "高",
	"郑", "梁", "谢", "宋", "唐", "许", "韩", "冯", "邓", "曹",
	"彭", "曾", "萧", "田", "董", "潘", "袁", "蔡", "蒋", "余",
	"于", "杜", "叶", "程", "魏", "苏", "吕", "丁", "任", "卢",
	"姚", "沈", "钟", "姜", "崔", "谭", "陆", "范", "汪", "廖",
	"石", "金", "韦", "贾", "夏", "付", "方", "邹", "熊", "白",
	"孟", "秦", "邱", "侯", "江", "尹", "薛", "闫", "雷", "龙",
	"史", "陶", "贺", "毛", "段", "郝", "顾", "龚", "邵", "万",
	"钱", "严", "赖", "覃", "洪", "武", "莫", "孔", "向", "常",
}

// 常见名字用字（混合性别，单字和双字名都从这里取）
var chineseGivenNameChars = []string{
	// 中性/通用
	"文", "华", "明", "国", "建", "平", "军", "海", "云", "林",
	"英", "玉", "春", "秀", "兰", "桂", "芳", "红", "金", "银",
	// 偏男性
	"伟", "强", "刚", "勇", "杰", "磊", "涛", "斌", "鹏", "飞",
	"辉", "超", "浩", "宏", "志", "威", "龙", "峰", "亮", "东",
	"波", "健", "宁", "成", "凯", "兵", "毅", "俊", "帅", "锋",
	// 偏女性
	"丽", "芬", "娟", "敏", "静", "燕", "艳", "霞", "婷", "雪",
	"梅", "莉", "琳", "倩", "颖", "萍", "慧", "娜", "蓉", "洁",
	"珍", "琴", "瑶", "薇", "蕾", "欣", "怡", "雅", "馨", "露",
}

// generateFakePhone 生成假手机号（中国格式）
func generateFakePhone() string {
	// 中国手机号前缀
	prefixes := []string{"130", "131", "132", "133", "134", "135", "136", "137", "138", "139",
		"150", "151", "152", "153", "155", "156", "157", "158", "159",
		"170", "171", "172", "173", "175", "176", "177", "178",
		"180", "181", "182", "183", "184", "185", "186", "187", "188", "189",
		"191", "198", "199"}
	prefix := prefixes[rand.Intn(len(prefixes))]
	// 生成后8位
	suffix := fmt.Sprintf("%08d", rand.Intn(100000000))
	return prefix + suffix
}

// generateFakeName 生成假中文姓名
func generateFakeName(surname string) (string, string) {
	if surname == "" {
		// 随机选择姓氏
		surname = chineseSurnames[rand.Intn(len(chineseSurnames))]
	}

	// 随机决定名字长度（1-2个字，70%双字名，30%单字名）
	givenName := ""
	if rand.Float32() < 0.7 {
		// 双字名
		char1 := chineseGivenNameChars[rand.Intn(len(chineseGivenNameChars))]
		char2 := chineseGivenNameChars[rand.Intn(len(chineseGivenNameChars))]
		givenName = char1 + char2
	} else {
		// 单字名
		givenName = chineseGivenNameChars[rand.Intn(len(chineseGivenNameChars))]
	}

	return surname, givenName
}

// 中国身份证号地区码（部分常用）
var areaCodes = []string{
	"110101", "110102", "110105", "110106", "110107", "110108", "110109", "110111", // 北京
	"310101", "310104", "310105", "310106", "310107", "310109", "310110", "310112", // 上海
	"440103", "440104", "440105", "440106", "440111", "440112", "440113", "440114", // 广州
	"440303", "440304", "440305", "440306", "440307", "440308", "440309", "440310", // 深圳
	"330102", "330103", "330104", "330105", "330106", "330108", "330109", "330110", // 杭州
	"320102", "320104", "320105", "320106", "320111", "320113", "320114", "320115", // 南京
	"510104", "510105", "510106", "510107", "510108", "510112", "510113", "510114", // 成都
	"420102", "420103", "420104", "420105", "420106", "420107", "420111", "420112", // 武汉
	"500101", "500102", "500103", "500104", "500105", "500106", "500107", "500108", // 重庆
	"610102", "610103", "610104", "610111", "610112", "610113", "610114", "610115", // 西安
}

// 身份证校验码权重
var idCardWeights = []int{7, 9, 10, 5, 8, 4, 2, 1, 6, 3, 7, 9, 10, 5, 8, 4, 2}

// 身份证校验码对应值
var idCardCheckCodes = []byte{'1', '0', 'X', '9', '8', '7', '6', '5', '4', '3', '2'}

// generateFakeIDCard 生成合法的18位身份证号
// 格式：6位地区码 + 8位出生日期 + 3位顺序码 + 1位校验码
func generateFakeIDCard() string {
	// 1. 随机选择地区码
	areaCode := areaCodes[rand.Intn(len(areaCodes))]

	// 2. 生成出生日期（3-15岁的儿童）
	now := time.Now()
	ageYears := 3 + rand.Intn(13) // 3-15岁
	ageDays := rand.Intn(365)
	birthDate := now.AddDate(-ageYears, 0, -ageDays)
	birthStr := birthDate.Format("20060102")

	// 3. 生成3位顺序码（奇数为男，偶数为女，这里随机）
	// 注意：顺序码 "000" 在某些系统被视为非法，避免生成 000
	seq := rand.Intn(999) + 1 // 1..999
	seqCode := fmt.Sprintf("%03d", seq)

	// 4. 计算校验码
	first17 := areaCode + birthStr + seqCode
	checkCode := calculateIDCardCheckCode(first17)

	return first17 + string(checkCode)
}

// calculateIDCardCheckCode 计算身份证校验码
func calculateIDCardCheckCode(first17 string) byte {
	if len(first17) != 17 {
		return '0'
	}
	sum := 0
	for i := 0; i < 17; i++ {
		digit := int(first17[i] - '0')
		sum += digit * idCardWeights[i]
	}
	return idCardCheckCodes[sum%11]
}

// nameToPinyin 将中文名转换为拼音
func nameToPinyin(name string) string {
	args := pinyin.NewArgs()
	args.Style = pinyin.Normal
	result := pinyin.Pinyin(name, args)
	var builder strings.Builder
	for _, py := range result {
		if len(py) > 0 {
			builder.WriteString(py[0])
		}
	}
	return builder.String()
}

// generateAlias 生成别名：姓名全拼 + 手机号后4位
func generateAlias(name, phone string) string {
	py := nameToPinyin(name)
	suffix := ""
	if len(phone) >= 4 {
		suffix = phone[len(phone)-4:]
	}
	return py + suffix
}

// ==================== 儿童数据生成函数 ====================

// generateChildBirthday 生成儿童生日（3-12岁）
func generateChildBirthday() string {
	now := time.Now()
	// 生成 3-12 岁的儿童
	ageYears := 3 + rand.Intn(10) // 3-12岁
	ageDays := rand.Intn(365)
	birthDate := now.AddDate(-ageYears, 0, -ageDays)
	return birthDate.Format("2006-01-02")
}

// generateChildHeight 根据年龄生成身高（厘米）
func generateChildHeight(birthday string) uint32 {
	age := calculateAge(birthday)
	// 基础身高 + 年龄增量 + 随机波动
	baseHeight := 80 + age*6 // 粗略估算：3岁约98cm，每年增长6cm
	variation := rand.Intn(15) - 7
	height := baseHeight + variation
	if height < 80 {
		height = 80
	}
	if height > 180 {
		height = 180
	}
	return uint32(height)
}

// generateChildWeight 根据年龄生成体重（克）
func generateChildWeight(birthday string) uint32 {
	age := calculateAge(birthday)
	// 基础体重 + 年龄增量 + 随机波动
	baseWeight := 12 + age*2 // 粗略估算：3岁约18kg，每年增长2kg
	variation := rand.Intn(6) - 3
	weight := baseWeight + variation
	if weight < 10 {
		weight = 10
	}
	if weight > 80 {
		weight = 80
	}
	return uint32(weight * 1000) // 转换为克
}

// calculateAge 根据生日计算年龄
func calculateAge(birthday string) int {
	birthDate, err := time.Parse("2006-01-02", birthday)
	if err != nil {
		return 5 // 默认5岁
	}
	now := time.Now()
	age := now.Year() - birthDate.Year()
	if now.YearDay() < birthDate.YearDay() {
		age--
	}
	return age
}

// 常见女性名字用字
var femaleNameChars = map[rune]bool{
	// 花草类
	'花': true, '芳': true, '芬': true, '兰': true, '梅': true, '菊': true, '莲': true, '荷': true,
	'蓉': true, '薇': true, '萍': true, '莉': true, '茹': true, '蕊': true, '芸': true, '蔓': true,
	'苗': true, '芝': true, '菲': true, '蕾': true, '茜': true, '莎': true, '萱': true,
	// 美丽/柔美类
	'美': true, '丽': true, '娜': true, '婷': true, '雅': true, '静': true, '婉': true, '娟': true,
	'娥': true, '姿': true, '妍': true, '姣': true, '婵': true, '媛': true, '妮': true, '娴': true,
	'嫣': true, '姝': true, '妙': true, '婕': true, '娇': true, '媚': true, '姗': true, '嫦': true,
	// 珍贵类
	'珍': true, '珠': true, '琳': true, '瑶': true, '琴': true, '玲': true, '瑾': true, '璐': true,
	'瑛': true, '珊': true, '琪': true, '璇': true, '珺': true, '琬': true, '瑜': true, '玥': true,
	// 柔和/温婉类
	'柔': true, '淑': true, '惠': true, '慧': true, '秀': true, '倩': true, '巧': true, '纤': true,
	'素': true, '洁': true, '雯': true, '霞': true, '露': true, '月': true, '雪': true,
	'冰': true, '虹': true, '彩': true, '凤': true, '燕': true, '莺': true, '蝶': true,
	// 情感类
	'爱': true, '怡': true, '悦': true, '欣': true, '馨': true, '思': true, '念': true,
	// 颜色类（女性倾向）
	'红': true, '粉': true, '紫': true, '黛': true, '碧': true, '翠': true,
	// 其他常见女名用字
	'颖': true, '敏': true, '晴': true, '岚': true, '影': true, '梦': true, '瑞': true,
}

// 常见男性名字用字
var maleNameChars = map[rune]bool{
	// 刚强类
	'刚': true, '强': true, '勇': true, '猛': true, '威': true, '毅': true, '坚': true, '雄': true,
	'壮': true, '健': true, '彪': true, '豪': true, '杰': true, '伟': true, '峰': true, '磊': true,
	'力': true, '武': true, '军': true, '兵': true, '锋': true, '钢': true, '铁': true, '石': true,
	// 志向类
	'志': true, '鹏': true, '飞': true, '翔': true, '腾': true, '龙': true, '虎': true, '鹰': true,
	'骏': true, '驰': true, '超': true, '越': true, '冠': true, '胜': true, '凯': true, '辉': true,
	// 才德类
	'才': true, '智': true, '博': true, '斌': true, '彬': true, '贤': true,
	'德': true, '仁': true, '义': true, '礼': true, '信': true, '忠': true, '孝': true, '廉': true,
	// 成就类
	'成': true, '功': true, '业': true, '建': true, '立': true, '国': true, '邦': true, '振': true,
	'兴': true, '昌': true, '盛': true, '荣': true, '富': true, '贵': true, '福': true, '禄': true,
	// 自然（阳刚）类
	'山': true, '海': true, '江': true, '河': true, '川': true, '林': true, '森': true, '松': true,
	'柏': true, '杨': true, '天': true, '阳': true, '晨': true, '旭': true, '晖': true, '明': true,
	'光': true, '耀': true, '雷': true, '霆': true, '风': true, '云': true,
	// 其他常见男名字
	'浩': true, '宏': true, '鸿': true, '涛': true, '波': true, '平': true, '安': true, '康': true,
	'宁': true, '泽': true, '鑫': true, '亮': true, '达': true, '栋': true, '梁': true,
	'华': true, '东': true, '南': true, '北': true, '西': true, '中': true,
}

// guessGenderByName 根据中文名字猜测性别
// 返回 "male" 或 "female"
func guessGenderByName(name string) string {
	// 去掉姓氏，只看名字部分（假设姓是1个字）
	runes := []rune(name)
	var nameRunes []rune
	if len(runes) > 1 {
		nameRunes = runes[1:] // 去掉姓氏
	} else {
		nameRunes = runes
	}

	maleScore := 0
	femaleScore := 0

	for _, r := range nameRunes {
		if maleNameChars[r] {
			maleScore++
		}
		if femaleNameChars[r] {
			femaleScore++
		}
	}

	// 根据得分判断
	if femaleScore > maleScore {
		return "female"
	} else if maleScore > femaleScore {
		return "male"
	}

	// 平分或无法判断时随机
	if rand.Float32() < 0.5 {
		return "male"
	}
	return "female"
}

// ==================== 家庭数据生成函数 ====================

// generateFamily 生成一个家庭的种子数据
// 规则：
// - 50% 只有母亲, 15% 只有父亲, 35% 有父亲和母亲
// - 70% 有1个孩子, 25% 有2个孩子, 5% 有3个孩子
func generateFamily(index int, phoneSet *PhoneSet) (*familySeed, error) {
	family := &familySeed{
		Index: index,
	}

	// 决定家长组成
	r := rand.Float32()
	hasFather := r >= 0.50 // 50% 以上有父亲
	fatherSurname := ""
	hasMother := r < 0.50 || r >= 0.65 // 小于65% 有母亲（即50%只有母亲 + 35%有双亲）

	if hasFather {
		phone, err := phoneSet.GenerateUniquePhone()
		if err != nil {
			return nil, fmt.Errorf("generate father phone: %w", err)
		}
		fatherSurname, name := generateFakeName(fatherSurname)
		alias := generateAlias(fatherSurname+name, phone)
		family.Father = &parentSeed{
			Name:     fatherSurname + name,
			Nickname: alias, // 昵称 = 姓名全拼 + 手机号后4位
			Phone:    phone,
			Gender:   "male",
			Alias:    alias,
		}
	}

	if hasMother {
		phone, err := phoneSet.GenerateUniquePhone()
		if err != nil {
			return nil, fmt.Errorf("generate mother phone: %w", err)
		}
		motherSurname, name := generateFakeName(fatherSurname)
		alias := generateAlias(motherSurname+name, phone)
		family.Mother = &parentSeed{
			Name:     motherSurname + name,
			Nickname: alias, // 昵称 = 姓名全拼 + 手机号后4位
			Phone:    phone,
			Gender:   "female",
			Alias:    alias,
		}
	}

	// 决定孩子数量
	r = rand.Float32()
	var childCount int
	switch {
	case r < 0.70:
		childCount = 1
	case r < 0.95:
		childCount = 2
	default:
		childCount = 3
	}

	family.Children = make([]childrenSeed, 0, childCount)
	for i := 0; i < childCount; i++ {
		birthday := generateChildBirthday()
		childSurname, name := generateFakeName(fatherSurname)
		gender := guessGenderByName(name) // 根据名字推测性别
		idCard := generateFakeIDCard()

		child := childrenSeed{
			Alias:    fmt.Sprintf("child_%d_%d", index, i),
			Name:     childSurname + name,
			IDCard:   idCard,
			Gender:   gender,
			Birthday: birthday,
			Height:   generateChildHeight(birthday),
			Weight:   generateChildWeight(birthday),
		}
		family.Children = append(family.Children, child)
	}

	return family, nil
}

// ==================== FamilySeedTask 任务定义 ====================

// familySeedTask 家庭 seed 任务
type familySeedTask struct {
	Index int
}

// familyServices 家庭相关的应用服务集合
type familyServices struct {
	UserAppSrv     userApp.UserApplicationService
	UserProfileSrv userApp.UserProfileApplicationService
	ChildAppSrv    childApp.ChildApplicationService
	GuardAppSrv    guardApp.GuardianshipApplicationService
	GuardQuerySrv  guardApp.GuardianshipQueryApplicationService
}

// Run 执行家庭 seed 任务
func (t *familySeedTask) Run(
	ctx context.Context,
	services *familyServices,
	phoneSet *PhoneSet,
	collectionURL string,
	iamServiceURL string,
	adminLoginID string,
	adminPassword string,
) error {
	// 1. 生成家庭数据
	family, err := generateFamily(t.Index, phoneSet)
	if err != nil {
		return fmt.Errorf("task %d: generate family: %w", t.Index, err)
	}

	// 2. 创建父亲（如果存在）
	var fatherID string
	if family.Father != nil {
		fatherID, err = createParentWithRetry(ctx, services.UserAppSrv, services.UserProfileSrv, family.Father, phoneSet)
		if err != nil {
			return fmt.Errorf("task %d: create father: %w", t.Index, err)
		}
	}

	// 3. 创建母亲（如果存在）
	var motherID string
	if family.Mother != nil {
		motherID, err = createParentWithRetry(ctx, services.UserAppSrv, services.UserProfileSrv, family.Mother, phoneSet)
		if err != nil {
			return fmt.Errorf("task %d: create mother: %w", t.Index, err)
		}
	}

	// 4. 创建孩子并建立监护关系
	for i, child := range family.Children {
		childID, err := createChild(ctx, services.ChildAppSrv, &child)
		if err != nil {
			return fmt.Errorf("task %d: create child %d: %w", t.Index, i, err)
		}

		// 建立监护关系
		if fatherID != "" {
			if err := createGuardianship(ctx, services.GuardAppSrv, services.GuardQuerySrv, fatherID, childID, "parent"); err != nil {
				return fmt.Errorf("task %d: create father guardianship for child %d: %w", t.Index, i, err)
			}
		}
		if motherID != "" {
			if err := createGuardianship(ctx, services.GuardAppSrv, services.GuardQuerySrv, motherID, childID, "parent"); err != nil {
				return fmt.Errorf("task %d: create mother guardianship for child %d: %w", t.Index, i, err)
			}
		}

		// 5. 创建受试者（testee）- 使用父母账号获取 token
		var guardianPhone string
		var guardianUserID string
		if family.Father != nil {
			guardianPhone = family.Father.Phone
			guardianUserID = fatherID
		} else if family.Mother != nil {
			guardianPhone = family.Mother.Phone
			guardianUserID = motherID
		}
		if guardianPhone != "" && collectionURL != "" {
			if err := createTestee(ctx, collectionURL, iamServiceURL, adminLoginID, adminPassword, guardianUserID, guardianPhone, childID, &child); err != nil {
				// 受试者创建失败不阻断流程，记录错误继续
				fmt.Printf("\nWarning: task %d: create testee for child %d failed: %v\n", t.Index, i, err)
			}
		}
	}

	return nil
}

// ==================== 数据库操作函数 ====================

// createParentWithRetry 创建父/母用户，遇到 duplicate key 时重试
func createParentWithRetry(
	ctx context.Context,
	userAppSrv userApp.UserApplicationService,
	userProfileSrv userApp.UserProfileApplicationService,
	parent *parentSeed,
	phoneSet *PhoneSet,
) (string, error) {
	var lastErr error
	for retry := 0; retry < maxDBRetry; retry++ {
		result, err := userAppSrv.Register(ctx, userApp.RegisterUserDTO{
			Name:  parent.Name,
			Phone: parent.Phone,
		})
		if err == nil {
			// 注册成功后设置昵称
			if parent.Nickname != "" {
				_ = userProfileSrv.Renickname(ctx, result.ID, parent.Nickname)
			}
			return result.ID, nil
		}

		// 检查是否是 duplicate key 错误
		if isDuplicateKeyError(err) {
			// 生成新的手机号重试
			newPhone, phoneErr := phoneSet.GenerateUniquePhone()
			if phoneErr != nil {
				return "", fmt.Errorf("retry %d: %w", retry, phoneErr)
			}
			parent.Phone = newPhone
			newAlias := generateAlias(parent.Name, newPhone)
			parent.Alias = newAlias
			parent.Nickname = newAlias // 昵称也同步更新
			lastErr = err
			continue
		}

		return "", err
	}
	return "", fmt.Errorf("max retries exceeded: %w", lastErr)
}

// createChild 创建儿童
func createChild(
	ctx context.Context,
	childAppSrv childApp.ChildApplicationService,
	child *childrenSeed,
) (string, error) {
	height := child.Height
	weight := child.Weight
	result, err := childAppSrv.Register(ctx, childApp.RegisterChildDTO{
		Name:     child.Name,
		Gender:   genderStringToUint8(child.Gender),
		Birthday: child.Birthday,
		IDCard:   child.IDCard,
		Height:   &height,
		Weight:   &weight,
	})
	if err != nil {
		return "", err
	}
	return result.ID, nil
}

// createGuardianship 创建监护关系
func createGuardianship(
	ctx context.Context,
	guardAppSrv guardApp.GuardianshipApplicationService,
	guardQuerySrv guardApp.GuardianshipQueryApplicationService,
	userID, childID, relation string,
) error {
	// 先检查是否已存在
	isGuardian, err := guardQuerySrv.IsGuardian(ctx, userID, childID)
	if err != nil {
		return fmt.Errorf("check guardian: %w", err)
	}
	if isGuardian {
		return nil // 已存在，跳过
	}

	err = guardAppSrv.AddGuardian(ctx, guardApp.AddGuardianDTO{
		UserID:   userID,
		ChildID:  childID,
		Relation: relation,
	})
	if err != nil && !isDuplicateGuardianError(err) {
		return err
	}
	return nil
}

// isDuplicateKeyError 检查是否是 duplicate key 错误
func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "duplicate") ||
		strings.Contains(errStr, "unique constraint") ||
		strings.Contains(errStr, "already exists")
}

// isDuplicateGuardianError 检查是否是重复监护关系错误
func isDuplicateGuardianError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "already exists")
}

// createTestee 调用 collection 服务 API 创建受试者
func createTestee(
	ctx context.Context,
	collectionURL string,
	iamServiceURL string,
	adminLoginID string,
	adminPassword string,
	guardianUserID string,
	guardianPhone string,
	childID string,
	child *childrenSeed,
) error {
	// 如果没有配置 collection URL，跳过
	if collectionURL == "" {
		return nil
	}

	// 1. 获取超级管理员 token（缓存）
	token, err := getSuperAdminToken(ctx, iamServiceURL, adminLoginID, adminPassword)
	if err != nil {
		return fmt.Errorf("login as super admin: %w", err)
	}

	// 如果没有 token，跳过
	if token == "" {
		return nil
	}

	// 2. 准备请求数据
	gender := uint8(1) // 默认男性
	if child.Gender == "female" {
		gender = 2
	}

	reqData := map[string]interface{}{
		"iam_user_id":  guardianUserID,
		"iam_child_id": childID,
		"name":         child.Name,
		"gender":       gender,
		"birthday":     child.Birthday,
		"source":       "imported",
	}

	jsonData, err := json.Marshal(reqData)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	// 3. 调用 collection 服务 API
	apiURL := collectionURL + "/testees"
	// 记录请求详情（仅开发模式）
	famPrintf("📤 发送创建受试者请求 url=%s method=POST iam_user_id=%s iam_child_id=%s request_body=%s has_token=true token_prefix=<hidden>\n", apiURL, guardianUserID, childID, string(jsonData))

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("❌ 请求失败: %v\n", err)
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应体
	var respBodyBytes bytes.Buffer
	_, _ = respBodyBytes.ReadFrom(resp.Body)
	respBodyStr := respBodyBytes.String()

	// 记录响应详情（仅开发模式）
	famPrintf("📥 收到创建受试者响应 status=%d status_text=%s response_headers=%v response_body=%s\n", resp.StatusCode, resp.Status, resp.Header, respBodyStr)

	// 4. 检查响应
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated {
		return nil // 创建成功
	}

	// 读取错误响应
	var errResp map[string]interface{}
	_ = json.Unmarshal(respBodyBytes.Bytes(), &errResp)
	fmt.Printf("❌ 创建受试者失败 status=%d response=%v\n", resp.StatusCode, errResp)
	return fmt.Errorf("collection API returned status %d: %v", resp.StatusCode, errResp)
}

// ==================== Worker Pool 实现 ====================

// printProgress 打印进度条
func printProgress(current, total int64, failed int64) {
	const barWidth = 40
	percent := float64(current) / float64(total)
	filled := int(percent * barWidth)

	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

	// \r 回到行首，覆盖之前的输出
	if failed > 0 {
		fmt.Printf("\r🏠 家庭数据: [%s] %d/%d (%.1f%%) ⚠️ 失败:%d", bar, current, total, percent*100, failed)
	} else {
		fmt.Printf("\r🏠 家庭数据: [%s] %d/%d (%.1f%%)", bar, current, total, percent*100)
	}
}

// seedFamilyCenter 使用 worker pool 模式创建家庭数据
//
// 设计说明：
// 1. 主 goroutine 只推送任务索引到 channel
// 2. 固定数量的 worker 从 channel 获取任务并执行完整业务逻辑
// 3. 手机号唯一性通过 PhoneSet 在内存中去重，数据库唯一索引兜底
// 4. 遇到 duplicate key error 时自动重试
func seedFamilyCenter(ctx context.Context, deps *dependencies, familyCount, workerCount int) error {
	if familyCount <= 0 {
		familyCount = defaultFamilyCount
	}
	if workerCount <= 0 {
		workerCount = defaultWorkerCount
	}

	fmt.Printf("🏠 开始创建家庭数据 (总数: %d, 并发: %d)\n", familyCount, workerCount)

	// 初始化应用服务
	uow := ucUOW.NewUnitOfWork(deps.DB)
	services := &familyServices{
		UserAppSrv:     userApp.NewUserApplicationService(uow),
		UserProfileSrv: userApp.NewUserProfileApplicationService(uow),
		ChildAppSrv:    childApp.NewChildApplicationService(uow),
		GuardAppSrv:    guardApp.NewGuardianshipApplicationService(uow),
		GuardQuerySrv:  guardApp.NewGuardianshipQueryApplicationService(uow),
	}

	// 创建手机号去重集合
	phoneSet := newPhoneSet()

	// 创建任务 channel
	taskCh := make(chan *familySeedTask, workerCount*2)

	// 统计
	var successCount, failCount int64
	var wg sync.WaitGroup
	// 失败任务详情收集（用于排查并发下偶发错误）
	var failedMu sync.Mutex
	failedDetails := make([]string, 0, 8)

	// 打印初始进度
	printProgress(0, int64(familyCount), 0)

	// 获取配置
	collectionURL := deps.Config.CollectionURL
	iamServiceURL := deps.Config.IAMServiceURL
	adminLoginID, adminPassword := resolveAdminLogin(deps.Config)

	// 预拉取超级管理员 token，避免 worker 启动后并发触发首次登录风暴
	if iamServiceURL != "" {
		if tkn, err := getSuperAdminToken(ctx, iamServiceURL, adminLoginID, adminPassword); err != nil {
			fmt.Printf("⚠️  预拉取 super-admin token 失败: %v (workers may retry)\n", err)
		} else {
			famPrintf("ℹ️  预拉取 super-admin token 成功，缓存到期: %v\n", superAdminTokenExpiry)
			_ = tkn
		}
	}

	// 启动 workers
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for task := range taskCh {
				if err := task.Run(ctx, services, phoneSet, collectionURL, iamServiceURL, adminLoginID, adminPassword); err != nil {
					// 记录失败详情，便于排查（只保存有限条）
					failedMu.Lock()
					if len(failedDetails) < 100 {
						failedDetails = append(failedDetails, fmt.Sprintf("task %d: %v", task.Index, err))
					}
					failedMu.Unlock()

					failed := atomic.AddInt64(&failCount, 1)
					success := atomic.LoadInt64(&successCount)
					printProgress(success+failed, int64(familyCount), failed)
				} else {
					success := atomic.AddInt64(&successCount, 1)
					failed := atomic.LoadInt64(&failCount)
					printProgress(success+failed, int64(familyCount), failed)
				}
			}
		}(i)
	}

	// 主 goroutine 推送任务
	for i := 0; i < familyCount; i++ {
		select {
		case <-ctx.Done():
			close(taskCh)
			wg.Wait()
			fmt.Println() // 换行
			return ctx.Err()
		case taskCh <- &familySeedTask{Index: i}:
		}
	}
	close(taskCh)

	// 等待所有 worker 完成
	wg.Wait()

	// 完成后换行并打印结果
	fmt.Println()
	if failCount > 0 {
		fmt.Printf("⚠️  家庭数据创建完成: 成功 %d, 失败 %d, 总计 %d\n", successCount, failCount, familyCount)
		// 打印部分失败详情以便排查（最多 100 条）
		failedMu.Lock()
		if len(failedDetails) > 0 {
			fmt.Println("---- 失败任务示例 ----")
			for i, d := range failedDetails {
				if i >= 20 {
					fmt.Printf("... 共 %d 条失败，已显示 20 条样例\n", len(failedDetails))
					break
				}
				fmt.Printf("%s\n", d)
			}
			fmt.Println("---- 结束 ----")
		}
		failedMu.Unlock()
		return fmt.Errorf("部分家庭创建失败: %d/%d", failCount, familyCount)
	}
	fmt.Printf("✅ 家庭数据创建完成: %d 个家庭\n", successCount)
	return nil
}

// resolveAdminLogin 从配置解析管理员登录ID和密码。
func resolveAdminLogin(cfg *SeedConfig) (loginID, password string) {
	if cfg == nil {
		return "", ""
	}

	// 记录用户手机号/邮箱，按别名索引
	userPhones := make(map[string]string, len(cfg.Users))
	userEmails := make(map[string]string, len(cfg.Users))
	userByAlias := make(map[string]UserConfig, len(cfg.Users))
	for _, u := range cfg.Users {
		userPhones[u.Alias] = u.Phone
		userEmails[u.Alias] = u.Email
		userByAlias[u.Alias] = u
	}

	for _, ac := range cfg.Accounts {
		if ac.Provider != "operation" {
			continue
		}
		// 取第一个 operation 账号作为管理员凭据
		password = ac.Password
		loginID = resolveLoginID(ac, userByAlias[ac.UserAlias])
		break
	}

	if loginID == "" {
		// 回退优先使用 system/admin 的邮箱式登录名，再退回手机号
		loginID = userEmails["system"]
		if loginID == "" {
			loginID = userEmails["admin"]
		}
		if loginID == "" {
			loginID = userPhones["admin"]
		}
	}

	return normalizeLoginID(loginID), password
}

// loginAsSuperAdmin 使用超级管理员账号登录 IAM 服务获取 token
// loginAsSuperAdmin 使用超级管理员账号登录 IAM 服务获取 TokenPair（含过期信息）
func loginAsSuperAdmin(ctx context.Context, iamServiceURL, loginID, password string) (TokenPair, error) {
	// 优先使用传入的 loginID，否则回退默认 system 邮箱式登录名
	if loginID == "" {
		loginID = "system@fangcunmount.com"
	}
	if password == "" {
		password = "Admin@123"
	}

	loginID = normalizeLoginID(loginID)

	// 构建接口期望的 JSON 凭证
	credentials, err := json.Marshal(struct {
		Username string `json:"username"`
		Password string `json:"password"`
		TenantID uint64 `json:"tenant_id,omitempty"`
	}{
		Username: loginID,
		Password: password,
	})
	if err != nil {
		return TokenPair{}, fmt.Errorf("marshal credentials: %w", err)
	}

	reqBody := LoginRequest{
		Method:      "password",
		Credentials: credentials,
		DeviceID:    "seeddata-collection",
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return TokenPair{}, fmt.Errorf("marshal request: %w", err)
	}

	url := iamServiceURL + "/authn/login"
	// 重试策略：最多尝试 3 次，指数退避
	maxAttempts := 3
	client := &http.Client{Timeout: 10 * time.Second}
	var lastErr error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			return TokenPair{}, fmt.Errorf("create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)
			// 网络/连接错误可重试
			if attempt < maxAttempts {
				sleep := time.Duration(200*(1<<uint(attempt-1))) * time.Millisecond
				fmt.Printf("⚠️ login attempt %d failed (network): %v, retrying after %s\n", attempt, err, sleep)
				time.Sleep(sleep)
				continue
			}
			return TokenPair{}, lastErr
		}

		// 读取响应体为 bytes，便于日志与解析
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		// 尝试解析错误响应体为 map 以便记录
		var respMap map[string]interface{}
		_ = json.Unmarshal(bodyBytes, &respMap)

		if resp.StatusCode >= 400 {
			// 5xx 视为临时性错误，可重试
			if resp.StatusCode >= 500 && attempt < maxAttempts {
				fmt.Printf("⚠️ login attempt %d got status %d, response=%v, retrying...\n", attempt, resp.StatusCode, respMap)
				sleep := time.Duration(200*(1<<uint(attempt-1))) * time.Millisecond
				time.Sleep(sleep)
				continue
			}
			return TokenPair{}, fmt.Errorf("login failed: status=%d, response=%v", resp.StatusCode, respMap)
		}

		// 正常响应，解析包装格式
		var wrapper struct {
			Code    int       `json:"code"`
			Message string    `json:"message"`
			Data    TokenPair `json:"data"`
		}
		if err := json.Unmarshal(bodyBytes, &wrapper); err != nil {
			return TokenPair{}, fmt.Errorf("decode response: %w", err)
		}
		if wrapper.Code != 0 {
			return TokenPair{}, fmt.Errorf("login failed: code=%d, message=%s, data=%v", wrapper.Code, wrapper.Message, wrapper.Data)
		}

		return wrapper.Data, nil
	}

	return TokenPair{}, lastErr
}
