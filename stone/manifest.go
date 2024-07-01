package stone

import (
	"fmt"
	"os"
	"strings"

	"github.com/GZGavinZhao/autobuild/common"
	"github.com/serpent-os/libstone-go"
	"github.com/serpent-os/libstone-go/stone1"
)

// func getCompressionReader(r io.ReaderAt, compressionType payload.Compression, offset, length int64) (io.Reader, error) {
// 	switch compressionType {
// 	case payload.CompressionNone:
// 		return io.NewSectionReader(r, offset, length), nil
// 	case payload.CompressionZstd:
// 		return zstd.NewReader(io.NewSectionReader(r, offset, length))
// 	}
// 	return nil, errors.New("Unknown compression type")
// }

func ParseManifest(path string) (cpkg common.Package, err error) {
	file, err := os.Open(path)
	if err != nil {
		err = fmt.Errorf("Failed to open manifest %s, reason: %w", path, err)
		return
	}
	defer file.Close()

	genericPrelude, err := libstone.ReadPrelude(file)
	if err != nil {
		return
	}

	prelude, err := stone1.NewPrelude(genericPrelude)
	if err != nil {
		return
	}

	cache, err := os.CreateTemp("", "")
	if err != nil {
		err = fmt.Errorf("Failed to create temp for parsing stone file: %w", err)
		return
	}
	defer os.Remove(cache.Name())

	rdr := stone1.NewReader(prelude, file, cache)

	for rdr.NextPayload() {
		if rdr.Header.Kind != stone1.Meta {
			continue
		}

		for rdr.NextRecord() {
			switch record := rdr.Record.(type) {
			case *stone1.MetaRecord:
				switch record.Tag {
				case stone1.SourceID:
					cpkg.Name = record.Field.String()
				case stone1.Version:
					cpkg.Version = record.Field.String()
				case stone1.Release:
					cpkg.Release = int(record.Field.Value.(uint64))
				case stone1.Depends:
					cpkg.BuildDeps = append(cpkg.BuildDeps, record.Field.String())
				case stone1.Provides:
					cpkg.Provides = append(cpkg.Provides, record.Field.String())
				case stone1.Name:
					pkgName := record.Field.String()
					cpkg.Provides = append(cpkg.Provides, fmt.Sprintf("name(%s)", pkgName))
					// Implcitily assume that `X-dbginfo` is provieded by
					// package `X`.
					if !strings.HasPrefix(pkgName, "-dbginfo") {
						cpkg.Provides = append(cpkg.Provides, fmt.Sprintf("name(%s-dbginfo)", pkgName))
					}
				}
			default:
			}
		}

		if rdr.Err != nil {
			err = fmt.Errorf("stone reader failure: %w", rdr.Err)
			return
		}
	}

	// packageHeader, err := header.ReadHeader(io.NewSectionReader(file, 0, 32))
	// if err != nil {
	// 	err = fmt.Errorf("Failed to read package header: %w", err)
	// 	return
	// }
	//
	// var pos int64
	// pos += 32
	// for i := 0; i < int(packageHeader.Data.NumPayloads); i++ {
	// 	payloadheader, err := payload.ReadPayloadHeader(io.NewSectionReader(file, pos, 32))
	// 	if err != nil {
	// 		return cpkg, fmt.Errorf("Failed to read payload header: %w", err)
	// 	}
	//
	// 	pos += 32
	//
	// 	payloadReader, err := getCompressionReader(file, payloadheader.Compression, pos, int64(payloadheader.StoredSize))
	// 	if err != nil {
	// 		return cpkg, fmt.Errorf("Failed to get compression reader: %w", err)
	// 	}
	//
	// 	pos += int64(payloadheader.StoredSize)
	//
	// 	if payloadheader.Kind == payload.KindMeta {
	// 		// payload.PrintMetaPayload(payloadReader, int(payloadheader.NumRecords))
	//
	// 		bufferedReader := bufio.NewReader(payloadReader)
	// 		for j := 0; j < int(payloadheader.NumRecords); j++ {
	// 			record := payload.MetaRecord{}
	//
	// 			if err = binary.Read(bufferedReader, binary.BigEndian, &record); err != nil {
	// 				return cpkg, err
	// 			}
	//
	// 			data, err := payload.ReadRecordData(bufferedReader, record.RecordType)
	// 			if err != nil {
	// 				return cpkg, err
	// 			}
	//
	// 			if stringData, ok := data.(string); ok {
	// 				data = strings.TrimSuffix(stringData, "\x00")
	// 			}
	//
	// 			switch record.RecordTag {
	// 			case payload.RecordTagSourceID:
	// 				cpkg.Name = data.(string)
	// 			case payload.RecordTagVersion:
	// 				cpkg.Version = data.(string)
	// 			case payload.RecordTagRelease:
	// 				cpkg.Release = int(data.(uint64))
	// 			case payload.RecordTagDepends:
	// 				cpkg.BuildDeps = append(cpkg.BuildDeps, data.(string))
	// 			case payload.RecordTagProvides:
	// 				cpkg.Provides = append(cpkg.Provides, data.(string))
	// 			case payload.RecordTagName:
	// 				pkgName := data.(string)
	// 				cpkg.Provides = append(cpkg.Provides, pkgName)
	// 				// Implcitily assume that `X-dbginfo` is provieded by
	// 				// package `X`.
	// 				if !strings.HasPrefix(pkgName, "-dbginfo") {
	// 					cpkg.Provides = append(cpkg.Provides, pkgName+"-dbginfo")
	// 				}
	// 			}
	// 		}
	// 	} else {
	// 		fmt.Println("Warning: ", path, " has a payload that's not Meta!")
	// 	}
	// }

	return
}
