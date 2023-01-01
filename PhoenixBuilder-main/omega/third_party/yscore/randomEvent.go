package yscore

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"phoenixbuilder/minecraft/protocol/packet"
	"phoenixbuilder/omega/defines"
	"regexp"
	"strconv"
	"time"

	"github.com/pterm/pterm"
)

type RandomEvent struct {
	*defines.BasicComponent
	EventPool map[string]*Event `json:"新事件群"`
	//正在活动中的事件
	ActiveEventPool map[string]*AcEvent
	//冷却事件池子
	ColdEvent                 map[string]*CoEvent
	BiologicalComparisonTable map[string]string
}

// 活动事件的信息
type AcEvent struct {
	//在内玩家
	PlayerList map[string]string
	//在内怪物
	MonsterList map[string]string
	TotolScore  int
	//事件是否完全完成注册
	isOk bool
}
type CoEvent struct {
	ColdTime int64
}

// 事件
type Event struct {
	TriggerMechanism      *TriggerData        `json:"触发机制"`
	Cmds                  map[string][]string `json:"需要执行的指令"`
	CheckMonsterMechanism map[string]string   `json:"检查怪物离开范围机制"`
	PrizePool             map[string]int      `json:"奖池机制"`
	SleepMechanism        map[string]int      `json:"冷却机制"`
	Words                 map[string]string   `json:"提示话语"`
}
type TriggerData struct {
	StarPos   []int `json:"初始坐标"`
	ExpandPos []int `json:"延长范围"`
}

func (o *RandomEvent) Init(cfg *defines.ComponentConfig) {
	marshal, _ := json.Marshal(cfg.Configs)
	if err := json.Unmarshal(marshal, o); err != nil {
		panic(err)
	}
	//初始化activeEventPool
	o.ActiveEventPool = make(map[string]*AcEvent)
	o.ColdEvent = make(map[string]*CoEvent)
	o.BiologicalComparisonTable = make(map[string]string)
	//更新
	if cfg.Version == "0.0.1" {
		eventData := map[string]interface{}{
			"事件一号": &Event{
				TriggerMechanism: &TriggerData{
					StarPos:   []int{63, -60, 44},
					ExpandPos: []int{5, 7, 4},
				},
				Cmds: map[string][]string{
					"事件触发执行指令(生成结构体前)":    []string{"tp @a[name=\"[机器人名字]\"] [x] [y] [z]"},
					"事件触发时执行指令(生成结构体后)":   []string{"tag @e[x=[x],y=[y],z=[z],dx=[dx],dy=[dy],dz=[dz],type=!player] add 怪物"},
					"事件结束后执行指令":           []string{"tell @a[name=\"[player]\"] 恭喜完成事件一号", "scoreboard players add @a[name=\"[player]\"] 挖掘 [获得积分分数]", "tp @a[x=[x],y=[y],z=[z],dx=[dx],dy=[dy],dz=[dz]] 69 -59 49"},
					"事件触发时执行指令(生成结构体的指令)": []string{"execute @a[name=\"[player]\"] ~~~ structure load clear ~~~"},
					"对应tag的怪物出了范围执行指令":    []string{"tp @e[name=[怪物名字]] 66 -59 46"},
					"玩家出了范围执行指令":          []string{"tp @a[name=\"[player]\"] 66 -59 46"},
				},
				CheckMonsterMechanism: map[string]string{
					"怪物tag": "怪物",
				},
				PrizePool: map[string]int{
					"单次增加总奖池值":            10,
					"生成结构体指令执行最高次数(1-10)": 1,
				},
				SleepMechanism: map[string]int{
					"冷却时间最小值(秒)":   500,
					"增加冷却时间最大值(秒)": 100,
				},
				Words: map[string]string{
					"事件结束提示":    "事件[事件名字] 已经结束 冷却时间为:[冷却时间]",
					"事件启动时提示":   "[事件名字] 事件已经开启 坐标为:[x] [y] [z] --- [dx] [dy] [dz]",
					"刷新提示":      "事件总刷新数为:[刷新次数] 总分值为:[分值]",
					"玩家逃出范围后提示": "你已经出了事件的范围了",
					"有新玩家加入事件":  "[player] 加入事件[事件名字] 当前事件总人数为:[总人数]",
				},
			},
		}
		cfg.Configs["事件群"] = nil
		cfg.Configs["新事件群"] = eventData
		cfg.Version = "0.0.2"
		cfg.Upgrade()
	}
}
func (o *RandomEvent) Inject(frame defines.MainFrame) {
	o.Frame = frame
	o.BasicComponent.Inject(frame)
	CreateNameHash(o.Frame)
	//同步让生物的名字与对应的uuid同步
	o.Frame.GetGameListener().SetOnAnyPacketCallBack(func(p packet.Packet) {
		if p.ID() == 13 {

			id := strconv.Itoa(int(p.(*packet.AddActor).EntityUniqueID))
			if name, ok := p.(*packet.AddActor).EntityMetadata[4]; ok {

				if v, ok := name.(string); ok {
					o.BiologicalComparisonTable[id] = v
				}
				//pterm.Info.Println(o.BiologicalComparisonTable)

			}

		}
	})
}

