package virtualFileSystem

import (
	u "vfs/utilities"
	"fmt"
	"github.com/fatih/color"
	"strconv"
	"strings"
	"time"
)

// TODO 5.  修改文件时未修改其改动时间 优先级4
// TODO 7.  打印超级块功能 优先级4
// TODO 8.  超级块属性维护（计算剩余inode数，计算磁盘空间等）优先级4

func Hash(fsMagic int, inodeNum int) string {
	return strconv.Itoa(fsMagic) + "|" + strconv.Itoa(inodeNum)
}

func (v Vfs) initPath(path string) (p Path) {

	path = v.parseRelativePath(path)

	p.pathSlice = strings.Split(path, "/")[1:]
	p.depth = len(p.pathSlice)
	p.currentIndex = 0
	p.pathString = path
	return
}

// 查询过程中，很重要的一个点是判断当前的目录是不是一个挂载点
// 如果是的话，通过vfsmount结构可以得到当前目录的超级块
// 一个目录项若要成为挂载点，那么它首先应该存在，并且为空目录
func (v *Vfs) Init(sb SuperBlock) {
	v.rootSb = sb
	v.rootVnode.inode = sb.GetRoot()
	v.rootVnode.sb = sb
	v.curDir = v.initPath("/")
}
func (v Vfs) Pwd() {
	fmt.Println(v.curDir.pathString)
}
func (v Vfs) GetCur() string {
	return v.curDir.pathString
}

// TODO 3.  多文件系统基础设施已经打好，未实现（mount和unmount功能）优先级3

func (v *Vfs) registerSuperBlock(path string, order int) {

}
func (v *Vfs) getInodeByPath(path string) (Inode, bool) {
	if path == "/" {
		v.curDir = v.initPath(path)
		return v.rootVnode.inode, true
	}

	p := v.initPath(path)
	path = p.pathString
	sb := v.rootSb
	curInode := v.rootVnode.inode
	curDir := "/"

	for _, x := range p.pathSlice {
		curDir += x
		// FIXME mount
		newInodeNum := curInode.LookUp(x)

		if newInodeNum > 0 {
			nInode := sb.ReadInode(newInodeNum)
			nInode.SetSb(sb)
			curInode = nInode
		} else {
			fmt.Println("no such kind of dir in path", path, " with name ", x)
			curInode.SetSb(v.rootSb)
			return curInode, false
		}

	}
	curInode.SetSb(v.rootSb)
	return curInode, true
}

// 工作目录必须是有效的
func (v *Vfs) ChangeDir(path string) {

	p := v.initPath(path)
	path = p.pathString
	ino, ok := v.getInodeByPath(path)
	if ok {
		if ino.GetAttr().FileType != u.Directory {
			fmt.Println("这不是一个目录！")
		} else {
			v.curDir = v.initPath(path)
		}
	} else {
		fmt.Println("不存在这样的目录")
	}
}

