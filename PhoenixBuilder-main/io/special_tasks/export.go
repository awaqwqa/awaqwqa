// +build !is_tweak

package special_tasks

import (
	"fmt"
	"time"
	"phoenixbuilder/fastbuilder/bdump"
	"phoenixbuilder/fastbuilder/configuration"
	"phoenixbuilder/fastbuilder/environment"
	"phoenixbuilder/fastbuilder/parsing"
	"phoenixbuilder/fastbuilder/task"
	"phoenixbuilder/fastbuilder/task/fetcher"
	"phoenixbuilder/fastbuilder/types"
	"phoenixbuilder/mirror"
	"phoenixbuilder/mirror/define"
	"phoenixbuilder/mirror/io/global"
	"phoenixbuilder/mirror/io/world"
	"phoenixbuilder/minecraft"
	"phoenixbuilder/minecraft/protocol"
	"phoenixbuilder/minecraft/protocol/packet"
	"phoenixbuilder/mirror/io/lru"
	"phoenixbuilder/mirror/chunk"
	"runtime/debug"
	"runtime"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/pterm/pterm"
)


type SolidSimplePos struct {
	X int64 `json:"x"`
	Y int64 `json:"y"`
	Z int64 `json:"z"`
}

type SolidRet struct {
	BlockName string `json:"blockName"`
	Position SolidSimplePos `json:"position"`
	StatusCode int64 `json:"statusCode"`
}

var ExportWaiter chan map[string]interface{}

