package ext0

import (
	vfs "vfs/virtualFileSystem"
	"encoding/binary"
	"fmt"
	u "vfs/utilities"
)


type FileType int

type Ext0Inode struct {
	attr vfs.InodeAttr
	sb   *Ext0SuperBlock
}

type Exto0DirectoryStorage struct {
	name        [DirStorageSIze - 2]byte // total 256byte
	inodeNumber uint16                   // 2 byte
}

func (node *Ext0Inode) NewInode() {
	node.attr.InodeNumber = node.sb.GetNextFreeInodeNumber()

}

func (ino *Ext0Inode) Create() {

}
func compareDirName(name string, b [DirStorageSIze - 2]byte) bool {
	nameByte := []byte(name)
	empty := make([]byte, DirStorageSIze-2-len(nameByte))
	nameByte = append(nameByte, empty...)
	if len(nameByte) > DirStorageSIze-2 {
		return false
	} else {
		for i, x := range nameByte {
			if x != b[i] {
				return false
			}
		}
	}
	return true
}
func getName(b [DirStorageSIze - 2]byte) string {
	var nameByte []byte
	for _, x := range b {
		if x != 0 {
			nameByte = append(nameByte, x)
		} else {

			return string(nameByte)
		}
	}
	return ""
}
func (ino *Ext0Inode) LookUp(name string) int {
	dir := ino.sb.ReadDir(ino.attr)
	for _, d := range dir {

		if compareDirName(name, d.name) {
			return int(d.inodeNumber)
		}
	}
	return 0
}
func (ino *Ext0Inode) List() (dirList []string, ok bool) {

	if ino.attr.FileType != u.Directory {
		return
	} else {
		dir := ino.sb.ReadDir(ino.attr)
		for _, d := range dir {
			if getName(d.name) == "." || getName(d.name) == ".." || getName(d.name) == "" {
				continue
			}
			dirList = append(dirList, getName(d.name))
			//getName(d.name)
		}

	}
	return dirList, true
}
func (ino *Ext0Inode) Link() {

}
func (ino *Ext0Inode) FollowLink() {

}
func (ino *Ext0Inode) SeAttr(attr vfs.InodeAttr) {
	ino.attr = attr
}
func (ino *Ext0Inode) GetAttr() vfs.InodeAttr {
	return ino.attr
}

// modify the blockCount attr, causing inconsistency
func (ino *Ext0Inode) allocBlock() int {
	num := ino.sb.GetNextFreeBlockeNumber()
	ino.sb.FreeBlockNumber += 1
	if ino.attr.StartAddr == 0 {
		ino.attr.StartAddr = num
		ino.sb.setBlockBitmap(int(ino.attr.StartAddr), true)
		ino.attr.BlockCount += 1
		return int(num)
	} else {
		addr := ino.attr.StartAddr
		for true {
			next := ino.sb.getFat(addr)
			// 将next标记为使用状态
			if next == 0 {
				ino.sb.setFat(int(addr), num)
				ino.sb.setBlockBitmap(int(num), true)
				ino.attr.BlockCount += 1
				return int(num)
			} else {
				addr = next
			}
		}
	}
	return 0
}
func makeDirData(name string, num uint16) []byte {
	nameByte := make([]byte, DirStorageSIze-2)
	for i, x := range []byte(name) {
		nameByte[i] = x
	}
	if len(nameByte) > DirStorageSIze-2 {
		return make([]byte, 0)
	} else {
		numBuf := make([]byte, 2)
		binary.BigEndian.PutUint16(numBuf, num)
		return append(nameByte, numBuf...)
	}
}

// 修改目录的attr，在其数据区初始化父子目录项，最后把inode写入磁盘
func (ino *Ext0Inode) initAsDir(parent uint16, sb *Ext0SuperBlock) uint16 {
	// buf 中存放最底层的切片指针
	ino.attr = sb.initAttr()
	ino.attr.BlockCount = 0
	ino.attr.FileType = u.Directory
	ino.sb = sb
	num := sb.GetNextFreeInodeNumber()
	ino.attr.InodeNumber = num
	sb.setInodeBitmap(int(num), true)
	buf := unifiedBuffer{}
	buf.Init(BlockSize, ino)
	// a struct to store dir
	buf.Write(makeDirData(".", ino.attr.InodeNumber))
	buf.Write(makeDirData("..", parent))
	sb.WriteInode(int(num), ino.attr)
	return num
}

