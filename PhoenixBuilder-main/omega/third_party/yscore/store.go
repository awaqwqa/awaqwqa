package yscore

import (
	"encoding/json"
	"fmt"
	"phoenixbuilder/minecraft/protocol/packet"
	"phoenixbuilder/omega/defines"
	"strconv"

	"github.com/pterm/pterm"
)

type Store struct {
	*defines.BasicComponent
	Triggers []string                     `json:"触发词"`
	Usage    string                       `json:"描述"`
	Pool     map[string]map[string]*Wares `json:"商品"`
	Menu     map[string]string            `json:"菜单"`
	Score    string                       `json:"所需计分板"`
}

// 商品
type Wares struct {
	//价格
	Price int `json:"价格"`
	//描述
	Usage string `json:"描述"`
	//执行指令
	Cmds []string `json:"执行指令"`
}

func (o *Store) Init(cfg *defines.ComponentConfig) {
	marshal, _ := json.Marshal(cfg.Configs)
	if err := json.Unmarshal(marshal, o); err != nil {
		panic(err)
	}
}
func (o *Store) Inject(frame defines.MainFrame) {
	o.Frame = frame
	o.BasicComponent.Inject(frame)
	//o.Frame.GetGameListener().SetGameChatInterceptor(o.onChat)
	o.Frame.GetGameListener().SetGameMenuEntry(&defines.GameMenuEntry{
		MenuEntry: defines.MenuEntry{
			Triggers:     o.Triggers,
			ArgumentHint: "",
			FinalTrigger: false,
			Usage:        o.Usage,
		},
		OptionalOnTriggerFn: o.onMenu,
	})
	CreateNameHash(o.Frame)
}
func (b *Store) onMenu(chat *defines.GameChat) bool {

	Sayto(b.Frame, chat.Name, b.Menu["主菜单"])
	list := make(map[string]string)
	num := 0
	msg := ""
	for k, _ := range b.Pool {
		relist := map[string]interface{}{
			"i":    num,
			"type": k,
		}
		msg += FormateMsg(b.Frame, relist, b.Menu["类型选择模板"])
		list[strconv.Itoa(num)] = k
		num++
	}
	Sayto(b.Frame, chat.Name, msg)
	b.Frame.GetGameControl().SetOnParamMsg(chat.Name, func(Newchat *defines.GameChat) (catch bool) {
		if len(Newchat.Msg) > 0 {
			if typename, ok := list[Newchat.Msg[0]]; ok {
				b.PoolMenu(Newchat.Name, typename)
			}
		}

		return true
	})

	return true
}

// 子菜单
func (b *Store) PoolMenu(name string, typeName string) {
	num := 0
	list := make(map[string]string)
	for k, v := range b.Pool[typeName] {
		list[strconv.Itoa(num)] = k
		relist := map[string]interface{}{
			"i":  num,
			"名字": k,
			"价格": v.Price,
			"描述": v.Usage,
		}
		Sayto(b.Frame, name, FormateMsg(b.Frame, relist, b.Menu["子菜单模板"]))
		num++
	}
	b.Frame.GetGameControl().SetOnParamMsg(name, func(chat *defines.GameChat) (catch bool) {
		if len(chat.Msg) > 0 {
			if WaresName, ok := list[chat.Msg[0]]; ok {
				b.Buy(name, *b.Pool[typeName][WaresName])
			} else {
				Sayto(b.Frame, name, "无效输入")
			}
		}

		return true
	})
}

// 购买
func (b *Store) Buy(name string, data Wares) {
	Sayto(b.Frame, name, b.Menu["询问购买数量"])

	b.Frame.GetGameControl().SetOnParamMsg(name, func(Newchat *defines.GameChat) (catch bool) {
		if len(Newchat.Msg) > 0 && CheckIsNum(Newchat.Msg[0]) {
			num, _ := strconv.Atoi(Newchat.Msg[0])
			//开始检测钱是否足够
			go func() {
				b.Frame.GetGameControl().SendCmdAndInvokeOnResponse(fmt.Sprintf("scoreboard players test @a[name=\"%v\"] %v %v *", name, b.Score, num*data.Price), func(output *packet.CommandOutput) {
					fmt.Println("test")
					if output.SuccessCount > 0 {
						for _, cmd := range data.Cmds {
							relist := map[string]interface{}{
								"player": name,
								"数量":     num,
								"价格":     num * data.Price,
							}
							Newcmd := FormateMsg(b.Frame, relist, cmd)
							b.Frame.GetGameControl().SendCmdAndInvokeOnResponse(Newcmd, func(output *packet.CommandOutput) {
								if output.SuccessCount > 0 {
									pterm.Info.Printfln("%v指令执行结果为:成功", Newcmd)
								} else {
									pterm.Info.Printfln("%v指令执行结果为:失败 原因为:\n%v", Newcmd, output.OutputMessages)
								}
							})
						}
					} else {
						fmt.Println("余额不足")
						Sayto(b.Frame, name, b.Menu["余额提示"])
					}

				})

			}()
		} else {
			Sayto(b.Frame, name, "输入不符合规范")
		}

		return true
	})

}
