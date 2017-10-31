// +build darwin,cgo

package disk

import (
	"fmt"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

/*
#cgo LDFLAGS: -framework DiskArbitration -framework CoreFoundation -framework IOKit
#include <DiskArbitration/DASession.h>
#include <DiskArbitration/DADisk.h>
#include <IOKit/Storage/IOBlockStorageDriver.h>
#include <mach/mach_error.h>
*/
import "C"

var systemHz uint64 = uint64(C.sysconf(C._SC_CLK_TCK))

type kernelReturn C.kern_return_t

// diskMetrics contains I/O data about a specific disk partition
// It is the interpreted output of statfs (https://www.kernel.org/doc/Documentation/block/stat.txt)
type diskMetrics struct {
	sampleTime int64
	name       string
	readCount  int64
	readTime   int64
	writeCount int64
	writeTime  int64
}

func getDiskMetrics() (map[string]IOCountersStat, error) {
	partitions, err := Partitions(false)
	if err != nil {
		return nil, fmt.Errorf("Error getting disk partitions: %v", err)
	}

	counters := make(map[string]IOCountersStat)
	for _, partition := range partitions {
		metrics, err := getSingleDiskMetrics(partition.Mountpoint)
		if err != nil && !strings.Contains(err.Error(), "Failed to find IOBlockStorageDriver for device") {
			return nil, fmt.Errorf("Error getting disk metrics for partition %v: %v", partition.Device, err)
		}
		if metrics == nil {
			continue
		}

		counters[partition.Device] = IOCountersStat{
			Name:       metrics.name,
			ReadCount:  uint64(metrics.readCount),
			ReadTime:   uint64(metrics.readTime),
			WriteCount: uint64(metrics.writeCount),
			WriteTime:  uint64(metrics.writeTime),
		}
	}

	return counters, nil
}

func getSingleDiskMetrics(filePath string) (*diskMetrics, error) {
	diskMetrics := &diskMetrics{
		sampleTime: time.Now().UnixNano() / 1e6,
	}

	var statfs syscall.Statfs_t
	if err := syscall.Statfs(filePath, &statfs); err != nil {
		return nil, fmt.Errorf("Error getting disk metrics for filePath = %v : Failed to statfs : %v", filePath, err)
	}

	var session C.DASessionRef = C.DASessionRef(C.DASessionCreate(C.kCFAllocatorDefault))
	defer C.CFRelease(C.CFTypeRef(session))

	disk := C.DADiskCreateFromBSDName(C.kCFAllocatorDefault, session, (*C.char)(unsafe.Pointer(&statfs.Mntfromname[0])))
	defer C.CFRelease(C.CFTypeRef(disk))

	ioService := C.DADiskCopyIOMedia(disk)
	defer C.IOObjectRelease(C.io_object_t(ioService))

	var match bool
	lastMatchedBsdName := C.GoString((*C.char)(unsafe.Pointer(&statfs.Mntfromname[0])))
	for i := 1; ioService != 0; i++ {

		var parent C.io_registry_entry_t
		if status := C.IORegistryEntryGetParentEntry(C.io_registry_entry_t(ioService), C.CString(C.kIOServicePlane), &parent); status != C.KERN_SUCCESS {
			return nil, fmt.Errorf("Error getting disk metrics for filePath = %v : Failed to get parent entry for io service: %v : %v. Status = %v", filePath, ioService, getMachErrorStringFromKernReturn(kernelReturn(status)), status)
		}

		var properties C.CFDictionaryRef
		if status := C.IORegistryEntryCreateCFProperties(parent, (*C.CFMutableDictionaryRef)(&properties), C.kCFAllocatorDefault, C.kNilOptions); status != C.KERN_SUCCESS {
			return nil, fmt.Errorf("Error getting disk metrics for filePath = %v : Failed to get properties for io service: %+v : %v. Status = %v, parent = %v", filePath, ioService, getMachErrorStringFromKernReturn(kernelReturn(status)), status, parent)
		}
		defer C.CFRelease(C.CFTypeRef(properties))

		var valRef C.CFDictionaryRef
		valRefPtr := unsafe.Pointer(valRef)
		ret := C.CFDictionaryGetValueIfPresent(properties, unsafe.Pointer(C.CFStringCreateWithCString(C.kCFAllocatorDefault, C.CString("BSD Name"), C.kCFStringEncodingUTF8)), &valRefPtr)
		if ret == 1 {
			cov := C.CFStringGetCStringPtr((C.CFStringRef)(unsafe.Pointer(valRefPtr)), C.kCFStringEncodingUTF8)
			lastMatchedBsdName = C.GoString(cov)
		}

		if C.IOObjectConformsTo(C.io_object_t(parent), C.CString("IOBlockStorageDriver")) != 0 {
			match = true
			C.IOObjectRelease(C.io_object_t(ioService))
			ioService = C.io_service_t(parent)
			break
		}

		C.IOObjectRelease(C.io_object_t(ioService))
		ioService = C.io_service_t(parent)
	}

	if !match {
		return nil, fmt.Errorf("Error getting disk metrics for filePath = %v : Failed to find IOBlockStorageDriver for device %s", filePath, C.GoString((*C.char)(unsafe.Pointer(&statfs.Mntfromname[0]))))
	}

	diskMetrics.name = lastMatchedBsdName

	var properties C.CFDictionaryRef
	if status := C.IORegistryEntryCreateCFProperties(C.io_registry_entry_t(ioService), (*C.CFMutableDictionaryRef)(&properties), C.kCFAllocatorDefault, C.kNilOptions); status != C.KERN_SUCCESS {
		return nil, fmt.Errorf("Error getting disk metrics for filePath = %v : Failed to get properties for io service: %+v : %v. Status = %v", filePath, ioService, getMachErrorStringFromKernReturn(kernelReturn(status)), status)
	}
	defer C.CFRelease(C.CFTypeRef(properties))

	var statistics C.CFDictionaryRef
	if ret := C.CFDictionaryGetValueIfPresent(properties, unsafe.Pointer(C.CFStringCreateWithCString(C.kCFAllocatorDefault, C.CString(C.kIOBlockStorageDriverStatisticsKey), C.kCFStringEncodingUTF8)), (*unsafe.Pointer)(unsafe.Pointer(&statistics))); ret != 1 {
		return nil, fmt.Errorf("Error getting disk metrics for filePath = %v : Failed to get statistics for %s. Ret = %v", filePath, lastMatchedBsdName, ret)
	}

	var value uint64
	var number C.CFNumberRef
	if ret := C.CFDictionaryGetValueIfPresent(statistics, unsafe.Pointer(C.CFStringCreateWithCString(C.kCFAllocatorDefault, C.CString(C.kIOBlockStorageDriverStatisticsReadsKey), C.kCFStringEncodingUTF8)), (*unsafe.Pointer)(unsafe.Pointer(&number))); ret == 1 {
		C.CFNumberGetValue(number, C.kCFNumberSInt64Type, unsafe.Pointer(&value))
		diskMetrics.readCount = int64(value)
	}

	if ret := C.CFDictionaryGetValueIfPresent(statistics, unsafe.Pointer(C.CFStringCreateWithCString(C.kCFAllocatorDefault, C.CString(C.kIOBlockStorageDriverStatisticsWritesKey), C.kCFStringEncodingUTF8)), (*unsafe.Pointer)(unsafe.Pointer(&number))); ret == 1 {
		C.CFNumberGetValue(number, C.kCFNumberSInt64Type, unsafe.Pointer(&value))
		diskMetrics.writeCount = int64(value)
	}

	if ret := C.CFDictionaryGetValueIfPresent(statistics, unsafe.Pointer(C.CFStringCreateWithCString(C.kCFAllocatorDefault, C.CString(C.kIOBlockStorageDriverStatisticsTotalReadTimeKey), C.kCFStringEncodingUTF8)), (*unsafe.Pointer)(unsafe.Pointer(&number))); ret == 1 {
		C.CFNumberGetValue(number, C.kCFNumberSInt64Type, unsafe.Pointer(&value))
		diskMetrics.readTime = int64(value / 1e6)
	}

	if ret := C.CFDictionaryGetValueIfPresent(statistics, unsafe.Pointer(C.CFStringCreateWithCString(C.kCFAllocatorDefault, C.CString(C.kIOBlockStorageDriverStatisticsTotalWriteTimeKey), C.kCFStringEncodingUTF8)), (*unsafe.Pointer)(unsafe.Pointer(&number))); ret == 1 {
		C.CFNumberGetValue(number, C.kCFNumberSInt64Type, unsafe.Pointer(&value))
		diskMetrics.writeTime = int64(value / 1e6)
	}
	return diskMetrics, nil
}

func getMachErrorStringFromKernReturn(ret kernelReturn) string {
	return getMachErrorString(C.mach_error_t(ret))
}

func getMachErrorString(err C.mach_error_t) string {
	errStringRaw := C.mach_error_string(C.mach_error_t(err))
	return C.GoString(errStringRaw)
}
