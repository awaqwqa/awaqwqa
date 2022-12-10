package yscore

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"phoenixbuilder/minecraft/protocol"
	"phoenixbuilder/minecraft/protocol/packet"
	"phoenixbuilder/omega/defines"
	"strconv"
	"time"

	"github.com/pterm/pterm"
)

type Lottery struct {
	*defines.BasicComponent
	//触发词
	Triggers []string `json:"触发词"`
	//描述
	Usage string `json:"描述"`
	//菜单显示
	Menu map[string]string `json:"菜单显示"`
	//提示词
	TitleWord map[string]string `json:"提示词"`
	//所用计分板
	Score map[string]string `json:"所用计分板"`
	//奖池列表
	Pool     map[string]*pools    `json:"奖池"`
	GodsPull map[string]*GodsPull `json:"神之拉货"`
	Data     map[string]*Datas
}
type Datas struct {
	//抽奖累计次数（总）
	DrawsCumulativelyNum int
	//奖池累计次数
	LotteryNum int
	//是否第一次保底
	IsGuarantees bool
}

// 神之拉货
type GodsPull struct {
	//结构方块名字
	StruceName string `json:"结构方块名字"`
	//宣传价值
	Value int `json:"宣传价值"`
	//价格
	Price int `json:"每百分之一价格"`
	//真实提升
	ReUp float64 `json:"真实提升"`
	//购买指令
	Cmds []string `json:"购买执行指令"`
}

// 奖池
type pools struct {
	//最大随机数
	MaxRandomNum int `json:"最大随机数"`
	//单发价格
	Price int `json:"奖池一抽价格"`
	//奖池保底设置
	Guarantees *Guarantees `json:"保底设置"`
	//奖品名单
	Prize map[string]*Prize `json:"奖品"`
	//简介
	BriefIntroduction string `json:"介绍"`
}

// 奖品
type Prize struct {
	//奖品名字
	PrizeName string `json:"奖品名字"`
	//简介
	BriefIntroduction string `json:"简介"`
	//虚假概率
	FalseProbability string `json:"虚假概率"`
	//随机数起始
	RandomStar int `json:"随机数起始"`
	//随机数扩展
	RandomRange int `json:"随机数扩展"`
	//抽奖时执行的指令
	Cmds []string `json:"抽奖时执行指令"`
}

// 保底设置
type Guarantees struct {
	//保底数
	GuaranteesNum int `json:"保底数"`
	//第一次保底奖池
	FPool []string `json:"第一次保底奖池"`
	//第二次保底奖品
	SPrize string `json:"第二次保底奖品"`
	//保底执行指令

	//全服通报
	Say string `json:"保底时的全服通报"`
	//奖品闪烁指令
	Cmd string `json:"抽奖闪烁指令"`
}

func (o *Lottery) Init(cfg *defines.ComponentConfig) {
	marshal, _ := json.Marshal(cfg.Configs)
	if err := json.Unmarshal(marshal, o); err != nil {
		panic(err)
	}
	//fmt.Println(o.Pool["奖池名字"].Guarantees.FPool)
}

// 写这个的时候温柔已经被gank拉！！！
// 匆忙开始把功能写完
func (o *Lottery) Inject(frame defines.MainFrame) {
	o.Frame = frame
	o.BasicComponent.Inject(frame)
	o.Frame.GetGameListener().SetGameMenuEntry(&defines.GameMenuEntry{
		MenuEntry: defines.MenuEntry{
			Triggers:     o.Triggers,
			ArgumentHint: "",
			FinalTrigger: false,
			Usage:        "",
		},
		OptionalOnTriggerFn: o.onMenu,
	})

	if o.Data == nil {
		o.Data = make(map[string]*Datas)
	}
	CreateNameHash(o.Frame)
	o.InitData()

	o.Frame.GetJsonData("yscore抽奖数据.json", &o.Data)

	o.Listener.AppendLoginInfoCallback(o.onLogin)
}

