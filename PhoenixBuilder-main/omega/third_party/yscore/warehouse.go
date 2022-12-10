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

type WareHouse struct {
	*defines.BasicComponent
	Triggers   []string          `json:"触发词"`
	Usage      string            `json:"描述"`
	Menu       map[string]string `json:"菜单"`
	TitleWord  map[string]string `json:"提示词"`
	OpenTime   int               `json:"仓库每次开启的时间(秒)"`
	MaxNum     int               `json:"个人拥有仓库上限"`
	Price      int               `json:"每一个仓库价格"`
	Score      string            `json:"购买计分板"`
	CenterPos  []int             `json:"仓库坐标(范围1k格)"`
	Target     string            `json:"开启仓库选择器限制"`
	TheSavePos string            `json:"存入仓库时相对坐标"`
	//map[name]map[仓库名字]坐标
	DataOfWareHouse map[string]map[string]*WareHouseData
	//允许在范围内的人员
	TempWareHouse map[string]string
}
type WareHouseData struct {
	StrucName string
	Pos       []int
}

func (o *WareHouse) Init(cfg *defines.ComponentConfig) {
	marshal, _ := json.Marshal(cfg.Configs)
	if err := json.Unmarshal(marshal, o); err != nil {
		panic(err)
	}

	o.DataOfWareHouse = make(map[string]map[string]*WareHouseData)
}

func (o *WareHouse) Inject(frame defines.MainFrame) {
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
	o.Frame.GetJsonData("仓库.json", &o.DataOfWareHouse)
}

// 保存数据
func (b *WareHouse) Signal(signal int) error {
	switch signal {
	case defines.SIGNAL_DATA_CHECKPOINT:
		return b.Frame.WriteJsonDataWithTMP("仓库.json", ".ckpt", &b.DataOfWareHouse)
	}
	return nil
}

func (b *WareHouse) onMenu(chat *defines.GameChat) (stop bool) {
	Sayto(b.Frame, chat.Name, b.Menu["主菜单"])
	if b.DataOfWareHouse[chat.Name] == nil {
		b.DataOfWareHouse[chat.Name] = make(map[string]*WareHouseData)
	}
	b.Frame.GetGameControl().SetOnParamMsg(chat.Name, func(chat *defines.GameChat) (catch bool) {
		if len(chat.Msg) > 0 {
			switch chat.Msg[0] {
			case "0":
				b.BuyWareHouse(chat.Name)
			case "1":
				b.Deposit(chat.Name)
			case "2":
				b.Take(chat.Name)

			}
		}

		return true
	})
	return true
}

// 获取最新仓库坐标
func (b *WareHouse) getNewPos() []int {
	num := 0
	x := 0
	z := 0
	for _, v := range b.DataOfWareHouse {
		num += len(v)
	}
	for {
		if num <= 0 {

			return []int{
				b.CenterPos[0] + x,
				b.CenterPos[1],
				b.CenterPos[2] + z,
			}

		} else {
			x += 5
			if x >= 100 {
				z += 5
				x = 0
			}
			num--
		}
	}
	return []int{}
}

// 命令发送者
func (b *WareHouse) CmdSender(str string) {
	b.Frame.GetGameControl().SendCmdAndInvokeOnResponse(str, func(output *packet.CommandOutput) {
		if output.SuccessCount > 0 {

		} else {
			pterm.Info.Printfln("指令错误 错误信息为:%v\n错误指令为:%v", output.OutputMessages, str)
		}
	})

}
func (b *WareHouse) Take(name string) {
	if data, ok := b.DataOfWareHouse[name]; ok {
		num := 0
		list := make(map[string]string)
		msg := ""
		for k, _ := range data {
			relist := map[string]interface{}{
				"i":  num,
				"仓库": k,
			}
			msg += FormateMsg(b.Frame, relist, b.TitleWord["显示自己仓库模板"]) + "\n"
			list[strconv.Itoa(num)] = k
			num++
		}
		Sayto(b.Frame, name, msg)
		b.Frame.GetGameControl().SetOnParamMsg(name, func(chat *defines.GameChat) (catch bool) {
			if len(chat.Msg) > 0 {
				if woreName, ok := list[chat.Msg[0]]; ok {
					data := b.DataOfWareHouse[name][woreName]
					go func() {

						thePlayerPos := <-GetPos(b.Frame, "@a[name=\""+name+"\"]")
						b.CmdSender(fmt.Sprintf("tp @a[name=\"%v\"] %v %v %v", name, data.Pos[0], data.Pos[1], data.Pos[2]))
						b.CmdSender(fmt.Sprintf("fill %v %v %v %v %v %v quartz_block 0 hollow", data.Pos[0]-2, data.Pos[1]-2, data.Pos[2]-2, data.Pos[0]+2, data.Pos[1]+2, data.Pos[2]+2))
						b.CmdSender(fmt.Sprintf("structure load %v %v %v %v ", data.StrucName, data.Pos[0], data.Pos[1], data.Pos[2]))
						b.CmdSender(fmt.Sprintf("tp @a[name=\"%v\"] %v %v %v", name, data.Pos[0], data.Pos[1], data.Pos[2]))
						defer func() {
							cmd := fmt.Sprintf("structure save %v %v %v %v %v %v %v", data.StrucName, data.Pos[0], data.Pos[1], data.Pos[2], data.Pos[0], data.Pos[1], data.Pos[2])
							b.Frame.GetGameControl().SendCmdAndInvokeOnResponse(cmd, func(output *packet.CommandOutput) {
								if !(output.SuccessCount > 0) {
									cmd = fmt.Sprintf("tp @s %v %v %v", data.Pos[0], data.Pos[1], data.Pos[2])
								}
								b.CmdSender(fmt.Sprintf("fill %v %v %v %v %v %v air 0 ", data.Pos[0]-2, data.Pos[1]-2, data.Pos[2]-2, data.Pos[0]+2, data.Pos[1]+2, data.Pos[2]+2))
								b.CmdSender(fmt.Sprintf("tp @a[name=\"%v\"] %v %v %v", name, thePlayerPos[name][0], thePlayerPos[name][1], thePlayerPos[name][2]))

							})
						}()
						time.Sleep(time.Second * time.Duration(b.OpenTime))

					}()

				} else {
					Sayto(b.Frame, name, "请输入有效数字")
				}
			}

			return true
		})

	} else {
		Sayto(b.Frame, name, "你暂时没有仓库")
	}
}