func CreateExportTask(commandLine string, env *environment.PBEnvironment) *task.Task {
	cmdsender:=env.CommandSender
	// WIP
	//cmdsender.Output("Sorry, but compatibility works haven't been done yet, redirected to lexport.")
	//return CreateLegacyExportTask(commandLine, env)
	cfg, err := parsing.Parse(commandLine, configuration.GlobalFullConfig(env).Main())
	if err!=nil {
		cmdsender.Output(fmt.Sprintf("Failed to parse command: %v",err))
		return nil
	}
	//cmdsender.Output("Sorry, but compatibility works haven't been done yet, please use lexport.")
	//return nil
	beginPos := cfg.Position
	endPos := cfg.End
	startX,endX,startZ,endZ:=0,0,0,0
	if(endPos.X-beginPos.X<0) {
		temp:=endPos.X
		endPos.X=beginPos.X
		beginPos.X=temp
	}
	startX,endX=beginPos.X,endPos.X
	if(endPos.Y-beginPos.Y<0) {
		temp:=endPos.Y
		endPos.Y=beginPos.Y
		beginPos.Y=temp
	}
	if(endPos.Z-beginPos.Z<0) {
		temp:=endPos.Z
		endPos.Z=beginPos.Z
		beginPos.Z=temp
	}
	startZ,endZ=beginPos.Z,endPos.Z
	hopPath,requiredChunks:=fetcher.PlanHopSwapPath(startX,startZ,endX,endZ,16)
	chunkPool:=map[fetcher.ChunkPosDefine]fetcher.ChunkDefine{}
	memoryCacheFetcher:=fetcher.CreateCacheHitFetcher(requiredChunks,chunkPool)
	env.LRUMemoryChunkCacher.(*lru.LRUMemoryChunkCacher).Iter(func(pos define.ChunkPos, chunk *mirror.ChunkData) (stop bool) {
		memoryCacheFetcher(fetcher.ChunkPosDefine{int(pos[0])*16,int(pos[1])*16},fetcher.ChunkDefine(chunk))
		return false
	})
	hopPath=fetcher.SimplifyHopPos(hopPath)
	fmt.Println("Hop Left: ",len(hopPath))
	teleportFn:=func (x,z int)  {
		cmd:=fmt.Sprintf("tp @s %v 128 %v",x,z)
		uid,_:=uuid.NewUUID()
		cmdsender.SendCommand(cmd,uid)
		cmd=fmt.Sprintf("execute @s ~~~ spreadplayers ~ ~ 3 4 @s")
		uid,_=uuid.NewUUID()
		cmdsender.SendCommand(cmd,uid)
	}
	feedChan:=make(chan *fetcher.ChunkDefineWithPos,1024)
	deRegFn:=env.ChunkFeeder.(*global.ChunkFeeder).RegNewReader(func (chunk *mirror.ChunkData)  {
		feedChan<-&fetcher.ChunkDefineWithPos{Chunk: fetcher.ChunkDefine(chunk),Pos:fetcher.ChunkPosDefine{int(chunk.ChunkPos[0])*16,int(chunk.ChunkPos[1])*16}}
	})
	inHopping:=true
	go func() {
		return
		yc:=23
		for {
			if(!inHopping) {
				break
			}
			uuidval, _:=uuid.NewUUID()
			yv:=(yc-4)*16+8
			yc--
			if yc<0 {
				yc=23
			}
			cmdsender.SendCommand(fmt.Sprintf("tp @s ~ %d ~", yv),uuidval)
			time.Sleep(time.Millisecond*50)
		}
	} ()
	fmt.Println("Begin Fast Hopping")
	fetcher.FastHopper(teleportFn,feedChan,chunkPool,hopPath,requiredChunks,0.5,3)
	fmt.Println("Fast Hopping Done")
	deRegFn()
	hopPath=fetcher.SimplifyHopPos(hopPath)
	fmt.Println("Hop Left: ",len(hopPath))
	if len(hopPath)>0{
		fetcher.FixMissing(teleportFn,feedChan,chunkPool,hopPath,requiredChunks,2,3)
	}
	inHopping=false
	hasMissing:=false
	for _,c:=range requiredChunks{
		if !c.CachedMark{
			hasMissing=true
			pterm.Error.Printfln("Missing Chunk %v",c.Pos)
		}
	}
	if !hasMissing{
		pterm.Success.Println("all chunks successfully fetched!")
	}
	providerChunksMap:=make(map[define.ChunkPos]*mirror.ChunkData)
	for _,chunk:=range chunkPool{
		providerChunksMap[chunk.ChunkPos]=(*mirror.ChunkData)(chunk)
	}
	var offlineWorld *world.World
	offlineWorld=world.NewWorld(SimpleChunkProvider{providerChunksMap})

	go func() {
		defer func() {
			r:=recover()
			if r!=nil{
				debug.PrintStack()
				fmt.Println("go routine @ fastbuilder.task export crashed ",r)
			}
		}()
		cmdsender.Output("EXPORT >> Exporting...")
		V:=(endPos.X-beginPos.X+1)*(endPos.Y-beginPos.Y+1)*(endPos.Z-beginPos.Z+1)
		blocks:=make([]*types.Module,V)
		counter:=0
		for x:=beginPos.X; x<=endPos.X; x++ {
			for z:=beginPos.Z; z<=endPos.Z; z++ {
				for y:=beginPos.Y; y<=endPos.Y; y++ {
					runtimeId, item, found:=offlineWorld.BlockWithNbt(define.CubePos{x,y,z})
					if !found {
						fmt.Printf("WARNING %d %d %d not found\n", x, y, z)
					}
					//block, item:=blk.EncodeBlock()
					block, static_item, _ := chunk.RuntimeIDToState(runtimeId)
					if block=="minecraft:air" {
						continue
					}
					var cbdata *types.CommandBlockData = nil
					var chestData *types.ChestData = nil
					var nbtData []byte = nil
					/*if(block=="chest"||block=="minecraft:chest"||strings.Contains(block,"shulker_box")) {
						content:=item["Items"].([]interface{})
						chest:=make(types.ChestData, len(content))
						for index, iface := range content {
							i:=iface.(map[string]interface{})
							name:=i["Name"].(string)
							count:=i["Count"].(uint8)
							damage:=i["Damage"].(int16)
							slot:=i["Slot"].(uint8)
							name_mcnk:=name[10:]
							chest[index]=types.ChestSlot {
								Name: name_mcnk,
								Count: count,
								Damage: uint16(int(damage)),
								Slot: slot,
							}
						}
						chestData=&chest
					}*/
					// TODO ^ Hope someone could help me to do that, just like what I did below ^
					if strings.Contains(block,"command_block") {
						/*
							=========
							Reference
							=========
							Types for command blocks are checked by their names
							Whether a command block is conditional is checked through its data value.
							THEY ARE NOT INCLUDED IN NBT DATA.
							
							normal
							\x01\x00\x00\x00\x00\x01\x00\x00\x00\bsay test\"\x00\x00\x00\x00\x01\xfa\xcd\x03\x00\x00
							===
							tick 60
							\x01\x00\x00\x00\x00\x01\x00\x00\x00\bsay test\"\x00\x00\x00\x00\x01\xfa\xcd\x03x\x00
							===
							no tracking output, tick 62
							\x01\x00\x00\x00\x00\x01\x00\x00\x00\bsay test\"\x00\x00\x00\x00\x00\xfa\xcd\x03|\x00
							===
							tick 62, custom name = "***"
							\x01\x00\x00\x00\x00\x01\x00\x00\x00\bsay test\"\x00\x03***\x00\x00\x01\xfa\xcd\x03|\x00
							===
							tick 62, w/ error output, executeonfirsttick
							\x01\x00\x00\x00\x00\x01\x00\x00\x00\tdsay test\"\x00\x00\x17commands.generic.syntax\x06\x00\x04dsay\x05 test\x01\xfa\xcd\x03|\x01
							===
							same with above, but will not execute on first tick
							\x01\x00\x00\x00\x00\x01\x00\x00\x00\tdsay test\"\x00\x00\x17commands.generic.syntax\x06\x00\x04dsay\x05 test\x01\xfa\xcd\x03|\x00
							===
							normal, noredstone
							\x01\x00\x00\x00\x01\x01\x00\x00\x00\bsay test\"\x02\x00\x00\x00\x01\xfa\xcd\x03\x00\x00
						*/
						__tag:=[]byte(item["__tag"].(string))
						//fmt.Printf("CMDBLK %#v\n\n",item["__tag"])
						var mode uint32
						if(block=="command_block"||block=="minecraft:command_block"){
							mode=packet.CommandBlockImpulse
						}else if(block=="repeating_command_block"||block=="minecraft:repeating_command_block"){
							mode=packet.CommandBlockRepeating
						}else if(block=="chain_command_block"||block=="minecraft:chain_command_block"){
							mode=packet.CommandBlockChain
						}
						len_tag:=len(__tag)
						tickdelay:=int32(__tag[len_tag-2])/2
						exeft:=__tag[len_tag-1]
						aut:=__tag[4]
						trackoutput:=__tag[len_tag-6]
						cmdlen:=__tag[9]
						cmd:=string(__tag[10:10+cmdlen])
						//cmd:=item["Command"].(string)
						cusname_len:=__tag[10+cmdlen+2]
						cusname:=string(__tag[10+cmdlen+2+1:10+cmdlen+2+1+cusname_len])
						//cusname:=item["CustomName"].(string)
						lo_len:=__tag[10+cmdlen+2+1+cusname_len]
						lo:=string(__tag[10+cmdlen+2+1+cusname_len+1:10+cmdlen+2+1+cusname_len+1+lo_len])
						//exeft:=item["ExecuteOnFirstTick"].(uint8)
						//tickdelay:=item["TickDelay"].(int32)
						//aut:=item["auto"].(uint8)
						//trackoutput:=item["TrackOutput"].(uint8)
						//lo:=item["LastOutput"].(string)
						conb_bit:=static_item["conditional_bit"].(uint8)
						conb:=false
						if conb_bit==1 {
							conb=true
						}
						var exeftb bool
						if exeft==0 {
							exeftb=true
						}else{
							exeftb=true
						}
						var tob bool
						if trackoutput==1 {
							tob=true
						}else{
							tob=false
						}
						var nrb bool
						if aut==1 {
							nrb=false
							//REVERSED!!
						}else{
							nrb=true
						}
						cbdata=&types.CommandBlockData {
							Mode: mode,
							Command: cmd,
							CustomName: cusname,
							ExecuteOnFirstTick: exeftb,
							LastOutput: lo,
							TickDelay: tickdelay,
							TrackOutput: tob,
							Conditional: conb,
							NeedRedstone: nrb,
						}
						//fmt.Printf("%#v\n",cbdata)
					}else{
						pnd, hasNBT:=item["__tag"]
						if hasNBT {
							nbtData=[]byte(pnd.(string))
						}
					}
					lb:=chunk.RuntimeIDToLegacyBlock(runtimeId)
					blocks[counter]=&types.Module {
						Block: &types.Block {
							Name: &lb.Name,
							Data: uint16(lb.Val),
						},
						CommandBlockData: cbdata,
						ChestData: chestData,
						NBTData: nbtData,
						Point: types.Position {
							X: x,
							Y: y,
							Z: z,
						},
					}
					counter++
				}
			}
		}
		blocks=blocks[:counter]
		runtime.GC()
		out:=bdump.BDumpLegacy {
			Blocks: blocks,
		}
		if(strings.LastIndex(cfg.Path,".bdx")!=len(cfg.Path)-4||len(cfg.Path)<4) {
			cfg.Path+=".bdx"
		}
		cmdsender.Output("EXPORT >> Writing output file")
		err, signerr:=out.WriteToFile(cfg.Path, env.LocalCert, env.LocalKey)
		if(err!=nil){
			cmdsender.Output(fmt.Sprintf("EXPORT >> ERROR: Failed to export: %v",err))
			return
		}else if(signerr!=nil) {
			cmdsender.Output(fmt.Sprintf("EXPORT >> Note: The file is unsigned since the following error was trapped: %v",signerr))
		}else{
			cmdsender.Output(fmt.Sprintf("EXPORT >> File signed successfully"))
		}
		cmdsender.Output(fmt.Sprintf("EXPORT >> Successfully exported your structure to %v",cfg.Path))
		runtime.GC()
	} ()
	return nil
}