// 登录时初始化分数
func (b *Lottery) onLogin(entry protocol.PlayerListEntry) {
	if _, ok := b.Data[entry.Username]; !ok {
		pterm.Info.Println("初始化成功", entry.Username)
		b.Data[entry.Username] = &Datas{
			DrawsCumulativelyNum: 0,
			LotteryNum:           0,
			IsGuarantees:         false,
		}
	}
}

// 机器人启动时初始化分数初始化分数
func (b *Lottery) InitData() {
	go func() {

		list := <-GetPlayerName(b.Frame, "@a")
		for _, v := range list {

			if _, ok := b.Data[v]; !ok {
				b.Data[v] = &Datas{
					DrawsCumulativelyNum: 0,
					LotteryNum:           0,
					IsGuarantees:         false,
				}
			}
		}
	}()
}
func (b *Lottery) onMenu(chat *defines.GameChat) (stop bool) {
	Sayto(b.Frame, chat.Name, b.Menu["主菜单显示"])
	b.Frame.GetGameControl().SetOnParamMsg(chat.Name, func(chat *defines.GameChat) (catch bool) {
		if len(chat.Msg) > 0 {
			switch chat.Msg[0] {
			case "0":
				b.LotteryingMenu(chat.Name)
			case "1":
				b.GodsPullMenu(chat.Name)
			}
		}

		return true
	})
	return true
}

func (o *Lottery) Activate() {

}

// 保存数据
func (b *Lottery) Signal(signal int) error {
	switch signal {
	case defines.SIGNAL_DATA_CHECKPOINT:
		return b.Frame.WriteJsonDataWithTMP("yscore抽奖数据.json", ".ckpt", &b.Data)
	}
	return nil
}

func (b *Lottery) GetGodPullDeatil(name string, prizeName string) {
	relist := map[string]interface{}{
		"货品名": prizeName,
		"价值":  b.GodsPull[prizeName].Value,
		"价格":  b.GodsPull[prizeName].Price,
	}
	Sayto(b.Frame, name, FormateMsg(b.Frame, relist, b.Menu["神之拉货详细界面"]))
	b.Frame.GetGameControl().SetOnParamMsg(name, func(chat *defines.GameChat) (catch bool) {
		if len(chat.Msg) > 0 {
			if CheckIsNum(chat.Msg[0]) {
				theProbability, _ := strconv.Atoi(chat.Msg[0])
				if theProbability > 0 && theProbability <= 100 {
					thePrice := theProbability * b.GodsPull[prizeName].Price
					cmd := fmt.Sprintf("scoreboard players remove @a[name=\"%v\",scores={%v=%v..}] %v %v", name, b.Score["购买所需计分板"], thePrice, b.Score["购买所需计分板"], thePrice)
					b.Frame.GetGameControl().SendCmdAndInvokeOnResponse(cmd, func(output *packet.CommandOutput) {
						if output.SuccessCount > 0 {
							rand.Seed(time.Now().Unix())
							randomNum := rand.Intn(101)
							pterm.Info.Println("真实概率", int(float64(theProbability)*b.GodsPull[prizeName].ReUp))
							theReProbability := int(float64(theProbability) * b.GodsPull[prizeName].ReUp)
							if theProbability == 100 {
								theReProbability = 100
							}
							if randomNum <= theReProbability {
								replaceList := map[string]interface{}{
									"player": name,
									"结构方块名":  b.GodsPull[prizeName].StruceName,
									"物品名字":   prizeName,
								}
								fmt.Println("..............")
								for _, v := range b.GodsPull[prizeName].Cmds {
									b.CmdSender(FormateMsg(b.Frame, replaceList, v))
								}
							} else {
								Sayto(b.Frame, name, b.TitleWord["神之拉货未中提示"])
							}
						} else {
							Sayto(b.Frame, name, "余额不足")
							fmt.Println("指令执行错误:", output.OutputMessages, "\n错误指令为:", cmd)
						}
					})
				} else {
					Sayto(b.Frame, name, "必须为大于0小于100的数字")
				}
			} else {
				Sayto(b.Frame, name, "请输入有效数字")
			}
		}

		return true
	})
}

