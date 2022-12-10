package yscore

import (
	"encoding/json"
	"fmt"
	"phoenixbuilder/minecraft/protocol/packet"
	"phoenixbuilder/omega/defines"
	"strconv"
	"time"

	"github.com/pterm/pterm"
)

type Production struct {
	*defines.BasicComponent
	DataOfProduction map[string]map[string]string
	Tirgger          []string                    `json:"触发词"`
	Usage            string                      `json:"描述"`
	Menu             map[string]string           `json:"菜单"`
	TitleWord        map[string]string           `json:"提示词"`
	Score            string                      `json:"所需计分板"`
	Equip            map[string]*ProductionEquip `json:"生产装备"`
	DelayTime        int                         `json:"生产计划执行周期(毫秒)"`
}

// 生产装备
type ProductionEquip struct {
	Price int      `json:"价格"`
	Usage string   `json:"描述"`
	Cmds  []string `json:"执行指令"`
}

// 初始化
func (b *Production) Init(cfg *defines.ComponentConfig) {
	m, _ := json.Marshal(cfg.Configs)
	err := json.Unmarshal(m, b)
	if err != nil {
		panic(err)
	}

}

// 注入
func (b *Production) Inject(frame defines.MainFrame) {
	b.Frame = frame
	b.BasicComponent.Inject(frame)
	CreateNameHash(b.Frame)
	b.Frame.GetGameListener().SetGameMenuEntry(&defines.GameMenuEntry{
		MenuEntry: defines.MenuEntry{
			Triggers:     b.Tirgger,
			ArgumentHint: " ",
			FinalTrigger: false,
			Usage:        b.Usage,
		},
		OptionalOnTriggerFn: b.Center,
	})

	b.DataOfProduction = make(map[string]map[string]string)
	b.Frame.GetJsonData("生产装备.json", &b.DataOfProduction)
}
func (b *Production) Activate() {
	pterm.Info.Println("当前延迟毫秒为:", b.DelayTime)
	for {
		time.Sleep(time.Millisecond * time.Duration(b.DelayTime))
		go func() {
			playerPos := <-GetPos(b.Frame, "@a")
			for k, v := range b.DataOfProduction {
				if v == nil {
					b.DataOfProduction[k] = make(map[string]string)
					v = make(map[string]string)
				}
				//遍历出所有的生产装备
				for equipName, _ := range v {
					relist := map[string]interface{}{
						"player": k,
						"x":      playerPos[k][0],
						"y":      playerPos[k][1],
						"z":      playerPos[k][2],
					}
					for _, cmd := range b.Equip[equipName].Cmds {
						//执行指令
						b.Frame.GetGameControl().SendCmd(FormateMsg(b.Frame, relist, cmd))
					}
				}
			}
		}()

	}
}

// 保存数据
func (b *Production) Signal(signal int) error {
	switch signal {
	case defines.SIGNAL_DATA_CHECKPOINT:
		return b.Frame.WriteJsonDataWithTMP("生产装备.json", ".ckpt", &b.DataOfProduction)
	}
	return nil
}

// 处理中心
func (b *Production) Center(chat *defines.GameChat) bool {
	Sayto(b.Frame, chat.Name, b.Menu["主菜单"])
	b.Frame.GetGameControl().SetOnParamMsg(chat.Name, func(chat *defines.GameChat) (catch bool) {
		if len(chat.Msg) > 0 {
			switch chat.Msg[0] {
			case "0":
				b.Store(chat.Name)
			case "1":
				b.WareHouse(chat.Name)
			}
		}

		return true
	})
	return true
}

// 商店
func (b *Production) Store(name string) {
	num := 0
	msg := ""
	list := make(map[string]string)
	for k, v := range b.Equip {
		relist := map[string]interface{}{
			"价格": v.Price,
			"i":  num,
			"商品": k,
			"介绍": v.Usage,
		}
		msg += FormateMsg(b.Frame, relist, b.TitleWord["商店模板"])
		list[strconv.Itoa(num)] = k
		num++
	}
	Sayto(b.Frame, name, msg)
	b.Frame.GetGameControl().SetOnParamMsg(name, func(chat *defines.GameChat) (catch bool) {
		if len(chat.Msg) > 0 {
			if equipName, isok := list[chat.Msg[0]]; isok {
				go func() {

					cmd := fmt.Sprintf("scoreboard players remove @a[name=\"%v\",scores={%v=%v..}] %v %v", name, b.Score, b.Equip[equipName].Price, b.Score, b.Equip[equipName].Price)

					b.Frame.GetGameControl().SendCmdAndInvokeOnResponse(cmd, func(output *packet.CommandOutput) {
						if output.SuccessCount > 0 {
							b.Frame.GetGameControl().SendCmd(cmd)
							if b.DataOfProduction[name] == nil {
								b.DataOfProduction[name] = make(map[string]string)
							}
							b.DataOfProduction[name][equipName] = "这是一个装备"
							pterm.Info.Println("成功")
							Sayto(b.Frame, name, b.TitleWord["购买成功提示词"])
						} else {
							pterm.Info.Println("余额不足")
							Sayto(b.Frame, name, b.TitleWord["余额不足"])
						}
					})
					//cmd := fmt.Sprintf("scoreboard players remove @a[name=\"%v\"] %v %v", name, b.Score, b.Equip[equipName].Price)

				}()
			} else {
				pterm.Info.Println("无效输入")
				Sayto(b.Frame, name, b.TitleWord["无效输入"])
			}
		}

		return true
	})

}

// 仓库
func (b *Production) WareHouse(name string) {

	if data, isok := b.DataOfProduction[name]; isok {
		num := 0
		msg := ""
		list := make(map[string]string)
		for k, _ := range data {
			relist := map[string]interface{}{
				"i":  num,
				"装备": k,
			}
			list[strconv.Itoa(num)] = k
			msg += FormateMsg(b.Frame, relist, b.TitleWord["仓库模板"])
			num++
		}
		Sayto(b.Frame, name, msg)
		b.Frame.GetGameControl().SetOnParamMsg(name, func(chat *defines.GameChat) (catch bool) {
			if len(chat.Msg) > 0 {
				if equipName, ok := list[chat.Msg[0]]; ok {
					relist := map[string]interface{}{
						"装置": equipName,
						"介绍": b.Equip[equipName].Usage,
					}
					Sayto(b.Frame, name, FormateMsg(b.Frame, relist, b.Menu["装置详细界面菜单"]))
					b.Frame.GetGameControl().SetOnParamMsg(name, func(chat *defines.GameChat) (catch bool) {
						if len(chat.Msg) > 0 {
							if chat.Msg[0] == "0" {
								delete(b.DataOfProduction[name], equipName) //b.DataOfProduction[name]
								Sayto(b.Frame, name, b.TitleWord["删除成功"])
							}
						}

						return true
					})
				} else {
					Sayto(b.Frame, name, b.TitleWord["无效输入"])
				}
			}

			return true
		})
	} else {
		Sayto(b.Frame, name, b.TitleWord["暂时没有装备"])
	}
}
