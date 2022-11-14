package rclone

import (
	"context"
	"strings"
	"time"

	"github.com/darkhz/rclone-tui/rclone"
)

// Mounts stores the list of mountpoints.
type Mounts struct {
	MountPoints []MountPoint `json:"mountPoints"`
}

// MountPoint stores information about a mountpoint.
type MountPoint struct {
	Fs         string    `json:"Fs"`
	MountPoint string    `json:"MountPoint"`
	MountedOn  time.Time `json:"MountedOn"`
}

// MountHelp stores information about a mount option.
type MountHelp struct {
	Name, Help, OptionType, ValueType string
	Windows, OSX, Other               bool

	Options []string
}

var mountOptionHelp = []MountHelp{
	{"FS", "A remote path to be mounted", "main:required", "string", true, true, true, nil},
	{"MountPoint", "Valid path on the local machine", "main:required", "string", true, true, true, nil},
	{"MountType", "One of the values (mount, cmount, mount2) specifies the mount implementation to use", "main:optional", "string", true, true, true, nil},
	{"AllowNonEmpty", "Allow mounting over a non-empty directory", "mountOpt", "bool", false, true, true, nil},
	{"AllowOther", "Allow access to other users", "mountOpt", "bool", false, true, true, nil},
	{"AllowRoot", "Allow access to root user", "mountOpt", "bool", false, true, true, nil},
	{"AsyncRead", "Use asynchronous reads", "mountOpt", "bool", false, true, true, nil},
	{"AttrTimeout", "Time for which file/directory attributes are cached", "mountOpt", "Duration", true, true, true, nil},
	{"CacheMaxAge", "Max age of objects in the cache", "vfsOpt", "Duration", true, true, true, nil},
	{"CacheMaxSize", "Max total size of objects in the cache", "vfsOpt", "int", true, true, true, nil},
	{"CacheMode", "Cache mode off|minimal|writes|full", "vfsOpt", "int", true, true, true, nil},
	{"CachePollInterval", "Interval to poll the cache for stale objects", "vfsOpt", "Duration", true, true, true, nil},
	{"CaseInsensitive", "If a file name not found, find a case insensitive match", "vfsOpt", "bool", true, true, true, nil},
	{"ChunkSize", "Read the source objects in chunks", "vfsOpt", "int", true, true, true, nil},
	{"ChunkSizeLimit", "If greater than --vfs-read-chunk-size, double the chunk size after each chunk read, until the limit is reached ('off' is unlimited)", "vfsOpt", "int", true, true, true, nil},
	{"Daemon", "Run mount in background and exit parent process (as background output is suppressed, use --log-file with --log-format=pid,... to monitor)", "mountOpt", "bool", false, true, true, nil},
	{"DaemonTimeout", "Time limit for rclone to respond to kernel", "mountOpt", "Duration", false, true, true, nil},
	{"DaemonWait", "Time to wait for ready mount from daemon (maximum time on Linux, constant sleep time on OSX/BSD)", "mountOpt", "Duration", false, true, true, nil},
	{"DebugFUSE", "Debug the FUSE internals", "mountOpt", "bool", true, true, true, nil},
	{"DefaultPermissions", "Makes kernel enforce access control based on the file mode", "mountOpt", "bool", false, true, true, nil},
	{"DeviceName", "Set the device name - default is remote:path", "mountOpt", "string", true, true, true, nil},
	{"DirCacheTime", "Time to cache directory entries for", "vfsOpt", "Duration", true, true, true, nil},
	{"DirPerms", "Directory permissions", "vfsOpt", "int", true, true, true, nil},
	{"DiskSpaceTotalSize", "Specify the total space of disk", "vfsOpt", "int", true, true, true, nil},
	{"ExtraFlags", "Flags or arguments to be passed direct to libfuse/WinFsp (repeat if required). Each mount option must be separated by a space.", "mountOpt", "StringArray", true, true, true, nil},
	{"ExtraOptions", "Option for libfuse/WinFsp (repeat if required). Each mount option must be separated by a space.", "mountOpt", "StringArray", true, true, true, nil},
	{"FastFingerprint", "Use fast (less accurate) fingerprints for change detection", "vfsOpt", "bool", true, true, true, nil},
	{"FilePerms", "File permissions", "vfsOpt", "int", true, true, true, nil},
	{"MaxReadAhead", "The number of bytes that can be prefetched for sequential reads", "mountOpt", "int", false, true, true, nil},
	{"NetworkMode", "Mount as remote network drive, instead of fixed disk drive", "mountOpt", "bool", true, false, false, nil},
	{"NoAppleDouble", "Ignore Apple Double (._) and .DS_Store files", "mountOpt", "bool", false, true, false, nil},
	{"NoAppleXattr", "Ignore all \"com.apple.*\" extended attributes", "mountOpt", "bool", false, true, false, nil},
	{"NoChecksum", "Don't compare checksums on up/download", "vfsOpt", "bool", true, true, true, nil},
	{"NoModTime", "Don't read/write the modification time (can speed things up)", "vfsOpt", "bool", true, true, true, nil},
	{"NoSeek", "Don't allow seeking in files", "vfsOpt", "bool", true, true, true, nil},
	{"PollInterval", "Time to wait between polling for changes, must be smaller than dir-cache-time and only on supported remotes (set 0 to disable)", "vfsOpt", "Duration", true, true, true, nil},
	{"ReadAhead", "Extra read ahead over --buffer-size when using cache-mode full", "vfsOpt", "int", true, true, true, nil},
	{"ReadOnly", "Only allow read-only access", "vfsOpt", "bool", true, true, true, nil},
	{"ReadWait", "Time to wait for in-sequence read before seeking", "vfsOpt", "Duration", true, true, true, nil},
	{"UsedIsSize", "Use the `rclone size` algorithm for Used size", "vfsOpt", "bool", true, true, true, nil},
	{"VolumeName", "Set the volume name", "mountOpt", "string", true, true, false, nil},
	{"WriteBack", "Time to writeback files after last use when using cache", "vfsOpt", "Duration", true, true, true, nil},
	{"WriteWait", "Time to wait for in-sequence write before giving error", "vfsOpt", "Duration", true, true, true, nil},
	{"WritebackCache", "Makes kernel buffer writes before sending them to rclone (without this, writethrough caching is used)", "mountOpt", "bool", false, true, true, nil},
}