// 神之拉货菜单
func (b *Lottery) GodsPullMenu(name string) {
	msg := ""
	num := 0
	list := make(map[string]string)
	for k, _ := range b.GodsPull {
		relist := map[string]interface{}{
			"i":     num,
			"拉货物品名": k,
		}
		list[strconv.Itoa(num)] = k
		msg += FormateMsg(b.Frame, relist, b.Menu["神之拉货菜单模板"]) + "\n"
		num++
	}
	Sayto(b.Frame, name, msg)
	b.Frame.GetGameControl().SetOnParamMsg(name, func(chat *defines.GameChat) (catch bool) {
		if len(chat.Msg) > 0 {
			if prizeName, ok := list[chat.Msg[0]]; ok {
				b.GetGodPullDeatil(name, prizeName)
			} else {
				Sayto(b.Frame, name, "输入有效数字")
			}
		}

		return true
	})
}

// 抽奖菜单
func (b *Lottery) LotteryingMenu(name string) {
	num := 0
	list := make(map[string]string)
	msg := ""
	for k, _ := range b.Pool {
		//奖池名字
		PoolName := k
		relist := map[string]interface{}{
			"i":    num,
			"奖池名字": PoolName,
		}
		list[strconv.Itoa(num)] = k
		msg += FormateMsg(b.Frame, relist, b.Menu["抽奖模板"]) + "\n"

		num++
	}
	Sayto(b.Frame, name, msg)
	b.Frame.GetGameControl().SetOnParamMsg(name, func(chat *defines.GameChat) (catch bool) {
		if len(chat.Msg) > 0 {
			if poolname, ok := list[chat.Msg[0]]; ok {
				b.GetBackLotteryData(name, poolname)
			} else {
				Sayto(b.Frame, name, "输入有效数字")
			}
		}

		return true
	})

}
func (b *Lottery) GetBackLotteryData(name string, PoolName string) {
	relist := map[string]interface{}{
		"奖池名字": PoolName,
		"简介":   b.Pool[PoolName].BriefIntroduction,
	}
	Sayto(b.Frame, name, FormateMsg(b.Frame, relist, b.Menu["奖池信息模板"]))
	b.Frame.GetGameControl().SetOnParamMsg(name, func(chat *defines.GameChat) (catch bool) {
		if len(chat.Msg) > 0 {
			switch chat.Msg[0] {
			case "0":
				b.Lotteryer(name, PoolName)
			}
		}

		return true
	})
}