func (b *RandomEvent) Activate() {

	for {
		time.Sleep(time.Second * 1)
		go func() {
			playerPos := <-GetPos(b.Frame, "@a")

			//首先是获取全部人的坐标 然后检测是否应该激活事件 -->tp所有非死亡且离开用户进入事件
			//-->然后tp所有离开的怪物进入事件-->检查事件的人数与怪物数量-->如果剩余人数或者怪物为0时杀死事件
			//--》将杀死了的事件进入冷却池子 然后标明应该等待的刻度时间<秒>

			//首先是注册事件
			b.AddEvent(playerPos)
			//再来是检查玩家是否有跑出事件 死亡则删除 没有则传送返回
			b.CheckPlayer(playerPos)
			b.CheckMonster()

		}()
	}
}

// 检查怪物是否在事件内部
func (b *RandomEvent) CheckMonster() {

	go func() {
		monsterPos := <-GetPos(b.Frame, "@e[type=!player]")
		for monsterName, pos := range monsterPos {
			//检查是否为玩家
			if match, _ := regexp.MatchString("^-", monsterName); match {
				for k, v := range b.ActiveEventPool {
					if !v.isOk {
						continue
					}
					data := b.EventPool[k]
					//范围外就tp 回来
					if _, isIn := v.MonsterList[monsterName]; isIn && !(pos[0] >= data.TriggerMechanism.StarPos[0] && pos[1] >= data.TriggerMechanism.StarPos[1] && pos[2] >= data.TriggerMechanism.StarPos[2] && pos[0] <= (data.TriggerMechanism.StarPos[0]+data.TriggerMechanism.ExpandPos[0]) && pos[1] <= (data.TriggerMechanism.StarPos[1]+data.TriggerMechanism.ExpandPos[1]) && pos[2] <= (data.TriggerMechanism.StarPos[2]+data.TriggerMechanism.ExpandPos[2])) {
						if name, ok := b.BiologicalComparisonTable[monsterName]; ok {
							monsterName = name
						}
						relist := map[string]interface{}{
							"怪物名字": monsterName,
						}
						for _, cmd := range b.EventPool[k].Cmds["对应tag的怪物出了范围执行指令"] {
							_cmd := FormateMsg(b.Frame, relist, cmd)
							b.Frame.GetGameControl().SendCmdAndInvokeOnResponse(_cmd, func(output *packet.CommandOutput) {
								if output.SuccessCount <= 0 {
									pterm.Info.Printfln("传送失败 失败原因:%v", output.OutputMessages)
								}
							})
						}
					}
				}
			}
		}
		//检查怪物是否死亡
		for k, v := range b.ActiveEventPool {
			if !v.isOk {
				continue
			}
			if len(v.MonsterList) == 0 {
				b.delectEvent(k)
			}
			for monsterName, _ := range v.MonsterList {
				if _, isIn := monsterPos[monsterName]; !isIn {
					delete(b.ActiveEventPool[k].MonsterList, monsterName)
					if len(b.ActiveEventPool[k].MonsterList) == 0 {
						b.delectEvent(k)
					}
				}
			}
		}
	}()

}

