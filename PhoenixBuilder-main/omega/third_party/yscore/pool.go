package yscore

import (
	_ "embed"
	"phoenixbuilder/omega/defines"

	"phoenixbuilder/omega/third_party/yscore/guild"
)

// 插件所属组织或者作者名
const NAMESPACE = "风也温柔"

// 插件默认配置
//
//go:embed default_components.json
var DefaultComponentConfigByte []byte

// 插件列表
var Components = map[string]defines.Component{
	"YsCore_自定义违禁词封禁系统会员组件": &BanName{BasicComponent: &defines.BasicComponent{}},
	"YsCore_自定义模糊回答会员组件":    &Talk{BasicComponent: &defines.BasicComponent{}},
	"公会系统":         &guild.Guild{BasicComponent: &defines.BasicComponent{}},
	"yscore_白名单系统": &WhiteList{BasicComponent: &defines.BasicComponent{}},
	//"YsCore_仓库系统":  &WareHouse{BasicComponent: &defines.BasicComponent{}},
	"YsCore_抽奖系统":    &Lottery{BasicComponent: &defines.BasicComponent{}},
	"yscore_虚拟储存":    &WareHouse{BasicComponent: &defines.BasicComponent{}},
	"YsCore_自定义生产计划": &Production{BasicComponent: &defines.BasicComponent{}},
	"全球商店系统":         &Store{BasicComponent: &defines.BasicComponent{}},
	"指令映射":           &TellCmd{BasicComponent: &defines.BasicComponent{}},
	"yscore_随机事件":    &RandomEvent{BasicComponent: &defines.BasicComponent{}},
}