// TODO 1.  对不同的指令设置不同的补全项 优先级0
func (v *Vfs) GetFileListInCurrentDir() (list []string, ok bool) {
	ino, ok := v.getInodeByPath(v.curDir.pathString)
	if ok {
		if ino.GetAttr().FileType != u.Directory {
			fmt.Println("错误！当前项不是一个目录！")
		} else {
			list, ok := ino.List()
			if ok {
				return list, true
			}
		}
	} else {
		fmt.Println("当前目录不存在")
	}
	return
}
func (v *Vfs) GetDiristInCurrentDir() (dirs []string, ok bool) {
	ino, ok := v.getInodeByPath(v.curDir.pathString)
	if ok {
		if ino.GetAttr().FileType != u.Directory {
			fmt.Println("错误！当前项不是一个目录！")
		} else {
			list, ok := ino.List()
			if ok {
				for _, x := range list {
					num := ino.LookUp(x)
					cino := ino.GetSb().ReadInode(num)
					if cino.GetAttr().FileType == u.Directory {
						dirs = append(dirs, x)
					}
				}
				return dirs, true
			}
		}
	} else {
		fmt.Println("当前目录不存在")
	}
	return
}
func (v *Vfs) ListCurrentDir() {
	ino, ok := v.getInodeByPath(v.curDir.pathString)
	if ok {
		if ino.GetAttr().FileType != u.Directory {
			fmt.Println("错误！当前项不是一个目录！")
		} else {
			list, ok := ino.List()
			if ok {
				for _, x := range list {
					in, _ := v.getInodeByPath(x)
					if in.GetAttr().FileType != 0 && x != "" {
						fmt.Println(beutifyString(in.GetAttr(), x))
					}
				}

			}
		}
	} else {
		fmt.Println("当前目录不存在")
	}
}
func (v *Vfs) ListDir(path string) {
	ino, ok := v.getInodeByPath(path)

	if ok {
		fmt.Println(ino.GetAttr())
	} else {
		_ = fmt.Errorf("stat error: path %s not found", path)
	}
}
func (v Vfs) Stat(path string) {
	ino, ok := v.getInodeByPath(path)

	if ok {
		fmt.Println(ino.GetAttr())
	} else {
		_ = fmt.Errorf("stat error: path %s not found", path)
	}
}
func (v *Vfs) Touch(path string) {
	p := v.initPath(path)
	path = p.pathString
	if p.depth < 1 {
		return
	}
	parentPath, childName := p.splitParentAndChild()
flag:
	parentInode, ok := v.getInodeByPath(parentPath)
	if ok {
		v.rootSb.CreateFile(childName, parentInode, 1)
	} else {
		v.createParentDir(parentPath)
		goto flag
	}

}
func (v *Vfs) MakeDir(path string) {
	p := v.initPath(path)
	path = p.pathString
	if p.depth < 1 {
		return
	}
	parentPath, childName := p.splitParentAndChild()

	parentInode, ok := v.getInodeByPath(parentPath)
	if ok {
		if parentInode.GetAttr().FileType != u.Directory {
			fmt.Println("mkdir error: ", "path: ", parentPath, " is not a directory")
			return
		}
		v.rootSb.CreateFile(childName, parentInode, int(u.Directory))
	} else {
		v.createParentDir(parentPath)
		parentInode, ok := v.getInodeByPath(parentPath)
		if ok {
			if parentInode.GetAttr().FileType != u.Directory {
				fmt.Println("mkdir error: ", "path: ", parentPath, " is not a directory")
				return
			}
			v.rootSb.CreateFile(childName, parentInode, int(u.Directory))
		} else {
			fmt.Println("Fatal error, no possible")
		}
	}
}
func (p Path) splitParentAndChild() (parent string, child string) {

	if p.depth == 1 {
		return "/", p.pathSlice[0]
	}
	parent = "/"
	for i, x := range p.pathSlice[:p.depth-1] {
		if i != p.depth-2 {
			parent += x + "/"
		} else {
			parent += x
		}
	}
	child = p.pathSlice[p.depth-1]
	return
}

// TODO 10. 解析".",".."目录
func (v *Vfs) parseRelativePath(path string) string {
	if !strings.HasPrefix(path, "/") {
		// 说明要解析的是相对路径
		if v.curDir.pathString == "/" {
			path = "/" + path
		} else {
			path = v.curDir.pathString + "/" + path
		}
	}
	return path
}
func (v *Vfs) createParentDir(path string) {
	root := v.rootVnode
	p := v.initPath(path)
	path = p.pathString
	if p.depth < 1 {
		return
	}
	curInode := root.inode
	curSb := v.rootSb
	for _, x := range p.pathSlice {
		num := curInode.LookUp(x)
		if num == 0 {
			curSb.CreateFile(x, curInode, 2)
			num := curInode.LookUp(x)
			if num > 0 {
				curInode = curSb.ReadInode(num)
				fmt.Println("impossible")
			}
		} else {

		}
	}
}

// TODO 2.  删除文件时未考虑Cache一致性 优先级0
func (v *Vfs) Remove(path string) {
	p := v.initPath(path)
	path = p.pathString
	if p.depth < 1 {
		return
	}
	parentPath, childName := p.splitParentAndChild()

	parentInode, ok := v.getInodeByPath(parentPath)

	if ok && parentInode.GetAttr().FileType == u.Directory {
		parentInode.SetSb(v.rootSb)
		ok = parentInode.Remove(childName)
		if !ok {
			fmt.Errorf("fatal error, can't delete file ", path)
		}
	} else {
		_ = fmt.Errorf("Not such file ", path)
	}
}
func beutifyString(attr InodeAttr, name string) string {
	tm := time.Unix(int64(attr.Ctime), 0)
	blue := color.New(color.FgHiCyan).SprintFunc()
	time := fmt.Sprintf(tm.Format("2006-01-02 03:04:05 PM"))
	fileType := ""
	if attr.FileType == u.Directory {
		fileType += "Directory"
		name = blue(name)
	} else {
		fileType += "Plain Text"
	}
	return fmt.Sprintf("drwxr-xr-x  %5db %s %-10s %-20s", attr.Size, time, fileType, name)
}

// TODO 4.  编辑文件功能残缺 优先级0
func (v *Vfs) Append(path string, data string) {
	p, ok := v.getInodeByPath(path)
	if ok {
		p.Append(data)
	} else {
		fmt.Println("not fount")
	}
}
func (v *Vfs) Cat(path string) {
	ino, ok := v.getInodeByPath(path)

	if ok {
		data := ino.ReadAll()
		fmt.Println(string(data))
	} else {
		_ = fmt.Errorf("stat error: path %s not found", path)
	}
}

// TODO 9. 软链接的基本实现 优先级5

func (v *Vfs) SoftLink(path1 string, path2 string) {

}