// CreateMount creates a mountpoint.
func CreateMount(mountData map[string]interface{}) error {
	parseMountExtras(mountData)

	job, err := rclone.SendCommandAsync("UI:Mounts", "Mounting remote", mountData, "/mount/mount")
	if err != nil {
		return err
	}

	_, err = rclone.GetJobReply(job)

	return err
}

// Unmount unmounts the provided mountpoint.
func Unmount(mountpoint string) error {
	command := map[string]interface{}{
		"mountPoint": mountpoint,
	}

	job, err := rclone.SendCommandAsync("UI:Mounts", "Unmounting mountpoint", command, "/mount/unmount")
	if err != nil {
		return err
	}

	_, err = rclone.GetJobReply(job)

	return err
}

// UnmountAll unmounts all mountpoints.
func UnmountAll() error {
	job, err := rclone.SendCommandAsync(
		"UI:Mounts", "Unmounting all mountpoints",
		map[string]interface{}{}, "/mount/unmountall",
	)
	if err != nil {
		return err
	}

	_, err = rclone.GetJobReply(job)

	return err
}

// ListMountTypes lists the mount types.
func ListMountTypes(ctx context.Context) ([]string, error) {
	return GetDataSlice(ctx, "/mount/types", "mountTypes")
}

// GetMountPoints returns a list of mountpoints.
func GetMountPoints() ([]MountPoint, error) {
	var mounts Mounts

	response, err := rclone.SendCommand(map[string]interface{}{}, "/mount/listmounts")
	if err != nil {
		return nil, err
	}

	err = response.Decode(&mounts)

	return mounts.MountPoints, err
}

// GetMountHelp returns the information for a mount option.
func GetMountHelp(name string) MountHelp {
	var mountHelp MountHelp

	for _, mh := range mountOptionHelp {
		if mh.Name == name {
			mountHelp = mh
			break
		}
	}

	return mountHelp
}

// GetMountOptions returns a list of all mount options.
func GetMountOptions() ([]MountHelp, error) {
	var opts []MountHelp

	remotes, err := ListRemotes(rclone.GetClientContext())
	if err != nil {
		return nil, err
	}

	mountTypes, err := ListMountTypes(rclone.GetClientContext())
	if err != nil {
		return nil, err
	}

	version, _ := rclone.GetVersion(false)

	for _, opt := range mountOptionHelp {
		switch opt.Name {
		case "FS":
			opt.Options = remotes

		case "MountType":
			opt.Options = mountTypes
		}

		windowsOnly := version.Os == "windows" && opt.Windows
		osxOnly := (version.Os == "darwin" || version.Os == "ios") && opt.OSX

		if windowsOnly || osxOnly || (!windowsOnly && !osxOnly) && opt.Other {
			opts = append(opts, opt)
		}
	}

	mountOptionHelp = opts

	return mountOptionHelp, nil
}

// parseMountExtras parses the 'ExtraFlags' and 'ExtraOptions' mount options.
func parseMountExtras(mountData map[string]interface{}) {
	mountMap, ok := mountData["mountOpt"]
	if !ok {
		return
	}

	mountOpts, ok := mountMap.(map[string]interface{})
	if !ok {
		return
	}

	for _, mountFlag := range []string{
		"ExtraFlags",
		"ExtraOptions",
	} {
		if value, ok := mountOpts[mountFlag]; ok && value != nil {
			if opts, ok := value.(string); ok {
				mountOpts[mountFlag] = strings.Split(opts, " ")
			}
		}
	}
}
