package yscore

import (
	"encoding/json"
	"phoenixbuilder/minecraft/protocol/packet"
	"phoenixbuilder/omega/defines"

	"github.com/pterm/pterm"
)

type TellCmd struct {
	*defines.BasicComponent
	CmdPool map[string][]*Tellcmd `json:"指令组"`
	Usage   string                `json:"描述"`
	Prefix  string                `json:"前缀"`
	IsTrue  bool
}
type Tellcmd struct {
	Cmd     string `json:"指令"`
	IsAllow bool   `json:"是否有条件"`
}

func (o *TellCmd) Init(cfg *defines.ComponentConfig) {
	marshal, _ := json.Marshal(cfg.Configs)
	if err := json.Unmarshal(marshal, o); err != nil {
		panic(err)
	}
	o.IsTrue = true
}
func (o *TellCmd) Inject(frame defines.MainFrame) {
	o.Frame = frame
	o.BasicComponent.Inject(frame)
	o.Frame.GetGameListener().SetGameChatInterceptor(o.onChat)
	CreateNameHash(o.Frame)
}

func (o *TellCmd) onChat(chat *defines.GameChat) (stop bool) {

	if len(chat.Msg) >= 2 && chat.Msg[0] == o.Prefix {
		for k, v := range o.CmdPool {
			//如果第二关键词符合
			if k == chat.Msg[1] && len(v) >= 1 {
				o.Executor(v, 0)
			}
		}
	}
	return true
}

// 执行指令并且保存当前执行进度
func (b *TellCmd) Executor(cmds []*Tellcmd, num int) {

	if num == 0 {
		pterm.Info.Println("开始执行指令组:", cmds)
	}
	//如果是条件执行 且条件不满足
	if cmds[num].IsAllow && (!b.IsTrue) {
		if len(cmds) > num+1 {
			b.Executor(cmds, num+1)
		} else {
			pterm.Info.Println("已完成指令执行")
		}

		return
	}
	b.Frame.GetGameControl().SendCmdAndInvokeOnResponse(cmds[num].Cmd, func(output *packet.CommandOutput) {
		if output.SuccessCount > 0 {
			b.IsTrue = true
		} else {
			b.IsTrue = false
		}
		pterm.Info.Println("当前指令执行返回结果为:", b.IsTrue)
		//如果还有剩下的指令则继续执行 否则就输出完成
		if len(cmds) > num+1 {
			b.Executor(cmds, num+1)
		} else {
			pterm.Info.Println("已完成指令执行")
		}

	})

}
