package commands_generator

// extern void SetBlockRequestInternal(void *preallocatedStr, int x, int y, int z, const char *blockName, unsigned short data, const char *method);
// extern void SetBlockWithBlockStatesRequestInternal(void *preallocatedStr, int x, int y, int z, const char *blockName, const char *blockStates, const char *method);
import "C"
import (
	"fmt"
	"phoenixbuilder/fastbuilder/types"
	"unsafe"
)

func SetBlockRequest(buf *string, module *types.Module, config *types.MainConfig) {
	Block := module.Block
	Point := module.Point
	Method := config.Method
	if Block != nil {
		if len(Block.BlockStates)!=0 {
			C.SetBlockWithBlockStatesRequestInternal(unsafe.Pointer(buf), C.int(Point.X), C.int(Point.Y), C.int(Point.Z), C.CString(*Block.Name), C.CString(Block.BlockStates), C.CString(Method))
		}else{
			C.SetBlockRequestInternal(unsafe.Pointer(buf), C.int(Point.X), C.int(Point.Y), C.int(Point.Z), C.CString(*Block.Name), C.ushort(Block.Data), C.CString(Method))
		}
	} else {
		C.SetBlockRequestInternal(unsafe.Pointer(buf), C.int(Point.X), C.int(Point.Y), C.int(Point.Z), C.CString(config.Block.Name), C.ushort(config.Block.Data), C.CString(Method))
	}
	
}

func SetBlockRequestDEPRECATED(module *types.Module, config *types.MainConfig) string {
	Block := module.Block
	Point := module.Point
	Method := config.Method
	if Block != nil {
		return fmt.Sprintf("setblock %v %v %v %v %v %v",Point.X, Point.Y, Point.Z, *Block.Name, Block.Data, Method)
	} else {
		return fmt.Sprintf("setblock %v %v %v %v %v %v",Point.X, Point.Y, Point.Z, config.Block.Name, config.Block.Data, Method)
	}
}