// 检查该物品是否为保底物品
func (b *Lottery) CheckIsGuarantees(poolname string, prizeName string) int {
	if prizeName == b.Pool[poolname].Guarantees.SPrize {
		return 2
	}
	for _, v := range b.Pool[poolname].Guarantees.FPool {
		if v == prizeName {
			return 1
		}
	}
	return 0
}
func (b *Lottery) LotteryTen(name string, poolName string) bool {
	replaceList := map[string]interface{}{
		"奖池名字":   poolName,
		"玩家累计次数": b.Data[name].LotteryNum,
	}

	Sayto(b.Frame, name, FormateMsg(b.Frame, replaceList, b.TitleWord["抽奖头部"]))
	num := 0
	msg := ""
	for {

		if num >= 10 {
			msg += "\n"
			Sayto(b.Frame, name, msg)
			Sayto(b.Frame, name, b.TitleWord["抽奖尾部"])
			return true
		}
		rand.Seed(time.Now().Unix() + int64(num)) // unix 时间戳，秒
		//设置随机数
		randomNum := rand.Intn(b.Pool[poolName].MaxRandomNum)
		fmt.Println("随机数", randomNum)
		if prizeName, isok := b.CheckNumIsPrizeNum(randomNum, poolName); isok {
			//替换保底
			if b.Data[name].LotteryNum == b.Pool[poolName].Guarantees.GuaranteesNum && !b.Data[name].IsGuarantees {
				//归零
				b.Data[name].LotteryNum = 0
				b.Data[name].IsGuarantees = true

				prizeName = b.FindPrizeNameByStrucName(b.Pool[poolName].Guarantees.FPool[rand.Intn(len(b.Pool[poolName].Guarantees.FPool))])
			} else if b.Data[name].LotteryNum == b.Pool[poolName].Guarantees.GuaranteesNum && b.Data[name].IsGuarantees {
				b.Data[name].LotteryNum = 0
				b.Data[name].IsGuarantees = false
				prizeName = b.FindPrizeNameByStrucName(b.Pool[poolName].Guarantees.SPrize)
			}

			for _, v := range b.Pool[poolName].Prize[b.FindStrucNameByPrizeName(prizeName)].Cmds {
				replaceList = map[string]interface{}{
					"player":   name,
					"奖品名字":     prizeName,
					"奖品结构方块名字": b.FindStrucNameByPrizeName(prizeName),
				}
				b.CmdSender(FormateMsg(b.Frame, replaceList, v))

			}
			if num := b.CheckIsGuarantees(poolName, prizeName); num == 1 {
				b.Data[name].LotteryNum = 0
				b.Data[name].IsGuarantees = true
			} else if num == 2 {
				b.Data[name].IsGuarantees = false
			} else {
				b.Data[name].LotteryNum++

			}
			b.Data[name].DrawsCumulativelyNum++
			replaceList = map[string]interface{}{
				"奖品": prizeName,
			}
			msg += FormateMsg(b.Frame, replaceList, b.TitleWord["奖品部分"])
			//b.LotteryAnimation(name, prizeName, poolName)
		} else {
			Sayto(b.Frame, name, "无法随机到任何奖品 \n请喊管理员检查抽奖配置文件")
		}

		num++
	}
	return false

}
func (b *Lottery) Lotteryer(name string, poolName string) {
	cmd := fmt.Sprintf("scoreboard players remove @a[name=\"%v\",scores={%v=%v..}] %v %v", name, b.Score["购买所需计分板"], b.Pool[poolName].Price, b.Score["购买所需计分板"], b.Pool[poolName].Price)
	b.Frame.GetGameControl().SendCmdAndInvokeOnResponse(cmd, func(output *packet.CommandOutput) {
		if output.SuccessCount > 0 {
			Sayto(b.Frame, name, b.Menu["抽奖菜单"])
			b.Frame.GetGameControl().SetOnParamMsg(name, func(chat *defines.GameChat) (catch bool) {
				if len(chat.Msg) > 0 {
					switch chat.Msg[0] {
					case "0":
						b.LotteryOne(name, poolName, 1)
					case "1":
						b.LotteryTen(name, poolName)
					}

				}
				return true
			})

		} else {
			pterm.Info.Println("执行错误\n   ", cmd)
			Sayto(b.Frame, name, "余额不足")
		}
	})

}

// 根据结构名字奖品名字
func (b *Lottery) FindPrizeNameByStrucName(strucname string) string {
	for _, v := range b.Pool {
		for i, j := range v.Prize {
			if i == strucname {
				return j.PrizeName
			}
		}
	}
	return ""
}

// 根据奖品名字找结构名字
func (b *Lottery) FindStrucNameByPrizeName(prizeName string) string {
	for _, v := range b.Pool {
		for i, j := range v.Prize {
			if j.PrizeName == prizeName {
				return i
			}
		}
	}
	return ""
}

