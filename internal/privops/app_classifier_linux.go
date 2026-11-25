package privops

import (
	"errors"
	"fmt"
	"math"
	"os"
	"strconv"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/asm"
	"github.com/cilium/ebpf/btf"
	"github.com/cilium/ebpf/link"
	"golang.org/x/sys/unix"
)

type appSocketKey struct {
	inode  uint64
	device uint32
	uid    uint32
}

type appSocketClassifier struct {
	marks *ebpf.Map
	link  link.Link
}

func openAppSocketClassifier(cgroup string) (*appSocketClassifier, error) {
	// The kernel pointer walk below reads pointers as eight-byte fields.
	if strconv.IntSize != 64 {
		return nil, fmt.Errorf("%w: executable classification requires 64-bit Linux", ErrUnsupported)
	}
	target, err := os.Open(cgroup)
	if err != nil {
		return nil, err
	}
	defer target.Close()
	spec, err := btf.LoadKernelSpec()
	if err != nil {
		return nil, fmt.Errorf("read kernel BTF: %w", err)
	}
	offsets, err := appClassifierOffsets(spec)
	if err != nil {
		return nil, err
	}
	marks, err := ebpf.NewMap(&ebpf.MapSpec{
		Name: "kongtrol_apps", Type: ebpf.Hash, KeySize: 16, ValueSize: 4, MaxEntries: 4096,
	})
	if err != nil {
		return nil, err
	}
	program, err := ebpf.NewProgram(&ebpf.ProgramSpec{
		Name: "kongtrol_app", Type: ebpf.CGroupSock, AttachType: ebpf.AttachCGroupInetSockCreate,
		License: "GPL", Instructions: appMarkInstructions(marks.FD(), offsets),
	})
	if err != nil {
		marks.Close()
		return nil, err
	}
	defer program.Close()
	attachment, err := link.AttachRawLink(link.RawLinkOptions{
		Target: int(target.Fd()), Program: program, Attach: ebpf.AttachCGroupInetSockCreate,
	})
	if err != nil {
		marks.Close()
		return nil, err
	}
	return &appSocketClassifier{marks: marks, link: attachment}, nil
}

func (c *appSocketClassifier) Close() error {
	return errors.Join(c.link.Close(), c.marks.Close())
}

type appSocketOffsets struct {
	taskMM, mmExeFile, fileInode, inodeNumber, inodeSuperBlock, superBlockDevice int32
	inodeSize                                                                    asm.Size
}

func appClassifierOffsets(spec *btf.Spec) (appSocketOffsets, error) {
	var findMember func(*btf.Struct, string) (btf.Bits, btf.Type)
	findMember = func(typ *btf.Struct, name string) (btf.Bits, btf.Type) {
		for _, member := range typ.Members {
			if member.Name == name && member.BitfieldSize == 0 {
				return member.Offset, member.Type
			}
			if member.Name == "" {
				if nested, ok := btf.UnderlyingType(member.Type).(*btf.Struct); ok {
					if offset, typ := findMember(nested, name); typ != nil {
						return member.Offset + offset, typ
					}
				}
			}
		}
		return 0, nil
	}
	field := func(structure, name string) (int32, btf.Type, error) {
		var typ *btf.Struct
		if err := spec.TypeByName(structure, &typ); err != nil {
			return 0, nil, err
		}
		offset, member := findMember(typ, name)
		if member != nil && offset.Bytes() <= math.MaxInt32 {
			return int32(offset.Bytes()), member, nil
		}
		return 0, nil, fmt.Errorf("unrecognized kernel BTF field %s.%s", structure, name)
	}
	var offsets appSocketOffsets
	for _, item := range []struct {
		structure, name string
		dest            *int32
	}{
		{"task_struct", "mm", &offsets.taskMM},
		{"mm_struct", "exe_file", &offsets.mmExeFile},
		{"file", "f_inode", &offsets.fileInode},
		{"inode", "i_sb", &offsets.inodeSuperBlock},
		{"super_block", "s_dev", &offsets.superBlockDevice},
	} {
		offset, _, err := field(item.structure, item.name)
		if err != nil {
			return offsets, err
		}
		*item.dest = offset
	}
	offset, typ, err := field("inode", "i_ino")
	if err != nil {
		return offsets, err
	}
	size, err := btf.Sizeof(typ)
	if err != nil {
		return offsets, err
	}
	offsets.inodeNumber = offset
	switch size {
	case 4:
		offsets.inodeSize = asm.Word
	case 8:
		offsets.inodeSize = asm.DWord
	default:
		return offsets, errors.New("unrecognized kernel inode number size")
	}
	return offsets, nil
}