// 修改目录的attr，并在
func (newInode *Ext0Inode) initAsOrdinaryFile(sb *Ext0SuperBlock) uint16 {
	newInode.attr = sb.initAttr()
	newInode.attr.FileType = u.OrdinaryFile
	newInode.attr.BlockCount = 0
	newInode.sb = sb
	num := sb.GetNextFreeInodeNumber()
	newInode.attr.InodeNumber = num
	sb.setInodeBitmap(int(num), true)
	newInode.attr.InodeNumber = num
	sb.WriteInode(int(num), newInode.attr)
	return num

}

func (ino *Ext0Inode) createChild(name string, num uint16) {
	attr := ino.attr
	sb := ino.sb
	// 确保不会创建重名文件
	if ino.LookUp(name) != 0 || attr.FileType != u.Directory {
		return
	}
	var buf unifiedBuffer
	buf.Init(BlockSize, ino)

	size := attr.Size

	if size%DirStorageSIze != 0 {
		return
	}
	var dentryNumber = size / DirStorageSIze

	block := buf.ReadAll()
	for i := 0; i < int(dentryNumber); i++ {
		var dirName [DirStorageSIze - 2]byte
		var dirInodeNumber uint16
		for j := 0; j < DirStorageSIze-2; j++ {
			dirName[j] = block[DirStorageSIze*i+j]
		}
		dirInodeNumber = binary.BigEndian.Uint16(block[DirStorageSIze*i+DirStorageSIze-2 : DirStorageSIze*i+DirStorageSIze])
		// 这是被删除的目录项，可以在这里创建新目录
		if dirName[0] == 0 && dirInodeNumber == 0 {
			buf.WriteAt(DirStorageSIze*i, makeDirData(name, num))
			return
		}
	}
	buf.WriteAt(int(attr.Size), makeDirData(name, num))
	sb.WriteInode(int(ino.attr.InodeNumber), ino.attr)
	return
}
func (ino *Ext0Inode) WriteAt(offset int, data []byte) int {
	attr := ino.attr
	sb := ino.sb
	if attr.FileType != u.OrdinaryFile {
		return 0
	}
	var buf unifiedBuffer
	buf.Init(BlockSize, ino)
	cnt := buf.WriteAt(offset, data)
	sb.WriteInode(int(ino.attr.InodeNumber), ino.attr)
	return cnt
}
func (ino *Ext0Inode) ReadAll() (re []byte) {
	attr := ino.attr

	if attr.FileType != u.OrdinaryFile {
		return
	}
	var buf unifiedBuffer
	buf.Init(BlockSize, ino)
	return buf.ReadAll()

}
func (ino *Ext0Inode) Append(data string) int {
	attr := ino.attr
	sb := ino.sb
	if attr.FileType != u.OrdinaryFile {
		return 0
	}
	var buf unifiedBuffer
	buf.Init(BlockSize, ino)
	cnt := buf.Write([]byte(data))
	sb.WriteInode(int(ino.attr.InodeNumber), ino.attr)
	return cnt
}
func (ino *Ext0Inode) Resize(newSize int) {
	buf := unifiedBuffer{}
	buf.Init(BlockSize, ino)
	buf.resize(newSize)
}
func (ino *Ext0Inode) Remove(name string) (ok bool) {

	childNum := ino.LookUp(name)

	if childNum == 0 {
		fmt.Println("can't find child with name ", name)
	}
	dir := ino.sb.ReadDir(ino.attr)

	var buf unifiedBuffer
	buf.Init(BlockSize, ino)
	emptyByte := make([]byte, DirStorageSIze)

	for i, d := range dir {

		if d.inodeNumber == uint16(childNum) {

			// 找到这个inode,将它在目录中的存储清空

			// 从inode表读取inode信息
			cino := ino.sb.ReadInode(int(childNum))
			if cino.GetAttr().FileType == u.OrdinaryFile {
				// 普通文件，前往超级块删除inode信息和数据信息
				ino.sb.destroyInode(cino.(*Ext0Inode))
				buf.WriteAt(i*DirStorageSIze, emptyByte)
				return true
			} else {
				ino.sb.destroyDir(cino.(*Ext0Inode))
				buf.WriteAt(i*DirStorageSIze, emptyByte)
				return true
			}
		}
	}

	return
}
func (ino *Ext0Inode) SetSb(sb vfs.SuperBlock) {
	ino.sb = sb.(*Ext0SuperBlock)
}
func (ino *Ext0Inode) GetSb() vfs.SuperBlock {
	return ino.sb
}
