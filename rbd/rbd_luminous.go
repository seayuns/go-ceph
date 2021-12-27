// +build luminous, !mimic !nautilus

package rbd

// #cgo LDFLAGS: -lrbd
// #include <rados/librados.h>
// #include <rbd/librbd.h>
// #include <errno.h>
import "C"
import ts "github.com/ceph/go-ceph/internal/timespec"

// GetCreateTimestamp returns the time the rbd image was created.
//
// Implements:
//  int rbd_get_create_timestamp(rbd_image_t image, struct timespec *timestamp);
func (image *Image) GetCreateTimestamp() (Timespec, error) {
	if err := image.validate(imageIsOpen); err != nil {
		return Timespec{}, err
	}

	var cts C.struct_timespec

	if ret := C.rbd_get_create_timestamp(image.image, &cts); ret < 0 {
		return Timespec{}, getError(ret)
	}

	return Timespec(ts.CStructToTimespec(ts.CTimespecPtr(&cts))), nil
}
