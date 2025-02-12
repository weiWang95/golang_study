package index

import (
	"os"
)

type Data struct {
	f *os.File
}

func NewData(path string) (*Data, error) {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return nil, err
	}

	return &Data{f: f}, nil
}

func (d *Data) WriteAt(data []byte, off int64) error {
	_, err := d.f.WriteAt(data, off)
	return err
}

func (d *Data) ReadNodeAt(off int64) (data []byte, err error) {
	p := make([]byte, NodeCap)
	i, err := d.f.ReadAt(p, off)
	if err != nil {
		return nil, err
	}
	if i == 0 {
		return nil, nil
	}
	return p, nil
}

func (d *Data) ReadAt(data []byte, off int64) (int, error) {
	return d.f.ReadAt(data, off)
}

func (d *Data) Close() error {
	return d.f.Close()
}