// 检查玩家是否在范围内
func (b *RandomEvent) CheckPlayer(playerPos map[string][]int) {
	for k, v := range b.ActiveEventPool {
		if !v.isOk {
			continue
		}
		data := b.EventPool[k]
		for playerName, _ := range v.PlayerList {
			if pos, ok := playerPos[playerName]; ok {
				if len(pos) != 3 {
					return
				}
				if !(pos[0] >= data.TriggerMechanism.StarPos[0] && pos[1] >= data.TriggerMechanism.StarPos[1] && pos[2] >= data.TriggerMechanism.StarPos[2] && pos[0] <= (data.TriggerMechanism.ExpandPos[0]+data.TriggerMechanism.StarPos[0]) && pos[1] <= (data.TriggerMechanism.ExpandPos[1]+data.TriggerMechanism.StarPos[1]) && pos[2] <= (data.TriggerMechanism.ExpandPos[2]+data.TriggerMechanism.StarPos[2])) {
					relist := map[string]interface{}{
						"player": playerName,
					}
					for _, cmd := range b.EventPool[k].Cmds["玩家出了范围执行指令"] {
						cmd = FormateMsg(b.Frame, relist, cmd)
						b.Frame.GetGameControl().SendCmdAndInvokeOnResponse(cmd, func(output *packet.CommandOutput) {
							if !(output.SuccessCount > 0) {
								pterm.Info.Printfln("指令%v执行失败 错误原因是%v", cmd, output.OutputMessages)
							}
						})
					}
					Sayto(b.Frame, playerName, data.Words["玩家逃出范围后提示"])
				}
			} else {
				//应该是死亡 或者不在线了
				pterm.Info.Println(playerName, "玩家应该是死亡或者不在线了 已从事件中删除")
				delete(b.ActiveEventPool[k].PlayerList, playerName)
				//如果所有人死亡则删除事件
				if len(b.ActiveEventPool[k].PlayerList) == 0 {
					b.delectEvent(k)
				}
			}

		}
	}
}

// 删除事件
func (b *RandomEvent) delectEvent(eventName string) {
	if data, ok := b.ActiveEventPool[eventName]; !ok {
		return
	} else if !data.isOk {
		//防止太早删除
		return
	}
	//奖励环节
	if len(b.ActiveEventPool[eventName].PlayerList) >= 1 {
		for playerName, _ := range b.ActiveEventPool[eventName].PlayerList {

			for _, cmd := range b.EventPool[eventName].Cmds["事件结束后执行指令"] {
				relist := map[string]interface{}{
					"player": playerName,
					"获得积分分数": int(b.ActiveEventPool[eventName].TotolScore / len(b.ActiveEventPool[eventName].PlayerList)),
					"x":      b.EventPool[eventName].TriggerMechanism.StarPos[0],
					"y":      b.EventPool[eventName].TriggerMechanism.StarPos[1],
					"z":      b.EventPool[eventName].TriggerMechanism.StarPos[2],
					"dx":     b.EventPool[eventName].TriggerMechanism.ExpandPos[0],
					"dy":     b.EventPool[eventName].TriggerMechanism.ExpandPos[1],
					"dz":     b.EventPool[eventName].TriggerMechanism.ExpandPos[2],
				}
				_cmd := FormateMsg(b.Frame, relist, cmd)
				b.Frame.GetGameControl().SendCmdAndInvokeOnResponse(_cmd, func(output *packet.CommandOutput) {
					if output.SuccessCount > 0 {
						pterm.Info.Printfln("执行指令成功", _cmd)
					} else {
						pterm.Info.Printfln("执行指令%v失败 失败原因是%v", _cmd, output.OutputMessages)
					}
				})
			}
		}
	}
	waiteTime := b.EventPool[eventName].SleepMechanism["冷却时间最小值(秒)"] + rand.Intn(b.EventPool[eventName].SleepMechanism["增加冷却时间最大值(秒)"])
	relist := map[string]interface{}{
		"事件名字": eventName,
		"冷却时间": waiteTime,
	}
	b.Frame.GetGameControl().SayTo("@a", FormateMsg(b.Frame, relist, b.EventPool[eventName].Words["事件结束提示"]))
	//添加冷却时间
	if _, isok := b.ColdEvent[eventName]; !isok {
		b.ColdEvent[eventName] = &CoEvent{
			time.Now().Unix() + int64(waiteTime),
		}
	}
	b.ColdEvent[eventName].ColdTime = time.Now().Unix() + int64(waiteTime)
	pterm.Info.Println("事件进入冷却时间", b.ColdEvent[eventName].ColdTime)
	delete(b.ActiveEventPool, eventName)

}

