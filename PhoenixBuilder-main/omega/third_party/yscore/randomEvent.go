package yscore

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"phoenixbuilder/minecraft/protocol/packet"
	"phoenixbuilder/omega/defines"
	"regexp"
	"time"

	"github.com/pterm/pterm"
)

type RandomEvent struct {
	*defines.BasicComponent
	EventPool map[string]*Event `json:"事件群"`
	//正在活动中的事件
	ActiveEventPool map[string]*AcEvent
	//冷却事件池子
	ColdEvent map[string]*CoEvent
}

// 活动事件的信息
type AcEvent struct {
	//在内玩家
	PlayerList map[string]string
	//在内怪物
	MonsterList map[string]string
	TotolScore  int
}
type CoEvent struct {
	ColdTime int64
}

// 事件
type Event struct {
	PrizeCmd   []string          `json:"事件结束奖励指令"`
	Position   Pos               `json:"范围"`
	WaiteTime  WaiteTime         `json:"冷却时间范围"`
	LoadMaxNum int               `json:"每次刷新结构随机刷新次数上限(1-10)"`
	StructName string            `json:"结构名字"`
	Words      map[string]string `json:"提示话语"`
	//单次刷新的总分值
	AdScore     int `json:"单次刷新总分值"`
	RanketScore string
}

// 坐标
type Pos struct {
	StartPosition  []int `json:"起点坐标"`
	ExpandPosition []int `json:"延长范围"`
	BackPos        []int `json:"事件结束后返回坐标"`
	TpBackPos      []int `json:"离开范围后返回坐标"`
}

// 等待时间
type WaiteTime struct {
	MinTime    int `json:"最小时间(秒)"`
	ExpandTime int `json:"最大增加"`
}

func (o *RandomEvent) Init(cfg *defines.ComponentConfig) {
	marshal, _ := json.Marshal(cfg.Configs)
	if err := json.Unmarshal(marshal, o); err != nil {
		panic(err)
	}
	//初始化activeEventPool
	o.ActiveEventPool = make(map[string]*AcEvent)
	o.ColdEvent = make(map[string]*CoEvent)
}
func (o *RandomEvent) Inject(frame defines.MainFrame) {
	o.Frame = frame
	o.BasicComponent.Inject(frame)
	CreateNameHash(o.Frame)
}