func appMarkInstructions(mapFD int, offsets appSocketOffsets) asm.Instructions {
	const (
		scratch = -24
		key     = -16
		mark    = 16 // offsetof(struct bpf_sock, mark) in linux/bpf.h
	)
	// Resolve current->mm->exe_file->f_inode using offsets from this kernel's
	// BTF. The cgroup is only an attachment point, so children opening their
	// own sockets are classified by their own executable.
	read := func(dest int16, base asm.Register, offset int32, size asm.Size) asm.Instructions {
		return asm.Instructions{
			asm.JEq.Imm(base, 0, "allow"),
			asm.Mov.Reg(asm.R3, base),
			asm.Add.Imm(asm.R3, offset),
			asm.Mov.Reg(asm.R1, asm.R10),
			asm.Add.Imm(asm.R1, int32(dest)),
			asm.Mov.Imm(asm.R2, int32(size.Sizeof())),
			asm.FnProbeReadKernel.Call(),
			asm.JNE.Imm(asm.R0, 0, "allow"),
		}
	}
	ins := asm.Instructions{
		asm.Mov.Reg(asm.R6, asm.R1),
		asm.FnGetCurrentTask.Call(),
	}
	for _, offset := range []int32{offsets.taskMM, offsets.mmExeFile, offsets.fileInode} {
		ins = append(ins, read(scratch, asm.R0, offset, asm.DWord)...)
		ins = append(ins, asm.LoadMem(asm.R0, asm.R10, scratch, asm.DWord))
	}
	ins = append(ins, asm.Mov.Reg(asm.R7, asm.R0))
	ins = append(ins, read(scratch, asm.R7, offsets.inodeNumber, offsets.inodeSize)...)
	ins = append(ins,
		asm.LoadMem(asm.R0, asm.R10, scratch, offsets.inodeSize),
		asm.StoreMem(asm.R10, key, asm.R0, asm.DWord),
	)
	ins = append(ins, read(scratch, asm.R7, offsets.inodeSuperBlock, asm.DWord)...)
	ins = append(ins, asm.LoadMem(asm.R0, asm.R10, scratch, asm.DWord))
	ins = append(ins, read(scratch, asm.R0, offsets.superBlockDevice, asm.Word)...)
	ins = append(ins,
		asm.LoadMem(asm.R0, asm.R10, scratch, asm.Word),
		asm.StoreMem(asm.R10, key+8, asm.R0, asm.Word),
		asm.FnGetCurrentUidGid.Call(),
		asm.StoreMem(asm.R10, key+12, asm.R0, asm.Word),
		asm.LoadMapPtr(asm.R1, mapFD),
		asm.Mov.Reg(asm.R2, asm.R10),
		asm.Add.Imm(asm.R2, key),
		asm.FnMapLookupElem.Call(),
		asm.JEq.Imm(asm.R0, 0, "allow"),
		asm.LoadMem(asm.R1, asm.R0, 0, asm.Word),
		asm.StoreMem(asm.R6, mark, asm.R1, asm.Word),
		asm.Mov.Imm(asm.R0, 1).WithSymbol("allow"),
		asm.Return(),
	)
	return ins
}

func executableSocketKey(path string, uid uint32) (appSocketKey, error) {
	var st unix.Stat_t
	if err := unix.Stat(path, &st); err != nil {
		return appSocketKey{}, err
	}
	if st.Mode&unix.S_IFMT != unix.S_IFREG || st.Mode&0111 == 0 {
		return appSocketKey{}, errors.New("path does not identify an executable file")
	}
	return appSocketKey{
		inode:  st.Ino,
		device: unix.Major(st.Dev)<<20 | unix.Minor(st.Dev),
		uid:    uid,
	}, nil
}