// 添加事件
func (b *RandomEvent) AddEvent(playerPos map[string][]int) {
	//检查所有人的坐标
	for k, v := range playerPos {
		//如果坐标正常继续检测
		//且不是机器人的坐标
		if k == b.Frame.GetUQHolder().GetBotName() {
			continue
		}
		if len(v) >= 3 {
			//遍历所有的事件 查看触发情况
			for EventName, event := range b.EventPool {

				//判断是否在内部
				if b.CheckInEvent(event, v) {
					//检查事件是否激活
					if _, ok := b.ActiveEventPool[EventName]; ok {
						//判断是否玩家重合
						if _, isok := b.ActiveEventPool[EventName].PlayerList[k]; !isok {

							b.ActiveEventPool[EventName].PlayerList[k] = ""
							pterm.Info.Printfln("%v 事件 加入玩家 %v", EventName, k)
							relist := map[string]interface{}{
								"player": k,
								"事件名字":   EventName,
								"总人数":    len(b.ActiveEventPool[EventName].PlayerList),
							}
							b.Frame.GetGameControl().SayTo("@a", FormateMsg(b.Frame, relist, b.EventPool[EventName].Words["有新玩家加入事件"]))

						}
						//并非实体才能触发
					} else if !(b.checkCool(EventName)) {
						b.EventRegister(EventName, k)
					}

				}
			}
		}

	}
}

// 检查是否还在冷却
// 获取当前时间戳 看是否满足大于事件规定的时间戳
func (b *RandomEvent) checkCool(eventName string) bool {
	timeNum := time.Now().Unix()
	if data, ok := b.ColdEvent[eventName]; ok {
		if timeNum >= data.ColdTime {
			return false
		} else {
			return true
		}
	}
	return false
}

// 检查是否在里面
func (b *RandomEvent) CheckInEvent(event *Event, pos []int) bool {
	if len(pos) != 3 {
		pterm.Info.Println("玩家坐标无法识别", pos)
		return false
	}

	if len(event.TriggerMechanism.StarPos) != 3 || len(event.TriggerMechanism.ExpandPos) != 3 {
		pterm.Info.Println("事件配置中 坐标或者延长坐标 格式出现了错误")
		return false
	}
	if event.TriggerMechanism.StarPos[0] <= pos[0] && event.TriggerMechanism.StarPos[1] <= pos[1] && event.TriggerMechanism.StarPos[2] <= pos[2] && (event.TriggerMechanism.ExpandPos[0]+event.TriggerMechanism.StarPos[0]) >= pos[0] && (event.TriggerMechanism.ExpandPos[1]+event.TriggerMechanism.StarPos[1]) >= pos[1] && (event.TriggerMechanism.ExpandPos[2]+event.TriggerMechanism.StarPos[2]) >= pos[2] {
		return true
	}
	return false
}

