package yscore

import (
	"encoding/json"
	"math/rand"
	"phoenixbuilder/omega/defines"
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
}
type CoEvent struct {
	ColdTime int64
}

// 事件
type Event struct {
	Position    Pos               `json:"范围"`
	WaiteTime   WaiteTime         `json:"冷却时间范围"`
	LoadMaxNum  int               `json:"每次刷新结构随机刷新次数上限(1-10)"`
	StructName  string            `json:"结构名字"`
	Words       map[string]string `json:"提示话语"`
	AdScore     int               `json:"单次刷新总分值"`
	RanketScore string
}

// 坐标
type Pos struct {
	StartPosition  []int `json:"起点坐标"`
	ExpandPosition []int `json:"延长范围"`
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
			pterm.Info.Println(playerPos)
			//首先是获取全部人的坐标 然后检测是否应该激活事件 -->tp所有非死亡且离开用户进入事件
			//-->然后tp所有离开的怪物进入事件-->检查事件的人数与怪物数量-->如果剩余人数或者怪物为0时杀死事件
			//--》将杀死了的事件进入冷却池子 然后标明应该等待的刻度时间<秒>

			//首先是注册事件
			b.AddEvent(playerPos)

		}()
	}
}

// 添加事件
func (b *RandomEvent) AddEvent(playerPos map[string][]int) {
	for k, v := range playerPos {
		if len(v) >= 3 {
			for EventName, event := range b.EventPool {

				//判断是否在内部
				if len(event.Position.StartPosition) >= 3 && event.Position.StartPosition[0] <= v[0] && event.Position.StartPosition[1] <= v[1] && event.Position.StartPosition[2] <= v[2] && len(event.Position.ExpandPosition) >= 3 && event.Position.ExpandPosition[0] >= v[0] && event.Position.ExpandPosition[1] >= v[1] && event.Position.ExpandPosition[2] >= v[2] {
					//检查事件是否激活
					if _, ok := b.ActiveEventPool[EventName]; ok {
						//判断是否玩家重合
						if _, isok := b.ActiveEventPool[EventName].PlayerList[k]; !isok {
							b.ActiveEventPool[EventName].PlayerList[k] = ""
							pterm.Info.Printfln("%v 事件 加入玩家 %v", EventName, k)
						}
					} else {
						b.EventRegister(EventName, k)
					}

				}
			}
		}

	}
}

// 注册事件 首先要传入首个进入的人员
func (b *RandomEvent) EventRegister(EventName string, name string) {
	//初始化人物
	b.ActiveEventPool[EventName] = &AcEvent{
		PlayerList: map[string]string{
			name: "",
		},
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
	rand.Seed(time.Now().Unix())
}