// 购买仓库
func (b *WareHouse) BuyWareHouse(name string) {
	cmd := fmt.Sprintf("scoreboard players remove @a[name=\"%v\",scores={%v=%v..}] %v %v", name, b.Score, b.Price, b.Score, b.Price)
	b.Frame.GetGameControl().SendCmdAndInvokeOnResponse(cmd, func(output *packet.CommandOutput) {
		if output.SuccessCount > 0 {
			if len(b.DataOfWareHouse[name]) >= b.MaxNum {
				Sayto(b.Frame, name, b.TitleWord["仓库达到上限"])
			} else {
				Sayto(b.Frame, name, b.TitleWord["购买仓库提示词"])
				b.Frame.GetGameControl().SetOnParamMsg(name, func(chat *defines.GameChat) (catch bool) {
					if len(chat.Msg) > 0 {
						theWoreName := chat.Msg[0]
						Sayto(b.Frame, name, b.TitleWord["购买仓库成功"])
						if b.DataOfWareHouse[name] == nil {
							b.DataOfWareHouse[name] = map[string]*WareHouseData{
								theWoreName: {
									Pos:       b.getNewPos(),
									StrucName: "yscore" + theWoreName + "l",
								},
							}
						} else {
							b.DataOfWareHouse[name][theWoreName] = &WareHouseData{
								Pos:       b.getNewPos(),
								StrucName: "yscore" + theWoreName + "l",
							}
						}

					}

					return true
				})
			}
		} else {
			pterm.Info.Printfln("余额不足")
			Sayto(b.Frame, name, "余额不足")
		}
	})
}

// 存东西
func (b *WareHouse) Deposit(name string) {
	if data, ok := b.DataOfWareHouse[name]; ok {
		num := 0
		list := make(map[string]string)
		msg := ""
		for k, _ := range data {
			relist := map[string]interface{}{
				"i":  num,
				"仓库": k,
			}
			msg += FormateMsg(b.Frame, relist, b.TitleWord["显示自己仓库模板"]) + "\n"
			list[strconv.Itoa(num)] = k
			num++
		}
		pterm.Info.Println(msg)
		Sayto(b.Frame, name, msg)
		b.Frame.GetGameControl().SetOnParamMsg(name, func(chat *defines.GameChat) (catch bool) {
			if len(chat.Msg) > 0 {
				if woreName, ok := list[chat.Msg[0]]; ok {
					if _, isok := b.DataOfWareHouse[name][woreName]; isok {
						b.DataOfWareHouse[name][woreName] = &WareHouseData{
							Pos:       b.getNewPos(),
							StrucName: "yscore" + woreName,
						}
					}
					//data := b.DataOfWareHouse[name][woreName]
					go func() {
						pterm.Info.Println(b.Target)
						player := <-GetPlayerName(b.Frame, b.Target)
						pterm.Info.Println(player)
						theList := make(map[string]string)
						for k, v := range player {
							theList[v] = k
						}
						if _, isok := theList[name]; isok {
							b.CmdSender(fmt.Sprintf("execute @a[name=\"%v\"] ~~~ structure save %v %v ", name, b.DataOfWareHouse[name][woreName].StrucName, b.TheSavePos))
							Sayto(b.Frame, name, b.TitleWord["保存成功提示词"])
						} else {
							Sayto(b.Frame, name, b.TitleWord["不符合开启条件提示"])
						}
					}()
				} else {
					Sayto(b.Frame, name, "请输入有效数字")
				}
			}

			return true
		})

	} else {
		Sayto(b.Frame, name, "你暂时没有仓库")
	}
}

// 所有组件 Inject 之后，会调用 BeforeActivate，在这个函数里可以去寻找其他组件的接口了，因为注入接口的过程是在 Inject 中完成的
// func (o *EchoMiao) BeforeActivate() {

// }

// 这个函数会在一个单独的协程中运行，可以自由的 sleep 或者阻塞
func (o *WareHouse) Activate() {
	for {
		time.Sleep(time.Second * 5)
		o.Frame.GetGameControl().SendCmd(fmt.Sprintf("gamemode 2 @a[m=0,x=%v,y=%v,z=%v,r=1000]", o.CenterPos[0], o.CenterPos[1], o.CenterPos[2]))
	}
}