func (b *RandomEvent) Activate() {
	for {
		time.Sleep(time.Second * 1)
		go func() {
			playerPos := <-GetPos(b.Frame, "@e")

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
			//检查是否为怪兽
			if match, _ := regexp.MatchString("^-", monsterName); match {
				for k, v := range b.ActiveEventPool {
					data := b.EventPool[k]
					//范围外就tp 回来
					if _, isIn := v.MonsterList[monsterName]; isIn && !(pos[0] >= data.Position.StartPosition[0] && pos[1] >= data.Position.StartPosition[1] && pos[2] >= data.Position.StartPosition[2] && pos[0] <= (data.Position.StartPosition[0]+data.Position.ExpandPosition[0]) && pos[1] <= (data.Position.StartPosition[1]+data.Position.ExpandPosition[1]) && pos[2] <= (data.Position.StartPosition[2]+data.Position.ExpandPosition[2])) {
						cmd := fmt.Sprintf("tp @e[name=\"%v\"] %v %v %v", monsterName, data.Position.TpBackPos[0], data.Position.TpBackPos[1], data.Position.TpBackPos[2])
						b.Frame.GetGameControl().SendCmd(cmd)
					}
				}
			}
		}
		//检查怪物是否死亡
		for k, v := range b.ActiveEventPool {
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
		data := b.EventPool[k]
		for playerName, _ := range v.PlayerList {
			if pos, ok := playerPos[playerName]; ok {
				if len(pos) != 3 {
					return
				}
				if !(pos[0] >= data.Position.StartPosition[0] && pos[1] >= data.Position.StartPosition[1] && pos[2] >= data.Position.StartPosition[2] && pos[0] <= (data.Position.ExpandPosition[0]+data.Position.StartPosition[0]) && pos[1] <= (data.Position.ExpandPosition[1]+data.Position.StartPosition[1]) && pos[2] <= (data.Position.ExpandPosition[2]+data.Position.StartPosition[2])) {
					cmd := fmt.Sprintf("tp @a[name=\"%v\"] %v %v %v", playerName, data.Position.TpBackPos[0], data.Position.TpBackPos[1], data.Position.TpBackPos[2])
					b.Frame.GetGameControl().SendCmd(cmd)
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
	if _, ok := b.ActiveEventPool[eventName]; !ok {
		return
	}
	//奖励环节
	if len(b.ActiveEventPool[eventName].PlayerList) >= 1 {
		for playerName, _ := range b.ActiveEventPool[eventName].PlayerList {
			for _, cmd := range b.EventPool[eventName].PrizeCmd {
				relist := map[string]interface{}{
					"player": playerName,
					"获得积分分数": int(b.ActiveEventPool[eventName].TotolScore / len(b.ActiveEventPool[eventName].PlayerList)),
					"返回坐标":   fmt.Sprintf("%v %v %v", b.EventPool[eventName].Position.BackPos[0], b.EventPool[eventName].Position.BackPos[1], b.EventPool[eventName].Position.BackPos[2]), //b.EventPool[eventName].Position.BackPos[0]
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
	waiteTime := b.EventPool[eventName].WaiteTime.MinTime + rand.Intn(b.EventPool[eventName].WaiteTime.ExpandTime)
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
		if len(v) >= 3 {
			//遍历所有的事件 查看触发情况
			for EventName, event := range b.EventPool {

				//判断是否在内部
				if b.CheckInEvent(event, v) {
					//检查事件是否激活
					if _, ok := b.ActiveEventPool[EventName]; ok {
						//判断是否玩家重合
						if _, isok := b.ActiveEventPool[EventName].PlayerList[k]; !isok {
							if match, _ := regexp.MatchString("^-", k); !match {
								b.ActiveEventPool[EventName].PlayerList[k] = ""
								pterm.Info.Printfln("%v 事件 加入玩家 %v", EventName, k)
							}

						}
					} else if !(b.checkCool(EventName)) {
						b.EventRegister(EventName, k)
					}

				}
			}
		}

	}
}

// 检查是否还在冷却
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
	if len(event.Position.StartPosition) != 3 || len(event.Position.ExpandPosition) != 3 {
		pterm.Info.Println("事件配置中 坐标或者延长坐标 格式出现了错误")
		return false
	}
	if event.Position.StartPosition[0] <= pos[0] && event.Position.StartPosition[1] <= pos[1] && event.Position.StartPosition[2] <= pos[2] && (event.Position.ExpandPosition[0]+event.Position.StartPosition[0]) >= pos[0] && (event.Position.ExpandPosition[1]+event.Position.StartPosition[1]) >= pos[1] && (event.Position.ExpandPosition[2]+event.Position.StartPosition[2]) >= pos[2] {
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
	}
	//发送语音提示
	msg := b.EventPool[EventName].Words["事件启动时提示"]
	if !(len(b.EventPool[EventName].Position.StartPosition) >= 3 && len(b.EventPool[EventName].Position.ExpandPosition) >= 3) {
		pterm.Info.Printfln("你的事件%v 配置文件中坐标修改错误 通常这个错误是因为你没有按照[x,y,z]的格式来填写", EventName)
		panic("")
	}
	relist := map[string]interface{}{
		"事件名字": EventName,
		"x":    b.EventPool[EventName].Position.StartPosition[0],
		"y":    b.EventPool[EventName].Position.StartPosition[1],
		"z":    b.EventPool[EventName].Position.StartPosition[2],
		"dx":   b.EventPool[EventName].Position.StartPosition[0] + b.EventPool[EventName].Position.ExpandPosition[0],
		"dy":   b.EventPool[EventName].Position.StartPosition[1] + b.EventPool[EventName].Position.ExpandPosition[1],
		"dz":   b.EventPool[EventName].Position.StartPosition[2] + b.EventPool[EventName].Position.ExpandPosition[2],
	}
	msg = FormateMsg(b.Frame, relist, msg)
	b.Frame.GetGameControl().SayTo("@a", msg)
	//随机生成怪物
	rand.Seed(time.Now().Unix())
	randNum := rand.Intn(b.EventPool[EventName].LoadMaxNum) + 1
	//保证不会超过10次
	if randNum >= 10 {
		randNum = 10
	}
	pterm.Info.Println("事件触发成功 刷新次数为:", randNum)
	relist = map[string]interface{}{
		"刷新次数": randNum,
		"分值":   randNum * b.EventPool[EventName].AdScore,
	}
	//同步积分
	b.ActiveEventPool[EventName].TotolScore = randNum * b.EventPool[EventName].AdScore
	b.Frame.GetGameControl().SayTo("@a", FormateMsg(b.Frame, relist, b.EventPool[EventName].Words["刷新提示"]))
	cmd := fmt.Sprintf("execute @a[name=\"%v\"] ~~~ structure load %v ~~~ ", name, b.EventPool[EventName].StructName) //"structure load "
	go func() {
		for i := 1; i <= randNum; i++ {
			//为了等待返回结果完毕 保证指令全部执行完全
			time.Sleep(time.Millisecond * 50)
			b.Frame.GetGameControl().SendCmdAndInvokeOnResponse(fmt.Sprintf(cmd), func(output *packet.CommandOutput) {
				if output.SuccessCount > 0 {
					pterm.Info.Printfln("执行%v指令成功", cmd)
				} else {
					pterm.Info.Printfln("执行指令失败 失败指令为%v失败原因是:\n%v", cmd, output.OutputMessages)
				}
			})
		}
		//保证最后一次完整结束
		time.Sleep(time.Millisecond * 20)
		go func() {
			//为了把周围得怪物聚集起来 怕 玩家是跑进事件的
			b.Frame.GetGameControl().SendCmd(fmt.Sprintf("execute @a[name=\"%v\"] ~~~ tp @e[r=4,type=!player] ~~~", name))
			MonsterPos := <-GetPos(b.Frame, "@e")
			//将怪物加入列表
			for k, v := range MonsterPos {
				if len(v) != 3 {
					break
				}
				//检查是否是怪物
				if match, _ := regexp.MatchString("^-", k); match {
					//检查怪物是否存在在玩家范围内
					if v[0] >= (MonsterPos[name][0]-2) && v[2] >= (MonsterPos[name][2]-2) && v[0] <= (MonsterPos[name][0]+2) && v[2] <= (MonsterPos[name][2]+2) {
						b.ActiveEventPool[EventName].MonsterList[k] = "这是一个事件内的怪物"
					}
				}

			}
			pterm.Info.Printfln("%v事件内 怪物有 %v", EventName, b.ActiveEventPool[EventName].MonsterList)
			if len(b.ActiveEventPool[EventName].MonsterList) == 0 {
				b.ActiveEventPool[EventName].PlayerList = make(map[string]string)
				b.delectEvent(EventName)
				b.Frame.GetGameControl().SayTo("@a", "事件出现意外情况 启动失败")
			}
		}()
	}()

}
