package drive

import (
	"fmt"
	"io"

	"google.golang.org/api/drive/v3"
)

type FileInfoArgs struct {
	Out         io.Writer
	Id          string
	SizeInBytes bool
}

func (args *FileInfoArgs) normalize(drive *Drive) {
	finder := drive.newPathFinder()
	args.Id = finder.SecureFileId(args.Id)
}

func (self *Drive) Info(args FileInfoArgs) error {
	args.normalize(self)

	f, err := self.service.Files.Get(args.Id).Fields("id", "name", "size", "createdTime", "modifiedTime", "md5Checksum", "mimeType", "parents", "shared", "description", "webContentLink", "webViewLink").Do()
	if err != nil {
		return fmt.Errorf("Failed to get file: %s", err)
	}

	finder := self.newPathFinder()
	absPath, err := finder.GetAbsPath(f)
	if err != nil {
		return err
	}

	PrintFileInfo(PrintFileInfoArgs{
		Out:         args.Out,
		File:        f,
		Path:        absPath,
		SizeInBytes: args.SizeInBytes,
	})

	return nil
}

type PrintFileInfoArgs struct {
	Out         io.Writer
	File        *drive.File
	Path        string
	SizeInBytes bool
}

func PrintFileInfo(args PrintFileInfoArgs) {
	f := args.File

	items := []kv{
		kv{"Id", f.Id},
		kv{"Name", f.Name},
		kv{"Path", args.Path},
		kv{"Description", f.Description},
		kv{"Mime", f.MimeType},
		kv{"Size", formatSize(f.Size, args.SizeInBytes)},
		kv{"Created", formatDatetime(f.CreatedTime)},
		kv{"Modified", formatDatetime(f.ModifiedTime)},
		kv{"Md5sum", f.Md5Checksum},
		kv{"Shared", formatBool(f.Shared)},
		kv{"Parents", formatList(f.Parents)},
		kv{"ViewUrl", f.WebViewLink},
		kv{"DownloadUrl", f.WebContentLink},
	}

	for _, item := range items {
		if item.value != "" {
			fmt.Fprintf(args.Out, "%s: %s\n", item.key, item.value)
		}
	}
}
