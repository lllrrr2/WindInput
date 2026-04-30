package buildvariant

// Variant 标识构建变体（debug/release）。
type Variant string

const (
	VariantDebug   Variant = "debug"
	VariantRelease Variant = "" // 默认空串表示发布版（ldflags 未注入）
)

// variant 通过 ldflags 注入: -X github.com/huanfeng/wind_input/pkg/buildvariant.variant=debug
// 注意：必须保持 string 类型，cmd/link 的 -X 仅支持 string 字段。
var variant = ""

// current 返回当前构建变体（类型化访问入口）。
func current() Variant {
	return Variant(variant)
}

const (
	AppNameRelease = "WindInput"
	AppNameDebug   = "WindInputDebug"
)

func IsDebug() bool {
	return current() == VariantDebug
}

func Suffix() string {
	if current() == VariantDebug {
		return "_debug"
	}
	return ""
}

func AppName() string {
	if current() == VariantDebug {
		return AppNameDebug
	}
	return AppNameRelease
}

func DisplayName() string {
	if current() == VariantDebug {
		return "清风输入法开发版"
	}
	return "清风输入法"
}