// 抽奖 返回值只是方便停止
func (b *Lottery) LotteryOne(name string, poolName string, LotteryNum int) bool {
	//Sayto(b.Frame, name, b.TitleWord["抽奖头部"])
	num := 0
	for {
		if num >= LotteryNum {
			//Sayto(b.Frame, name, b.TitleWord["抽奖尾部"])
			return true
		}
		rand.Seed(time.Now().Unix()) // unix 时间戳，秒
		//设置随机数
		randomNum := rand.Intn(b.Pool[poolName].MaxRandomNum)

		if prizeName, isok := b.CheckNumIsPrizeNum(randomNum, poolName); isok {
			//替换保底
			if b.Data[name].LotteryNum == b.Pool[poolName].Guarantees.GuaranteesNum && !b.Data[name].IsGuarantees {
				//归零
				b.Data[name].LotteryNum = 0
				b.Data[name].IsGuarantees = true
				prizeName = b.FindPrizeNameByStrucName(b.Pool[poolName].Guarantees.FPool[rand.Intn(len(b.Pool[poolName].Guarantees.FPool)-1)])
			} else if b.Data[name].LotteryNum == b.Pool[poolName].Guarantees.GuaranteesNum && b.Data[name].IsGuarantees {
				b.Data[name].LotteryNum = 0
				b.Data[name].IsGuarantees = false
				prizeName = b.FindPrizeNameByStrucName(b.Pool[poolName].Guarantees.SPrize)
			}
			if num := b.CheckIsGuarantees(poolName, prizeName); num == 1 {
				b.Data[name].LotteryNum = 0
				b.Data[name].IsGuarantees = true
			} else if num == 2 {
				b.Data[name].IsGuarantees = false
			} else {
				b.Data[name].LotteryNum++
			}
			b.Data[name].DrawsCumulativelyNum++
			b.LotteryAnimation(name, prizeName, poolName)
		} else {
			Sayto(b.Frame, name, "无法随机到任何奖品 \n请喊管理员检查抽奖配置文件")
		}

		num++
	}
	return false
}

// 命令发送者
func (b *Lottery) CmdSender(str string) {
	b.Frame.GetGameControl().SendCmdAndInvokeOnResponse(str, func(output *packet.CommandOutput) {
		if output.SuccessCount > 0 {

		} else {
			pterm.Info.Printfln("指令错误 错误信息为:%v\n错误指令为:%v", output.OutputMessages, str)
		}
	})

}

// 播放动画并给东西
func (b *Lottery) LotteryAnimation(name string, prizeName string, poolName string) {
	num := 50
	go func() {

		//用于循环
		ThePrizeList := []string{}
		for _, v := range b.Pool[poolName].Prize {
			ThePrizeList = append(ThePrizeList, v.PrizeName)
		}
		for {
			time.Sleep(time.Millisecond * time.Duration(num))
			//如果时间到了切换显示
			relist := make(map[string]interface{}, 2)
			//判断时间
			if num >= 600 {
				relist = map[string]interface{}{
					"player": name,
					"奖品名字":   prizeName,
					"奖品结构名字": b.FindStrucNameByPrizeName(prizeName),
				}
				//发送最后一次
				b.CmdSender(FormateMsg(b.Frame, relist, b.Pool[poolName].Guarantees.Cmd))
				//给予物品
				//b.CmdSender(fmt.Sprintf("execute @a[name=\"%v\"] ~~~ structure load %v ~~~~", name, b.FindStrucNameByStrucName(prizeName)))
				//执行指令
				strucName := b.FindStrucNameByPrizeName(prizeName)
				fmt.Println("找到指定的指令:", b.Pool[poolName].Prize[strucName].Cmds)
				for _, v := range b.Pool[poolName].Prize[strucName].Cmds {
					b.CmdSender(FormateMsg(b.Frame, relist, v))
				}

				break
			} else {
				rand.Seed(time.Now().Unix())
				relist = map[string]interface{}{
					"player": name,
					"奖品名字":   ThePrizeList[rand.Intn(len(ThePrizeList))],
				}
			}
			b.CmdSender(FormateMsg(b.Frame, relist, b.Pool[poolName].Guarantees.Cmd))
			time.Sleep(time.Millisecond * 50)
			//减少数字
			num += 20
		}

	}()

}

// 对应数字返回对应的奖品名字
func (b *Lottery) CheckNumIsPrizeNum(num int, poolNum string) (prizeName string, isok bool) {
	for _, v := range b.Pool[poolNum].Prize {
		if v.RandomStar <= num && num < (v.RandomRange+v.RandomStar) {
			return v.PrizeName, true
		}
	}
	return "", false

}
