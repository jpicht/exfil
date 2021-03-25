package exfil

import (
	"archive/tar"
	"errors"
	"fmt"
	"hash/crc32"
	"os"
	"path"
	"sort"
	"strconv"
	"sync"
	"time"
)

var (
	ERR_EXISTS        = errors.New("Id already in use")
	ERR_INCOMPLETE    = errors.New("File is incomplete")
	ERR_NOT_DECLARED  = errors.New("Id not declared")
	ERR_OUT_OF_BOUNDS = errors.New("Offset out of bounds")
)

type (
	// file represents the transfer state of a single file
	file struct {
		lock      sync.Mutex
		name      string
		size      uint32
		timestamp time.Time
		chunks    map[uint32][]byte
	}
	// receiver manages the files that are currently in transfer
	//
	// Todo:
	//  - write chunks to disk to allow for
	//    - server restarts
	//    - transfer of files that do not fit into RAM
	receiver struct {
		lock  sync.Mutex
		out   string
		files map[uint32]*file
	}
)

// NewReceiver creates a receiver object
func NewReceiver(dir string) *receiver {
	return &receiver{
		out:   dir,
		files: make(map[uint32]*file),
	}
}

// AddFile registers a new file for transfer
func (r *receiver) AddFile(id, size uint32, name string) error {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("AddFile Recovered", r)
		}
	}()

	r.lock.Lock()
	defer r.lock.Unlock()

	if f, ok := r.files[id]; ok {
		if name != f.name || size != f.size {
			log.Infof("File exists: %08x %s", id, f.name)
			return ERR_EXISTS
		}
		log.Infof("Override file: %08x %s", id, name)
	}

	log.Infof("New file: %08x %s", id, name)
	r.files[id] = &file{
		name:      name,
		size:      size,
		timestamp: time.Now(),
		chunks:    make(map[uint32][]byte),
	}

	return nil
}

// AddData stores a file chunk in the corresponding file object
func (r *receiver) AddData(id, offset uint32, data []byte) error {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("AddData Recovered", r)
		}
	}()

	r.lock.Lock()
	f, ok := r.files[id]
	r.lock.Unlock()

	if !ok {
		log.Infof("Data for unknown file: %08x", id)
		return ERR_NOT_DECLARED
	}

	// cast to int because otherwise uint32 could overflow
	if int(offset)+len(data) > int(f.size) {
		return ERR_OUT_OF_BOUNDS
	}

	log.Infof("Data for file: %08x offset 0x%x", id, offset)
	err := f.Add(offset, data)
	if err != nil {
		return err
	}

	if f.Complete() {
		log.Infof("File complete: %08x %s %d", id, f.name, f.size)
		err = f.ToDisk(id, r.out)
		if err != nil {
			return err
		}

		r.lock.Lock()
		delete(r.files, id)
		r.lock.Unlock()
	}

	return nil
}

// Add a chunk to this file
func (f *file) Add(offset uint32, data []byte) error {
	f.lock.Lock()
	defer f.lock.Unlock()

	if _, ok := f.chunks[offset]; ok {
		return ERR_EXISTS
	}

	f.chunks[offset] = data

	return nil
}

// Complete checks if this file is transfered completely
func (f *file) Complete() bool {
	f.lock.Lock()
	defer f.lock.Unlock()
	offsets := f.offsets()
	var pos uint32 = 0
	for _, offset := range offsets {
		if offset != pos {
			return false
		}
		pos += uint32(len(f.chunks[uint32(offset)]))
	}

	return uint32(pos) == f.size
}

// Uint32Slize to make []uint32 sortable
type Uint32Slize []uint32

func (p Uint32Slize) Len() int           { return len(p) }
func (p Uint32Slize) Less(i, j int) bool { return p[i] < p[j] }
func (p Uint32Slize) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

// offsets returns all offsets in ascending order
func (f *file) offsets() []uint32 {
	offsets := make([]uint32, 0, len(f.chunks))
	for offset := range f.chunks {
		offsets = append(offsets, offset)
	}
	sort.Sort(Uint32Slize(offsets))
	return offsets
}

// ToDisk writes the file to the disk
//
// The file is wrapped in a tar archive, which would easily allow to transfer
// all file attributes (including ownership) and metadata without the need to
// run coredns as root.
func (f *file) ToDisk(id uint32, baseDir string) error {
	if !f.Complete() {
		return ERR_INCOMPLETE
	}

	f.lock.Lock()
	defer f.lock.Unlock()

	// create a sane file name
	base := path.Base(f.name)
	if base == "/" || base == "." {
		base = ""
	} else {
		base = "_" + base
	}
	name := path.Join(baseDir, f.timestamp.Format("20060102T150405")+"_"+strconv.Itoa(int(id))+base+".tar")

	fh, err := os.OpenFile(name, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer fh.Close()

	t := tar.NewWriter(fh)
	defer t.Close()

	header := &tar.Header{
		Name:    f.name,
		Size:    int64(f.size),
		Mode:    int64(0644),
		ModTime: f.timestamp,
	}

	err = t.WriteHeader(header)
	if err != nil {
		return err
	}

	crc := crc32.NewIEEE()
	for _, offset := range f.offsets() {
		t.Write(f.chunks[uint32(offset)])
		crc.Write(f.chunks[uint32(offset)])
	}

	log.Infof("Checksum for %08x == %08x", id, crc.Sum32()&0x7fffffff)

	if id == crc.Sum32()&0x7fffffff {
		log.Info("Checksum OK")
		return nil
	}

	log.Infof("Checksum mismatch %08x != %08x", id, crc.Sum32()&0x7fffffff)

	// fixme: return error
	return nil
}
