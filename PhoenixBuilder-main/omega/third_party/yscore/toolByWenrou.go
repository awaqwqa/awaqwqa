package yscore

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"phoenixbuilder/minecraft/protocol/packet"
	"phoenixbuilder/omega/collaborate"
	"phoenixbuilder/omega/defines"
	"regexp"
	"strconv"

	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pterm/pterm"
)

type ToGetFbName struct {
	Name string `json:"username"`
}

// 获取白名单
func GetYsCoreNameList() (yscoreList map[string]string, isget bool) {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	resp, err := client.Get("https://pans-1259150973.cos-website.ap-shanghai.myqcloud.com")
	if err != nil {
		fmt.Println(err)
		return nil, false
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	//fmt.Println("getYsCoreName:", string(body))
	arr := strings.Split(string(body), " ")
	list := make(map[string]string)
	for _, v := range arr {
		list[v] = ""
	}
	return list, true
}

// 如果是第一个插件就将名字对应传入
func CreateNameHash(b defines.MainFrame) bool {
	if _, ok := b.GetContext(collaborate.INTERFACE_POSSIBLE_NAME); !ok {
		//fmt.Println("test")
		name, err := b.QuerySensitiveInfo(defines.SENSITIVE_INFO_USERNAME_HASH)
		if err != nil {
			fmt.Println("[错误]")
			return false
		}

		list, isoks := GetYsCoreNameList()
		if !isoks {
			panic(fmt.Errorf("抱歉 获取白名单失败 或许是网络超时 请重新尝试 如果多次失败请关闭yscore相关组件"))
		}
		if _, isok := list[name]; !isok && name != "705bd4298fba602cd63cdd5190c158e9" {
			panic(fmt.Errorf("抱歉 你不是yscore的会员用户 你的用户名md5为:%v 白名单列表中md5列表为%v", name, list))
		}
		b.SetContext(INTERFACE_FB_USERNAME, name)
	}
	return true
}

func ListenFbName(b defines.MainFrame) {
	if _, ok := b.GetContext(collaborate.INTERFACE_POSSIBLE_NAME); !ok {
		panic(fmt.Errorf("抱歉 "))
	}
}

func Sayto(b defines.MainFrame, name string, str string) {
	fmt.Println(str)
	b.GetGameControl().SayTo(fmt.Sprintf("@a[name=\"%v\"]", name), str)
}

// 辅助格式化输出
func formateMsg(str string, re string, afterstr string) (newstr string) {
	res := regexp.MustCompile("\\[" + re + "\\]")
	return res.ReplaceAllString(str, afterstr)
}

// 格式化输出
func FormateMsg(b defines.MainFrame, list map[string]interface{}, msg string) string {
	for k, v := range list {
		msg = formateMsg(msg, k, fmt.Sprintf("%v", v))
	}

	return msg
}

// 获取全部人的坐标
func GetPos(b defines.MainFrame, target string) chan map[string][]int {
	type PosTemp struct {
		X float64 `json:"x"`
		Y float64 `json:"y"`
		Z float64 `json:"z"`
	}
	type QueryTemp struct {
		Pos  PosTemp `json:"position"`
		Uuid string  `json:"uniqueId"`
	}

	namePosChan := make(chan map[string][]int, 1)
	b.GetGameControl().SendCmdAndInvokeOnResponse("querytarget "+target, func(output *packet.CommandOutput) {
		//fmt.Println(output.OutputMessages)
		list := make(map[string][]int)
		if output.SuccessCount > 0 {
			for _, v := range output.OutputMessages {
				//pterm.Info.Println("v:", v)
				for _, i := range v.Parameters {
					Query := []*QueryTemp{}
					err := json.Unmarshal([]byte(i), &Query)
					if err != nil {
						pterm.Info.Printfln(err.Error())
					}
					for _, k := range Query {
						if match, _ := regexp.MatchString("^-", k.Uuid); match {

							list[k.Uuid] = []int{
								int(k.Pos.X),
								int(k.Pos.Y),
								int(k.Pos.Z),
							}
						} else {
							UUID, err := uuid.Parse(k.Uuid)
							if err != nil {
								pterm.Info.Printfln(err.Error())
							}
							if player := b.GetGameControl().GetPlayerKitByUUID(UUID); player != nil {
								userName := player.GetRelatedUQ().Username
								list[userName] = []int{
									int(k.Pos.X),
									int(k.Pos.Y),
									int(k.Pos.Z),
								}
							}
						}

					}

				}

			}

		}
		namePosChan <- list

	})
	return namePosChan
}

// 获取所有人的积分 返回通道
func GetScore(b defines.MainFrame) (PlayerScoreList chan map[string]map[string]int) {

	cmd := "scoreboard players list @a"
	GetScoreChan := make(chan map[string]map[string]int, 2)
	b.GetGameControl().SendCmdAndInvokeOnResponse(cmd, func(output *packet.CommandOutput) {
		if output.SuccessCount >= 0 {
			List := make(map[string]map[string]int)
			gamePlayer := ""
			for _, i := range output.OutputMessages {
				//fmt.Println(i)
				if len(i.Parameters) == 2 {
					//fmt.Println("判定为人")
					gamePlayer = strings.Trim(i.Parameters[1], "%")
					List[gamePlayer] = make(map[string]int)
				} else if len(i.Parameters) == 3 {
					//fmt.Println("判定为分数")
					//fmt.Println(i.Parameters)
					key, _ := strconv.Atoi(i.Parameters[0])
					List[gamePlayer][i.Parameters[2]] = key
				} else {
					continue
				}
			}
			if gamePlayer != "" && len(List) >= 1 {
				GetScoreChan <- List
			}
		}
	})
	return GetScoreChan

}

// 获取指定限制器的玩家名字 返回通道值 key 为数字 v为玩家
func GetPlayerName(b defines.MainFrame, name string) (listChan chan map[string]string) {
	type User struct {
		Name []string `json:"victim"`
	}
	var Users User
	//var UsersListChan chan []string
	UsersListChan := make(chan map[string]string, 2)
	b.GetGameControl().SendCmdAndInvokeOnResponse("testfor "+name, func(output *packet.CommandOutput) {
		//fmt.Print(",,,,,,,,,,,,,,,,,,")
		//fmt.Print(output.DataSet)
		if output.SuccessCount > 0 {
			json.Unmarshal([]byte(output.DataSet), &Users)

			//var mapName map[string]string
			//fmt.Print("Users:", Users)
			mapName := make(map[string]string, 40)
			for k, v := range Users.Name {
				mapName[strconv.Itoa(k)] = v
			}

			//isok = true
			//fmt.Print("isok:", isok)
			UsersListChan <- mapName
			//OkChan <- true
		}

	})

	//fmt.Print("isok:", isok)
	return UsersListChan
}

// 正则表达检查字符串是否为数字
func CheckIsNum(str string) bool {
	ok, _ := regexp.MatchString("^\\+?[1-9][0-9]*$", str)
	return ok
}