func CreateLegacyExportTask(commandLine string, env *environment.PBEnvironment) *task.Task {
	cfg, err := parsing.Parse(commandLine, configuration.GlobalFullConfig(env).Main())
	if err!=nil {
		env.CommandSender.Output(fmt.Sprintf("Failed to parse command: %v", err))
		return nil
	}
	
	beginPos := cfg.Position
	endPos   := cfg.End
	msizex:=0
	msizey:=0
	msizez:=0
	if(endPos.X-beginPos.X<0) {
		temp:=endPos.X
		endPos.X=beginPos.X
		beginPos.X=temp
	}
	msizex=endPos.X-beginPos.X+1
	if(endPos.Y-beginPos.Y<0) {
		temp:=endPos.Y
		endPos.Y=beginPos.Y
		beginPos.Y=temp
	}
	msizey=endPos.Y-beginPos.Y+1
	if(endPos.Z-beginPos.Z<0) {
		temp:=endPos.Z
		endPos.Z=beginPos.Z
		beginPos.Z=temp
	}
	msizez=endPos.Z-beginPos.Z+1
	gsizez:=msizez
	go func() {
		u_d, _ := uuid.NewUUID()
		env.CommandSender.SendWSCommand("gamemode c", u_d)
		originx:=0
		originz:=0
		var blocks []*types.Module
		for {
			env.CommandSender.Output("EXPORT >> Fetching data")
			cursizex:=msizex
			cursizez:=msizez
			if msizex>100 {
				cursizex=100
			}
			if msizez>100 {
				cursizez=100
			}
			posx:=beginPos.X+originx*100
			posz:=beginPos.Z+originz*100
			u_d2, _ := uuid.NewUUID()
			wchan:=make(chan *packet.CommandOutput)
			(*env.CommandSender.GetUUIDMap()).Store(u_d2.String(),wchan)
			env.CommandSender.SendWSCommand(fmt.Sprintf("tp %d %d %d",posx,beginPos.Y+1,posz), u_d2)
			<-wchan
			close(wchan)
			ExportWaiter=make(chan map[string]interface{})
			env.Connection.(*minecraft.Conn).WritePacket(&packet.StructureTemplateDataRequest {
				StructureName: "mystructure:a",
				Position: protocol.BlockPos {int32(posx),int32(beginPos.Y),int32(posz)},
				Settings: protocol.StructureSettings {
					PaletteName: "default",
					IgnoreEntities: true,
					IgnoreBlocks: false,
					Size: protocol.BlockPos {int32(cursizex),int32(msizey),int32(cursizez)},
					Offset: protocol.BlockPos {0,0,0},
					LastEditingPlayerUniqueID: env.Connection.(*minecraft.Conn).GameData().EntityUniqueID,
					Rotation: 0,
					Mirror: 0,
					Integrity: 100,
					Seed: 0,
				},
				RequestType: packet.StructureTemplateRequestExportFromSave,
			})
			exportData:=<-ExportWaiter
			close(ExportWaiter)
			env.CommandSender.Output("EXPORT >> Data received, processing.")
			env.CommandSender.Output("EXPORT >> Extracting blocks")
			sizeoo, _:=exportData["size"].([]interface{})
			if len(sizeoo)==0 {
				originz++
				msizez-=cursizez
				if(msizez<=0){
					msizez=gsizez
					originz=0
					originx++
					msizex-=cursizex
				}
				if(msizex<=0) {
					break
				}
				continue
			}
			sizea,_:=sizeoo[0].(int32)
			sizeb,_:=sizeoo[1].(int32)
			sizec,_:=sizeoo[2].(int32)
			size:=[]int{int(sizea),int(sizeb),int(sizec)}
			structure, _:=exportData["structure"].(map[string]interface{})
			indicesP, _:=structure["block_indices"].([]interface{})
			indices,_:=indicesP[0].([]interface{})
			if len(indicesP)!=2 {
				panic(fmt.Errorf("Unexcepted indices data: %v\n",indices))
			}
			{
				ind,_:=indices[0].(int32)
				if ind==-1 {
					indices,_=indicesP[1].([]interface{})
				}
				ind,_=indices[0].(int32)
				if ind==-1 {
					panic(fmt.Errorf("Exchanged but still -1: %v\n",indices))
				}
			}
			blockpalettepar,_:=structure["palette"].(map[string]interface{})
			blockpalettepar2,_:=blockpalettepar["default"].(map[string]interface{})
			blockpalette,_:=blockpalettepar2["block_palette"].([]/*map[string]*/interface{})
			blockposdata,_:=blockpalettepar2["block_position_data"].(map[string]interface{})
			airind:=int32(-1)
			i:=0
			for x:=0;x<size[0];x++ {
				for y:=0;y<size[1];y++ {
					for z:=0;z<size[2];z++ {
						ind,_:=indices[i].(int32)
						if ind==-1 {
							i++
							continue
						}
						if ind==airind {
							i++
							continue
						}
						curblock,_:=blockpalette[ind].(map[string]interface{})
						curblocknameunsplitted,_:=curblock["name"].(string)
						curblocknamesplitted:=strings.Split(curblocknameunsplitted,":")
						curblockname:=curblocknamesplitted[1]
						var cbdata *types.CommandBlockData=nil
						if curblockname=="air" {
							i++
							airind=ind
							continue
						}else if(!cfg.ExcludeCommands&&strings.Contains(curblockname,"command_block")) {
							itemp,_:=blockposdata[strconv.Itoa(i)].(map[string]interface{})
							item,_:=itemp["block_entity_data"].(map[string]interface{})
							var mode uint32
							if(curblockname=="command_block"){
								mode=packet.CommandBlockImpulse
							}else if(curblockname=="repeating_command_block"){
								mode=packet.CommandBlockRepeating
							}else if(curblockname=="chain_command_block"){
								mode=packet.CommandBlockChain
							}
							cmd,_:=item["Command"].(string)
							cusname,_:=item["CustomName"].(string)
							exeft,_:=item["ExecuteOnFirstTick"].(uint8)
							tickdelay,_:=item["TickDelay"].(int32)//*/
							aut,_:=item["auto"].(uint8)//!needrestone
							trackoutput,_:=item["TrackOutput"].(uint8)//
							lo,_:=item["LastOutput"].(string)
							conditionalmode:=item["conditionalMode"].(uint8)
							var exeftb bool
							if exeft==0 {
								exeftb=false
							}else{
								exeftb=true
							}
							var tob bool
							if trackoutput==1 {
								tob=true
							}else{
								tob=false
							}
							var nrb bool
							if aut==1 {
								nrb=false
								//REVERSED!!
							}else{
								nrb=true
							}
							var conb bool
							if conditionalmode==1 {
								conb=true
							}else{
								conb=false
							}
							cbdata=&types.CommandBlockData {
								Mode: mode,
								Command: cmd,
								CustomName: cusname,
								ExecuteOnFirstTick: exeftb,
								LastOutput: lo,
								TickDelay: tickdelay,
								TrackOutput: tob,
								Conditional: conb,
								NeedRedstone: nrb,
							}
						}
						curblockdata,_:=curblock["val"].(int16)
						blocks=append(blocks,&types.Module{
							Block: &types.Block {
								Name:&curblockname,
								Data:uint16(curblockdata),
							},
							CommandBlockData: cbdata,
							Point: types.Position {
								X: originx*100+x,
								Y: y,
								Z: originz*100+z,
							},
						})
						i++
					}
				}
			}
			originz++
			msizez-=cursizez
			if(msizez<=0){
				msizez=gsizez
				originz=0
				originx++
				msizex-=cursizex
			}
			if(msizex<=0) {
				break
			}
		}
		out:=bdump.BDumpLegacy {
			Blocks: blocks,
		}
		if(strings.LastIndex(cfg.Path,".bdx")!=len(cfg.Path)-4||len(cfg.Path)<4) {
			cfg.Path+=".bdx"
		}
		env.CommandSender.Output("EXPORT >> Writing output file")
		err, signerr:=out.WriteToFile(cfg.Path, env.LocalCert, env.LocalKey)
		if(err!=nil){
			env.CommandSender.Output(fmt.Sprintf("EXPORT >> ERROR: Failed to export: %v",err))
			return
		}else if(signerr!=nil) {
			env.CommandSender.Output(fmt.Sprintf("EXPORT >> Note: The file is unsigned since the following error was trapped: %v",signerr))
		}else{
			env.CommandSender.Output(fmt.Sprintf("EXPORT >> File signed successfully"))
		}
		env.CommandSender.Output(fmt.Sprintf("EXPORT >> Successfully exported your structure to %v",cfg.Path))
	} ()
	return nil
}