// 注册事件 首先要传入首个进入的人员
func (b *RandomEvent) EventRegister(EventName string, name string) {

	//初始化人物
	b.ActiveEventPool[EventName] = &AcEvent{
		PlayerList: map[string]string{
			name: name,
		},
		MonsterList: map[string]string{},
		isOk:        false,
	}
	//发送语音提示
	msg := b.EventPool[EventName].Words["事件启动时提示"]
	if !(len(b.EventPool[EventName].TriggerMechanism.StarPos) >= 3 && len(b.EventPool[EventName].TriggerMechanism.ExpandPos) >= 3) {
		pterm.Info.Printfln("你的事件%v 配置文件中坐标修改错误 通常这个错误是因为你没有按照[x,y,z]的格式来填写", EventName)
		panic("")
	}
	relist := map[string]interface{}{
		"player": name,
		"事件名字":   EventName,
		"x":      b.EventPool[EventName].TriggerMechanism.StarPos[0],
		"y":      b.EventPool[EventName].TriggerMechanism.StarPos[1],
		"z":      b.EventPool[EventName].TriggerMechanism.StarPos[2],
		"dx":     b.EventPool[EventName].TriggerMechanism.ExpandPos[0],
		"dy":     b.EventPool[EventName].TriggerMechanism.ExpandPos[1],
		"dz":     b.EventPool[EventName].TriggerMechanism.ExpandPos[2],
	}
	msg = FormateMsg(b.Frame, relist, msg)
	b.Frame.GetGameControl().SayTo("@a", msg)
	//随机生成怪物
	func() {
		rand.Seed(time.Now().Unix())
		randNum := rand.Intn(b.EventPool[EventName].PrizePool["生成结构体指令执行最高次数(1-10)"]) + 1
		//保证不会超过10次
		if randNum >= 10 {
			randNum = 10
			pterm.Info.Println("检测到你配置中 生成结构体指令执行最高次数(1-10) 次数大于10 已经自动改为10")
		}
		pterm.Info.Println("事件触发成功 刷新次数为:", randNum)

		scoreRelist := map[string]interface{}{
			"刷新次数": randNum,
			"分值":   randNum * b.EventPool[EventName].PrizePool["单次增加总奖池值"],
		}
		//同步积分
		b.ActiveEventPool[EventName].TotolScore = randNum * b.EventPool[EventName].PrizePool["单次增加总奖池值"]
		b.Frame.GetGameControl().SayTo("@a", FormateMsg(b.Frame, scoreRelist, b.EventPool[EventName].Words["刷新提示"]))
	}()

	//生成结构前指令
	func(tpRelist map[string]interface{}) {
		tpRelist["机器人名字"] = b.Frame.GetUQHolder().GetBotName()
		for _, cmd := range b.EventPool[EventName].Cmds["事件触发执行指令(生成结构体前)"] {
			reCmd := FormateMsg(b.Frame, tpRelist, cmd)
			b.Frame.GetGameControl().SendCmdAndInvokeOnResponse(reCmd, func(output *packet.CommandOutput) {
				if !(output.SuccessCount > 0) {
					pterm.Info.Printfln("执行指令:%v 失败 失败原因是:%v", reCmd, output.OutputMessages)
				}
			})
		}
	}(relist)
	//生成结构
	func(summonRelist map[string]interface{}) {
		if data, ok := b.ActiveEventPool[EventName]; ok {
			loadNum := int(data.TotolScore / (b.EventPool[EventName].PrizePool["单次增加总奖池值"]))
			for i := 0; i < loadNum; i++ {
				for _, cmd := range b.EventPool[EventName].Cmds["事件触发时执行指令(生成结构体的指令)"] {
					reCmd := FormateMsg(b.Frame, summonRelist, cmd)
					b.Frame.GetGameControl().SendCmdAndInvokeOnResponse(reCmd, func(output *packet.CommandOutput) {
						if !(output.SuccessCount > 0) {
							pterm.Info.Printfln("执行指令:%v 失败 失败原因是:%v", reCmd, output.OutputMessages)
						} else {
							pterm.Info.Println("执行成功")
						}
					})
				}
			}
		}
	}(relist)
	//生成结构后指令
	func(afterSummonRelist map[string]interface{}) {
		//等待50毫秒确定已经完成前面的指令执行
		time.Sleep(time.Millisecond * 50)
		for _, cmd := range b.EventPool[EventName].Cmds["事件触发时执行指令(生成结构体后)"] {
			reCmd := FormateMsg(b.Frame, afterSummonRelist, cmd)
			b.Frame.GetGameControl().SendCmdAndInvokeOnResponse(reCmd, func(output *packet.CommandOutput) {
				if !(output.SuccessCount > 0) {
					pterm.Info.Printfln("执行指令:%v 失败 失败原因是:%v", reCmd, output.OutputMessages)
				}
			})
		}
	}(relist)
	//生成怪物名单
	go func() {
		tag := b.EventPool[EventName].CheckMonsterMechanism["怪物tag"]
		MonsterPos := <-GetPos(b.Frame, fmt.Sprintf("@e[tag=\"%v\",type=!player]", tag))
		for monsterName, _ := range MonsterPos {
			//如果有真名就替换
			if name, ok := b.BiologicalComparisonTable[monsterName]; ok {
				monsterName = name
			}
			b.ActiveEventPool[EventName].MonsterList[monsterName] = "这是一个怪物"
		}
		//如果怪物列表为空则不触发事件
		if len(b.ActiveEventPool[EventName].MonsterList) == 0 {
			pterm.Info.Println("触发事件失败 已经强行进入冷却期")
			b.Frame.GetGameControl().SayTo("@a", "事件触发失败 未检测到怪物 已经强行进入冷却期50秒")
			b.ColdEvent[EventName] = &CoEvent{
				ColdTime: time.Now().Unix() + int64(50),
			}
			delete(b.ActiveEventPool, EventName)
		} else {
			pterm.Info.Printfln("事件%v触发成功 且运行完成 怪物列表为%v", EventName, b.ActiveEventPool[EventName].MonsterList)
			b.ActiveEventPool[EventName].isOk = true
		}

	}()

}
